package main

import (
	"fmt"
	"strings"

	"github.com/gomarkdown/markdown"
	gomatrix "maunium.net/go/mautrix"
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
	app             *appContext
}

type UnverifiedUser struct {
	Verified bool
	User     *MatrixUser
}

type MatrixUser struct {
	RoomID  string
	UserID  string
	Lang    string
	Contact bool
}

type MatrixIdentifier struct {
	User      string `json:"user"`
	IdentType string `json:"type"`
}

func (m MatrixIdentifier) Type() string { return m.IdentType }

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
	}
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
	d.app.info.Println("Starting Matrix bot daemon")
	syncer := d.bot.Syncer.(*gomatrix.DefaultSyncer)
	syncer.OnEventType(event.NewEventType("m.room.message"), d.handleMessage)
	// syncer.OnEventType("m.room.member", d.handleMembership)
	if err := d.bot.Sync(); err != nil {
		d.app.err.Printf("Matrix sync failed: %v", err)
	}
}

func (d *MatrixDaemon) Shutdown() {
	d.bot.StopSync()
	d.Stopped = true
	close(d.ShutdownChannel)
}

func (d *MatrixDaemon) handleMessage(source gomatrix.EventSource, evt *event.Event) {
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

func (d *MatrixDaemon) CreateRoom(userID string) (id.RoomID, error) {
	room, err := d.bot.CreateRoom(&gomatrix.ReqCreateRoom{
		Visibility: "private",
		Invite:     []id.UserID{id.UserID(userID)},
		Topic:      d.app.config.Section("matrix").Key("topic").String(),
	})
	if err != nil {
		return "", err
	}
	return room.RoomID, nil
}

func (d *MatrixDaemon) SendStart(userID string) (ok bool) {
	roomID, err := d.CreateRoom(userID)
	if err != nil {
		d.app.err.Printf("Failed to create room for user \"%s\": %v", userID, err)
		return
	}
	lang := "en-us"
	pin := genAuthToken()
	d.tokens[pin] = UnverifiedUser{
		false,
		&MatrixUser{
			RoomID: string(roomID),
			UserID: userID,
			Lang:   lang,
		},
	}
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

func (d *MatrixDaemon) Send(message *Message, roomID ...string) (err error) {
	md := ""
	if message.Markdown != "" {
		// Convert images to links
		md = string(markdown.ToHTML([]byte(strings.ReplaceAll(message.Markdown, "![", "[")), nil, renderer))
	}
	for _, ident := range roomID {

		if md != "" {
			_, err = d.bot.SendMessageEvent(id.RoomID(ident), event.NewEventType("m.room.message"), map[string]interface{}{
				"msgtype":        "m.text",
				"body":           message.Text,
				"formatted_body": md,
				"format":         "org.matrix.custom.html",
			}, gomatrix.ReqSendEvent{})
		} else {
			_, err = d.bot.SendText(id.RoomID(ident), message.Text)
		}
		if err != nil {
			return
		}
	}
	return
}

// User enters ID on sign-up, a PIN is sent to them. They enter it on sign-up.

// Message the user first, to avoid E2EE by default
