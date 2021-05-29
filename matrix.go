package main

import (
	"encoding/json"

	"github.com/matrix-org/gomatrix"
)

type MatrixDaemon struct {
	Stopped         bool
	ShutdownChannel chan string
	bot             *gomatrix.Client
	userID          string
	tokens          map[string]UnverifiedUser // Map of tokens to users
	languages       map[string]string         // Map of roomIDs to language codes
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

var matrixFilter = gomatrix.Filter{
	Room: gomatrix.RoomFilter{
		Timeline: gomatrix.FilterPart{
			Types: []string{
				"m.room.message",
				"m.room.member",
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
		userID:          matrix.Key("user_id").String(),
		tokens:          map[string]UnverifiedUser{},
		languages:       map[string]string{},
		app:             app,
	}
	d.bot, err = gomatrix.NewClient(homeserver, d.userID, token)
	if err != nil {
		return
	}
	filter, err := json.Marshal(matrixFilter)
	if err != nil {
		return
	}
	resp, err := d.bot.CreateFilter(filter)
	d.bot.Store.SaveFilterID(d.userID, resp.FilterID)
	for _, user := range app.storage.matrix {
		if user.Lang != "" {
			d.languages[user.RoomID] = user.Lang
		}
	}
	return
}

func (d *MatrixDaemon) run() {
	d.app.info.Println("Starting Matrix bot daemon")
	syncer := d.bot.Syncer.(*gomatrix.DefaultSyncer)
	syncer.OnEventType("m.room.message", d.handleMessage)
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

func (d *MatrixDaemon) handleMessage(event *gomatrix.Event) { return }

func (d *MatrixDaemon) SendStart(userID string) (ok bool) {
	room, err := d.bot.CreateRoom(&gomatrix.ReqCreateRoom{
		Visibility: "private",
		Invite:     []string{userID},
		Topic:      "jfa-go",
	})
	if err != nil {
		d.app.err.Printf("Failed to create room for user \"%s\": %v", userID, err)
		return
	}
	lang := "en-us"
	pin := genAuthToken()
	d.tokens[pin] = UnverifiedUser{
		false,
		&MatrixUser{
			RoomID: room.RoomID,
			UserID: userID,
			Lang:   lang,
		},
	}
	_, err = d.bot.SendText(
		room.RoomID,
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

// User enters ID on sign-up, a PIN is sent to them. They enter it on sign-up.

// Message the user first, to avoid E2EE by default
