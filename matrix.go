package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/timshannon/badgerhold/v4"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type MatrixDaemon struct {
	Stopped         bool
	ShutdownChannel chan string
	bot             *mautrix.Client
	userID          id.UserID
	tokens          map[string]UnverifiedUser // Map of tokens to users
	languages       map[id.RoomID]string      // Map of roomIDs to language codes
	Encryption      bool
	isEncrypted     map[id.RoomID]bool
	crypto          Crypto
	app             *appContext
	start           int64
}

type UnverifiedUser struct {
	Verified bool
	User     *MatrixUser
}

var matrixFilter = mautrix.Filter{
	Room: mautrix.RoomFilter{
		Timeline: mautrix.FilterPart{
			Types: []event.Type{
				event.EventMessage,
				event.EventEncrypted,
				event.StateMember,
			},
		},
	},
	EventFields: []string{
		"type",
		"event_id",
		"room_id",
		"state_key",
		"sender",
		"content",
		"timestamp",
		// "content.body",
		// "content.membership",
	},
}

func newMatrixDaemon(app *appContext) (d *MatrixDaemon, err error) {
	matrix := app.config.Section("matrix")
	homeserver := matrix.Key("homeserver").String()
	token := matrix.Key("token").String()
	d = &MatrixDaemon{
		ShutdownChannel: make(chan string),
		userID:          id.UserID(matrix.Key("user_id").String()),
		tokens:          map[string]UnverifiedUser{},
		languages:       map[id.RoomID]string{},
		isEncrypted:     map[id.RoomID]bool{},
		app:             app,
		start:           time.Now().UnixNano() / 1e6,
	}
	d.bot, err = mautrix.NewClient(homeserver, d.userID, token)
	if err != nil {
		return
	}
	// resp, err := d.bot.CreateFilter(&matrixFilter)
	// if err != nil {
	// 	return
	// }
	// d.bot.Store.SaveFilterID(d.userID, resp.FilterID)
	for _, user := range app.storage.GetMatrix() {
		if user.Lang != "" {
			d.languages[id.RoomID(user.RoomID)] = user.Lang
		}
		d.isEncrypted[id.RoomID(user.RoomID)] = user.Encrypted
	}
	err = InitMatrixCrypto(d)
	return
}

// SetTransport sets the http.Transport to use for requests. Can be used to set a proxy.
func (d *MatrixDaemon) SetTransport(t *http.Transport) {
	d.bot.Client.Transport = t
}

func (d *MatrixDaemon) generateAccessToken(homeserver, username, password string) (string, error) {
	req := &mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: username,
		},
		Password: password,
		DeviceID: id.DeviceID("jfa-go-" + commit),
	}
	bot, err := mautrix.NewClient(homeserver, id.UserID(username), "")
	if err != nil {
		return "", err
	}
	resp, err := bot.Login(context.TODO(), req)
	if err != nil {
		return "", err
	}
	return resp.AccessToken, nil
}

func (d *MatrixDaemon) run() {
	startTime := d.start
	d.app.info.Println(lm.StartDaemon, lm.Matrix)
	syncer := d.bot.Syncer.(*mautrix.DefaultSyncer)
	HandleSyncerCrypto(startTime, d, syncer)
	syncer.OnEventType(event.EventMessage, d.handleMessage)

	if err := d.bot.Sync(); err != nil {
		d.app.err.Printf(lm.FailedSyncMatrix, err)
	}
}

func (d *MatrixDaemon) Shutdown() {
	CryptoShutdown(d)
	d.bot.StopSync()
	d.Stopped = true
	close(d.ShutdownChannel)
}

func (d *MatrixDaemon) handleMessage(ctx context.Context, evt *event.Event) {
	if evt.Timestamp < d.start {
		return
	}
	if evt.Sender == d.userID {
		return
	}
	lang := "en-us"
	if l, ok := d.languages[evt.RoomID]; ok {
		if _, ok := d.app.storage.lang.Telegram[l]; ok {
			lang = l
		}
	}
	sects := strings.Split(evt.Content.Raw["body"].(string), " ")
	switch sects[0] {
	case "!lang":
		if len(sects) == 2 {
			d.commandLang(evt, sects[1], lang)
		} else {
			d.commandLang(evt, "", lang)
		}
	}
}

func (d *MatrixDaemon) commandLang(evt *event.Event, code, lang string) {
	if code == "" {
		list := "!lang <lang>\n"
		for c := range d.app.storage.lang.Telegram {
			list += fmt.Sprintf("%s: %s\n", c, d.app.storage.lang.Telegram[c].Meta.Name)
		}
		_, err := d.bot.SendText(
			context.TODO(),
			evt.RoomID,
			list,
		)
		if err != nil {
			d.app.err.Printf(lm.FailedReply, lm.Matrix, evt.Sender, err)
		}
		return
	}
	if _, ok := d.app.storage.lang.Telegram[code]; !ok {
		return
	}
	d.languages[evt.RoomID] = code
	if u, ok := d.app.storage.GetMatrixKey(string(evt.RoomID)); ok {
		u.Lang = code
		d.app.storage.SetMatrixKey(string(evt.RoomID), u)
	}
}

func (d *MatrixDaemon) CreateRoom(userID string) (roomID id.RoomID, encrypted bool, err error) {
	var room *mautrix.RespCreateRoom
	room, err = d.bot.CreateRoom(context.TODO(), &mautrix.ReqCreateRoom{
		Visibility: "private",
		Invite:     []id.UserID{id.UserID(userID)},
		Topic:      d.app.config.Section("matrix").Key("topic").String(),
		IsDirect:   true,
	})
	if err != nil {
		return
	}
	encrypted = EncryptRoom(d, room, id.UserID(userID))
	roomID = room.RoomID
	return
}

func (d *MatrixDaemon) SendStart(userID string) (ok bool) {
	roomID, encrypted, err := d.CreateRoom(userID)
	if err != nil {
		d.app.err.Printf(lm.FailedCreateMatrixRoom, userID, err)
		return
	}
	lang := "en-us"
	pin := genAuthToken()
	d.tokens[pin] = UnverifiedUser{
		false,
		&MatrixUser{
			RoomID:    string(roomID),
			UserID:    userID,
			Lang:      lang,
			Encrypted: encrypted,
		},
	}
	err = d.sendToRoom(
		&event.MessageEventContent{
			MsgType: event.MsgText,
			Body: d.app.storage.lang.Telegram[lang].Strings.get("matrixStartMessage") + "\n\n" + pin + "\n\n" +
				d.app.storage.lang.Telegram[lang].Strings.template("languageMessage", tmpl{"command": "!lang"}),
		},
		roomID,
	)
	if err != nil {
		d.app.err.Printf(lm.FailedMessage, lm.Matrix, userID, err)
		return
	}
	ok = true
	return
}

func (d *MatrixDaemon) sendToRoom(content *event.MessageEventContent, roomID id.RoomID) (err error) {
	if encrypted, ok := d.isEncrypted[roomID]; ok && encrypted {
		err = SendEncrypted(d, content, roomID)
	} else {
		_, err = d.bot.SendMessageEvent(context.TODO(), roomID, event.EventMessage, content, mautrix.ReqSendEvent{})
	}
	return
}

func (d *MatrixDaemon) send(content *event.MessageEventContent, roomID id.RoomID) (err error) {
	_, err = d.bot.SendMessageEvent(context.TODO(), roomID, event.EventMessage, content, mautrix.ReqSendEvent{})
	return
}

func (d *MatrixDaemon) Send(message *Message, users ...MatrixUser) (err error) {
	md := ""
	if message.Markdown != "" {
		// Convert images to links
		md = string(markdown.ToHTML([]byte(strings.ReplaceAll(message.Markdown, "![", "[")), nil, markdownRenderer))
	}
	content := &event.MessageEventContent{
		MsgType: "m.text",
		Body:    message.Text,
	}
	if md != "" {
		content.FormattedBody = md
		content.Format = "org.matrix.custom.html"
	}
	for _, user := range users {
		err = d.sendToRoom(content, id.RoomID(user.RoomID))
		if err != nil {
			return
		}
	}
	return
}

// UserExists returns whether or not a user with the given User ID exists.
func (d *MatrixDaemon) UserExists(userID string) bool {
	c, err := d.app.storage.db.Count(&MatrixUser{}, badgerhold.Where("UserID").Eq(userID))
	return err != nil || c > 0
}

// Exists returns whether or not the given user exists.
func (d *MatrixDaemon) Exists(user ContactMethodUser) bool {
	return d.UserExists(user.Name())
}

// User enters ID on sign-up, a PIN is sent to them. They enter it on sign-up.

// Message the user first, to avoid E2EE by default

func (d *MatrixDaemon) PIN(req newUserDTO) string { return req.MatrixPIN }

func (d *MatrixDaemon) Name() string { return lm.Matrix }

func (d *MatrixDaemon) Required() bool {
	return d.app.config.Section("telegram").Key("required").MustBool(false)
}

func (d *MatrixDaemon) UniqueRequired() bool {
	return d.app.config.Section("telegram").Key("require_unique").MustBool(false)
}

// TokenVerified returns whether or not a token with the given PIN has been verified, and the token itself.
func (d *MatrixDaemon) TokenVerified(pin string) (token UnverifiedUser, ok bool) {
	token, ok = d.tokens[pin]
	// delete(t.verifiedTokens, pin)
	return
}

// DeleteVerifiedToken removes the token with the given PIN.
func (d *MatrixDaemon) DeleteVerifiedToken(PIN string) {
	delete(d.tokens, PIN)
}

func (d *MatrixDaemon) UserVerified(PIN string) (ContactMethodUser, bool) {
	token, ok := d.TokenVerified(PIN)
	if !ok {
		return &MatrixUser{}, false
	}
	return token.User, ok
}

func (d *MatrixDaemon) PostVerificationTasks(string, ContactMethodUser) error { return nil }

func (m *MatrixUser) Name() string                          { return m.UserID }
func (m *MatrixUser) SetMethodID(id any)                    { m.UserID = id.(string) }
func (m *MatrixUser) MethodID() any                         { return m.UserID }
func (m *MatrixUser) SetJellyfin(id string)                 { m.JellyfinID = id }
func (m *MatrixUser) Jellyfin() string                      { return m.JellyfinID }
func (m *MatrixUser) SetAllowContactFromDTO(req newUserDTO) { m.Contact = req.MatrixContact }
func (m *MatrixUser) SetAllowContact(contact bool)          { m.Contact = contact }
func (m *MatrixUser) AllowContact() bool                    { return m.Contact }
func (m *MatrixUser) Store(st *Storage) {
	st.SetMatrixKey(m.Jellyfin(), *m)
}
