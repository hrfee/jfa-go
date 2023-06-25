package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/timshannon/badgerhold/v4"
)

const (
	VERIF_TOKEN_EXPIRY_SEC = 10 * 60
)

type TelegramVerifiedToken struct {
	ChatID     int64
	Username   string
	JellyfinID string // optional, for ensuring a user-requested change is only accessed by them.
}

// VerifToken stores details about a pending user verification token.
type VerifToken struct {
	Expiry     time.Time
	JellyfinID string // optional, for ensuring a user-requested change is only accessed by them.
}

type TelegramDaemon struct {
	Stopped         bool
	ShutdownChannel chan string
	bot             *tg.BotAPI
	username        string
	tokens          map[string]VerifToken            // Map of pins to tokens.
	verifiedTokens  map[string]TelegramVerifiedToken // Map of token pins to the responsible ChatID+Username.
	languages       map[int64]string                 // Store of languages for chatIDs. Added to on first interaction, and loaded from app.storage.telegram on start.
	link            string
	app             *appContext
}

func newTelegramDaemon(app *appContext) (*TelegramDaemon, error) {
	token := app.config.Section("telegram").Key("token").String()
	if token == "" {
		return nil, fmt.Errorf("token was blank")
	}
	bot, err := tg.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	td := &TelegramDaemon{
		ShutdownChannel: make(chan string),
		bot:             bot,
		username:        bot.Self.UserName,
		tokens:          map[string]VerifToken{},
		verifiedTokens:  map[string]TelegramVerifiedToken{},
		languages:       map[int64]string{},
		link:            "https://t.me/" + bot.Self.UserName,
		app:             app,
	}
	for _, user := range app.storage.GetTelegram() {
		if user.Lang != "" {
			td.languages[user.ChatID] = user.Lang
		}
	}
	return td, nil
}

func genAuthToken() string {
	rand.Seed(time.Now().UnixNano())
	pin := make([]rune, 8)
	for i := range pin {
		if (i+1)%3 == 0 {
			pin[i] = '-'
		} else {
			pin[i] = runes[rand.Intn(len(runes))]
		}
	}
	return string(pin)
}

var runes = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// NewAuthToken generates an 8-character pin in the form "A1-2B-CD".
func (t *TelegramDaemon) NewAuthToken() string {
	pin := genAuthToken()
	t.tokens[pin] = VerifToken{Expiry: time.Now().Add(VERIF_TOKEN_EXPIRY_SEC * time.Second), JellyfinID: ""}
	return pin
}

// NewAssignedAuthToken generates an 8-character pin in the form "A1-2B-CD",
// and assigns it for access only with the given Jellyfin ID.
func (t *TelegramDaemon) NewAssignedAuthToken(id string) string {
	pin := genAuthToken()
	t.tokens[pin] = VerifToken{Expiry: time.Now().Add(VERIF_TOKEN_EXPIRY_SEC * time.Second), JellyfinID: id}
	return pin
}

func (t *TelegramDaemon) run() {
	t.app.info.Println("Starting Telegram bot daemon")
	u := tg.NewUpdate(0)
	u.Timeout = 60
	updates, err := t.bot.GetUpdatesChan(u)
	if err != nil {
		t.app.err.Printf("Failed to start Telegram daemon: %v", err)
		telegramEnabled = false
		return
	}
	for {
		var upd tg.Update
		select {
		case upd = <-updates:
			if upd.Message == nil {
				continue
			}
			sects := strings.Split(upd.Message.Text, " ")
			if len(sects) == 0 {
				continue
			}
			lang := t.app.storage.lang.chosenTelegramLang
			storedLang, ok := t.languages[upd.Message.Chat.ID]
			if !ok {
				found := false
				for code := range t.app.storage.lang.Telegram {
					if code[:2] == upd.Message.From.LanguageCode {
						lang = code
						found = true
						break
					}
				}
				if found {
					t.languages[upd.Message.Chat.ID] = lang
				}
			} else {
				lang = storedLang
			}
			switch msg := sects[0]; msg {
			case "/start":
				t.commandStart(&upd, sects, lang)
				continue
			case "/lang":
				t.commandLang(&upd, sects, lang)
				continue
			default:
				t.commandPIN(&upd, sects, lang)
			}

		case <-t.ShutdownChannel:
			t.bot.StopReceivingUpdates()
			t.ShutdownChannel <- "Down"
			return
		}
	}
}

func (t *TelegramDaemon) Reply(upd *tg.Update, content string) error {
	msg := tg.NewMessage((*upd).Message.Chat.ID, content)
	_, err := t.bot.Send(msg)
	return err
}

func (t *TelegramDaemon) QuoteReply(upd *tg.Update, content string) error {
	msg := tg.NewMessage((*upd).Message.Chat.ID, content)
	msg.ReplyToMessageID = (*upd).Message.MessageID
	_, err := t.bot.Send(msg)
	return err
}

var escapedChars = []string{"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-", "=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!"}
var escaper = strings.NewReplacer(escapedChars...)

// Send will send a telegram message to a list of chat IDs. message.text is used if no markdown is given.
func (t *TelegramDaemon) Send(message *Message, ID ...int64) error {
	for _, id := range ID {
		var msg tg.MessageConfig
		if message.Markdown == "" {
			msg = tg.NewMessage(id, message.Text)
		} else {
			text := escaper.Replace(message.Markdown)
			msg = tg.NewMessage(id, text)
			msg.ParseMode = "MarkdownV2"
		}
		_, err := t.bot.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *TelegramDaemon) Shutdown() {
	t.Stopped = true
	t.ShutdownChannel <- "Down"
	<-t.ShutdownChannel
	close(t.ShutdownChannel)
}

func (t *TelegramDaemon) commandStart(upd *tg.Update, sects []string, lang string) {
	content := t.app.storage.lang.Telegram[lang].Strings.get("startMessage") + "\n"
	content += t.app.storage.lang.Telegram[lang].Strings.template("languageMessage", tmpl{"command": "/lang"})
	err := t.Reply(upd, content)
	if err != nil {
		t.app.err.Printf("Telegram: Failed to send message to \"%s\": %v", upd.Message.From.UserName, err)
	}
}

func (t *TelegramDaemon) commandLang(upd *tg.Update, sects []string, lang string) {
	if len(sects) == 1 {
		list := "/lang `<lang>`\n"
		for code := range t.app.storage.lang.Telegram {
			list += fmt.Sprintf("`%s`: %s\n", code, t.app.storage.lang.Telegram[code].Meta.Name)
		}
		err := t.Reply(upd, list)
		if err != nil {
			t.app.err.Printf("Telegram: Failed to send message to \"%s\": %v", upd.Message.From.UserName, err)
		}
		return
	}
	if _, ok := t.app.storage.lang.Telegram[sects[1]]; ok {
		t.languages[upd.Message.Chat.ID] = sects[1]
		for _, user := range t.app.storage.GetTelegram() {
			if user.ChatID == upd.Message.Chat.ID {
				user.Lang = sects[1]
				t.app.storage.SetTelegramKey(user.JellyfinID, user)
				break
			}
		}
	}
}

func (t *TelegramDaemon) commandPIN(upd *tg.Update, sects []string, lang string) {
	token, ok := t.tokens[upd.Message.Text]
	if !ok || time.Now().After(token.Expiry) {
		err := t.QuoteReply(upd, t.app.storage.lang.Telegram[lang].Strings.get("invalidPIN"))
		if err != nil {
			t.app.err.Printf("Telegram: Failed to send message to \"%s\": %v", upd.Message.From.UserName, err)
		}
		delete(t.tokens, upd.Message.Text)
		return
	}
	err := t.QuoteReply(upd, t.app.storage.lang.Telegram[lang].Strings.get("pinSuccess"))
	if err != nil {
		t.app.err.Printf("Telegram: Failed to send message to \"%s\": %v", upd.Message.From.UserName, err)
	}
	t.verifiedTokens[upd.Message.Text] = TelegramVerifiedToken{
		ChatID:     upd.Message.Chat.ID,
		Username:   upd.Message.Chat.UserName,
		JellyfinID: token.JellyfinID,
	}
	delete(t.tokens, upd.Message.Text)
}

// TokenVerified returns whether or not a token with the given PIN has been verified, and the token itself.
func (t *TelegramDaemon) TokenVerified(pin string) (token TelegramVerifiedToken, ok bool) {
	token, ok = t.verifiedTokens[pin]
	// delete(t.verifiedTokens, pin)
	return
}

// AssignedTokenVerified returns whether or not a token with the given PIN has been verified, and the token itself.
// Returns false if the given Jellyfin ID does not match the one in the token.
func (t *TelegramDaemon) AssignedTokenVerified(pin string, jfID string) (token TelegramVerifiedToken, ok bool) {
	token, ok = t.verifiedTokens[pin]
	if ok && token.JellyfinID != jfID {
		ok = false
	}
	// delete(t.verifiedTokens, pin)
	return
}

// UserExists returns whether or not a user with the given username exists.
func (t *TelegramDaemon) UserExists(username string) bool {
	c, err := t.app.storage.db.Count(&TelegramUser{}, badgerhold.Where("Username").Eq(username))
	return err != nil || c > 0
}

// DeleteVerifiedToken removes the token with the given PIN.
func (t *TelegramDaemon) DeleteVerifiedToken(pin string) {
	delete(t.verifiedTokens, pin)
}
