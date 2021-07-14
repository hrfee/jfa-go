package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"database/sql"

	"github.com/gomarkdown/markdown"
	_ "github.com/mattn/go-sqlite3"
	gomatrix "maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type MatrixDaemon struct {
	Stopped         bool
	ShutdownChannel chan string
	bot             *gomatrix.Client
	userID          id.UserID
	tokens          map[string]UnverifiedUser // Map of tokens to users
	languages       map[id.RoomID]string      // Map of roomIDs to language codes
	isEncrypted     map[id.RoomID]bool
	cryptoStore     *crypto.GobStore
	olm             *crypto.OlmMachine
	db              *sql.DB
	app             *appContext
	start           int64
}

type UnverifiedUser struct {
	Verified bool
	User     *MatrixUser
}

type MatrixUser struct {
	RoomID    string
	Encrypted bool
	UserID    string
	Lang      string
	Contact   bool
}

func (m *MatrixDaemon) IsEncrypted(roomID id.RoomID) bool {
	encrypted, ok := m.isEncrypted[roomID]
	return ok && encrypted
}

func (m *MatrixDaemon) GetEncryptionEvent(roomID id.RoomID) *event.EncryptionEventContent {
	return &event.EncryptionEventContent{
		Algorithm:              id.AlgorithmMegolmV1,
		RotationPeriodMillis:   7 * 24 * 60 * 60 * 1000,
		RotationPeriodMessages: 100,
	}
}

// Users are assumed to only have one common channel with the bot, so we can stub this out.
func (m *MatrixDaemon) FindSharedRooms(userID id.UserID) []id.RoomID {
	for _, user := range m.app.storage.matrix {
		if id.UserID(user.UserID) == userID {
			return []id.RoomID{id.RoomID(user.RoomID)}
		}
	}
	return []id.RoomID{}
}

type olmLogger struct {
	app *appContext
}

func (o olmLogger) Error(message string, args ...interface{}) {
	o.app.err.Printf("OLM: "+message+"\n", args)
}

func (o olmLogger) Warn(message string, args ...interface{}) {
	o.app.info.Printf("OLM: "+message+"\n", args)
}

func (o olmLogger) Debug(message string, args ...interface{}) {
	o.app.debug.Printf("OLM: "+message+"\n", args)
}

func (o olmLogger) Trace(message string, args ...interface{}) {
	o.app.debug.Printf("OLM [TRACE]: "+message+"\n", args)
}

var matrixFilter = gomatrix.Filter{
	Room: gomatrix.RoomFilter{
		Timeline: gomatrix.FilterPart{
			Types: []event.Type{
				event.NewEventType("m.room.message"),
				event.NewEventType("m.room.member"),
			},
		},
	},
	EventFields: []string{
		"type",
		"event_id",
		"room_id",
		"state_key",
		"sender",
		"content.body",
		"content.membership",
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
	}
	d.bot, err = gomatrix.NewClient(homeserver, d.userID, token)
	if err != nil {
		return
	}
	resp, err := d.bot.CreateFilter(&matrixFilter)
	d.bot.Store.SaveFilterID(d.userID, resp.FilterID)
	for _, user := range app.storage.matrix {
		if user.Lang != "" {
			d.languages[id.RoomID(user.RoomID)] = user.Lang
		}
		d.isEncrypted[id.RoomID(user.RoomID)] = user.Encrypted
	}
	dbPath := app.config.Section("files").Key("matrix_sql").String()
	// If the db is maintained after restart, element reports "The secure channel with the sender was corrupted" when sending a message from the bot.
	// This obviously isn't right, but it seems to work.
	// Since its not really used anyway, just use the deprecated GobStore. This reduces cgo usage anyway.
	os.Remove(dbPath)
	cryptoStore, err := crypto.NewGobStore(dbPath)
	// d.db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return
	}
	olmLog := &olmLogger{app}
	// deviceID := "jfa-go" + commit
	// cryptoStore := crypto.NewSQLCryptoStore(d.db, "sqlite3", string(d.userID)+deviceID, id.DeviceID(deviceID), []byte("jfa-go"), olmLog)
	// err = cryptoStore.CreateTables()
	// if err != nil {
	// 	return
	// }
	d.olm = crypto.NewOlmMachine(d.bot, olmLog, cryptoStore, crypto.StateStore(d))
	// d.olm.AllowUnverifiedDevices = true
	err = d.olm.Load()
	if err != nil {
		return
	}
	d.cryptoStore = cryptoStore
	return
}

func (d *MatrixDaemon) generateAccessToken(homeserver, username, password string) (string, error) {
	req := &gomatrix.ReqLogin{
		Type: "m.login.password",
		Identifier: gomatrix.UserIdentifier{
			User: username,
			Type: "m.id.user",
		},
		Password: password,
		DeviceID: id.DeviceID("jfa-go-" + commit),
	}
	bot, err := gomatrix.NewClient(homeserver, id.UserID(username), "")
	if err != nil {
		return "", err
	}
	resp, err := bot.Login(req)
	if err != nil {
		return "", err
	}
	return resp.AccessToken, nil
}

func (d *MatrixDaemon) run() {
	d.start = time.Now().UnixNano() / 1000000
	d.app.info.Println("Starting Matrix bot daemon")
	syncer := d.bot.Syncer.(*gomatrix.DefaultSyncer)
	syncer.OnEventType(event.EventMessage, d.handleMessage)
	syncer.OnEventType(event.EventEncrypted, func(source gomatrix.EventSource, evt *event.Event) {
		if evt.Timestamp < d.start {
			return
		}
		decrypted, err := d.olm.DecryptMegolmEvent(evt)
		if err != nil {
			d.app.err.Printf("Failed to decrypt Matrix message: %v", err)
			return
		}
		d.handleMessage(source, decrypted)
	})
	syncer.OnSync(d.olm.ProcessSyncResponse)
	syncer.OnEventType(event.StateMember, func(source gomatrix.EventSource, evt *event.Event) {
		d.olm.HandleMemberEvent(evt)
	})

	// syncer.OnEventType("m.room.member", d.handleMembership)
	if err := d.bot.Sync(); err != nil {
		d.app.err.Printf("Matrix sync failed: %v", err)
	}
}

func (d *MatrixDaemon) Shutdown() {
	d.olm.FlushStore()
	d.bot.StopSync()
	d.db.Close()
	d.Stopped = true
	close(d.ShutdownChannel)
}

func (d *MatrixDaemon) handleMessage(source gomatrix.EventSource, evt *event.Event) {
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
			evt.RoomID,
			list,
		)
		if err != nil {
			d.app.err.Printf("Matrix: Failed to send message to \"%s\": %v", evt.Sender, err)
		}
		return
	}
	if _, ok := d.app.storage.lang.Telegram[code]; !ok {
		return
	}
	d.languages[evt.RoomID] = code
	if u, ok := d.app.storage.matrix[string(evt.RoomID)]; ok {
		u.Lang = code
		d.app.storage.matrix[string(evt.RoomID)] = u
		if err := d.app.storage.storeMatrixUsers(); err != nil {
			d.app.err.Printf("Matrix: Failed to store Matrix users: %v", err)
		}
	}
}

func (d *MatrixDaemon) CreateRoom(userID string) (roomID id.RoomID, encrypted bool, err error) {
	var room *gomatrix.RespCreateRoom
	room, err = d.bot.CreateRoom(&gomatrix.ReqCreateRoom{
		Visibility: "private",
		Invite:     []id.UserID{id.UserID(userID)},
		Topic:      d.app.config.Section("matrix").Key("topic").String(),
	})
	if err != nil {
		return
	}
	_, err = d.bot.SendStateEvent(room.RoomID, event.StateEncryption, "", &event.EncryptionEventContent{
		Algorithm:              id.AlgorithmMegolmV1,
		RotationPeriodMillis:   7 * 24 * 60 * 60 * 1000,
		RotationPeriodMessages: 100,
	})
	if err == nil {
		encrypted = true
	} else {
		d.app.debug.Printf("Matrix: Failed to enable encryption in room: %v", err)
	}
	roomID = room.RoomID
	return
}

func (d *MatrixDaemon) SendStart(userID string) (ok bool) {
	roomID, encrypted, err := d.CreateRoom(userID)
	if err != nil {
		d.app.err.Printf("Failed to create room for user \"%s\": %v", userID, err)
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
	d.isEncrypted[roomID] = encrypted
	_, err = d.bot.SendText(
		roomID,
		d.app.storage.lang.Telegram[lang].Strings.get("matrixStartMessage")+"\n\n"+pin+"\n\n"+
			d.app.storage.lang.Telegram[lang].Strings.template("languageMessage", tmpl{"command": "!lang"}),
	)
	if err != nil {
		d.app.err.Printf("Matrix: Failed to send welcome message to \"%s\": %v", userID, err)
		return
	}
	ok = true
	return
}

func (d *MatrixDaemon) Send(message *Message, users ...MatrixUser) (err error) {
	md := ""
	if message.Markdown != "" {
		// Convert images to links
		md = string(markdown.ToHTML([]byte(strings.ReplaceAll(message.Markdown, "![", "[")), nil, renderer))
	}
	content := event.MessageEventContent{
		MsgType: "m.text",
		Body:    message.Text,
	}
	if md != "" {
		content.FormattedBody = md
		content.Format = "org.matrix.custom.html"
	}
	for _, user := range users {
		if user.Encrypted {
			err = d.SendEncrypted(&content, user)
		} else {
			_, err = d.bot.SendMessageEvent(id.RoomID(user.RoomID), event.NewEventType("m.room.message"), content, gomatrix.ReqSendEvent{})
		}
		if err != nil {
			return
		}
	}
	return
}

func (d *MatrixDaemon) SendEncrypted(content *event.MessageEventContent, users ...MatrixUser) (err error) {
	for _, user := range users {
		var encrypted *event.EncryptedEventContent
		encrypted, err = d.olm.EncryptMegolmEvent(id.RoomID(user.RoomID), event.EventMessage, content)
		if err == crypto.SessionExpired || err == crypto.SessionNotShared || err == crypto.NoGroupSession {
			err = d.olm.ShareGroupSession(id.RoomID(user.RoomID), []id.UserID{id.UserID(user.UserID), d.userID})
			if err != nil {
				return
			}
			encrypted, err = d.olm.EncryptMegolmEvent(id.RoomID(user.RoomID), event.EventMessage, content)
		}
		if err != nil {
			return
		}
		_, err = d.bot.SendMessageEvent(id.RoomID(user.RoomID), event.EventEncrypted, &event.Content{Parsed: encrypted})
		if err != nil {
			return
		}
	}
	return
}

// User enters ID on sign-up, a PIN is sent to them. They enter it on sign-up.

// Message the user first, to avoid E2EE by default
