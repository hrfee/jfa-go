package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

type TelegramVerifiedToken struct {
	Token    string
	ChatID   int64
	Username string
}

type TelegramDaemon struct {
	Stopped         bool
	ShutdownChannel chan string
	bot             *tg.BotAPI
	username        string
	tokens          []string
	verifiedTokens  []TelegramVerifiedToken
	languages       map[int64]string // Store of languages for chatIDs. Added to on first interaction, and loaded from app.storage.telegram on start.
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
		tokens:          []string{},
		verifiedTokens:  []TelegramVerifiedToken{},
		languages:       map[int64]string{},
		link:            "https://t.me/" + bot.Self.UserName,
		app:             app,
	}
	for _, user := range app.storage.telegram {
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
	t.tokens = append(t.tokens, pin)
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
		for jfID, user := range t.app.storage.telegram {
			if user.ChatID == upd.Message.Chat.ID {
				user.Lang = sects[1]
				t.app.storage.telegram[jfID] = user
				if err := t.app.storage.storeTelegramUsers(); err != nil {
					t.app.err.Printf("Failed to store Telegram users: %v", err)
				}
				break
			}
		}
	}
}

func (t *TelegramDaemon) commandPIN(upd *tg.Update, sects []string, lang string) {
	tokenIndex := -1
	for i, token := range t.tokens {
		if upd.Message.Text == token {
			tokenIndex = i
			break
		}
	}
	if tokenIndex == -1 {
		err := t.QuoteReply(upd, t.app.storage.lang.Telegram[lang].Strings.get("invalidPIN"))
		if err != nil {
			t.app.err.Printf("Telegram: Failed to send message to \"%s\": %v", upd.Message.From.UserName, err)
		}
		return
	}
	err := t.QuoteReply(upd, t.app.storage.lang.Telegram[lang].Strings.get("pinSuccess"))
	if err != nil {
		t.app.err.Printf("Telegram: Failed to send message to \"%s\": %v", upd.Message.From.UserName, err)
	}
	t.verifiedTokens = append(t.verifiedTokens, TelegramVerifiedToken{
		Token:    upd.Message.Text,
		ChatID:   upd.Message.Chat.ID,
		Username: upd.Message.Chat.UserName,
	})
	t.tokens[len(t.tokens)-1], t.tokens[tokenIndex] = t.tokens[tokenIndex], t.tokens[len(t.tokens)-1]
	t.tokens = t.tokens[:len(t.tokens)-1]
}
