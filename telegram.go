package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

type VerifiedToken struct {
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
	verifiedTokens  []VerifiedToken
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
		Stopped:         false,
		ShutdownChannel: make(chan string),
		bot:             bot,
		username:        bot.Self.UserName,
		tokens:          []string{},
		verifiedTokens:  []VerifiedToken{},
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

var runes = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// NewAuthToken generates an 8-character pin in the form "A1-2B-CD".
func (t *TelegramDaemon) NewAuthToken() string {
	rand.Seed(time.Now().UnixNano())
	pin := make([]rune, 8)
	for i := range pin {
		if i == 2 || i == 5 {
			pin[i] = '-'
		} else {
			pin[i] = runes[rand.Intn(len(runes))]
		}
	}
	t.tokens = append(t.tokens, string(pin))
	return string(pin)
}

func (t *TelegramDaemon) run() {
	t.app.info.Println("Starting Telegram bot daemon")
	u := tg.NewUpdate(0)
	u.Timeout = 60
	updates, err := t.bot.GetUpdatesChan(u)
	if err != nil {
		t.app.err.Printf("Failed to start Telegram daemon: %v", err)
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
				content := t.app.storage.lang.Telegram[lang].Strings.get("startMessage") + "\n"
				content += t.app.storage.lang.Telegram[lang].Strings.get("languageMessage")
				err := t.Reply(&upd, content)
				if err != nil {
					t.app.err.Printf("Telegram: Failed to send message to \"%s\": %v", upd.Message.From.UserName, err)
				}
				continue
			case "/lang":
				if len(sects) == 1 {
					list := "/lang <lang>\n"
					for code := range t.app.storage.lang.Telegram {
						list += fmt.Sprintf("%s: %s\n", code, t.app.storage.lang.Telegram[code].Meta.Name)
					}
					err := t.Reply(&upd, list)
					if err != nil {
						t.app.err.Printf("Telegram: Failed to send message to \"%s\": %v", upd.Message.From.UserName, err)
					}
					continue
				}
				if _, ok := t.app.storage.lang.Telegram[sects[1]]; ok {
					t.languages[upd.Message.Chat.ID] = sects[1]
					for jfID, user := range t.app.storage.telegram {
						if user.ChatID == upd.Message.Chat.ID {
							user.Lang = sects[1]
							t.app.storage.telegram[jfID] = user
							err := t.app.storage.storeTelegramUsers()
							if err != nil {
								t.app.err.Printf("Failed to store Telegram users: %v", err)
							}
							break
						}
					}
				}
				continue
			default:
				tokenIndex := -1
				for i, token := range t.tokens {
					if upd.Message.Text == token {
						tokenIndex = i
						break
					}
				}
				if tokenIndex == -1 {
					err := t.QuoteReply(&upd, t.app.storage.lang.Telegram[lang].Strings.get("invalidPIN"))
					if err != nil {
						t.app.err.Printf("Telegram: Failed to send message to \"%s\": %v", upd.Message.From.UserName, err)
					}
					continue
				}
				err := t.QuoteReply(&upd, t.app.storage.lang.Telegram[lang].Strings.get("pinSuccess"))
				if err != nil {
					t.app.err.Printf("Telegram: Failed to send message to \"%s\": %v", upd.Message.From.UserName, err)
				}
				t.verifiedTokens = append(t.verifiedTokens, VerifiedToken{
					Token:    upd.Message.Text,
					ChatID:   upd.Message.Chat.ID,
					Username: upd.Message.Chat.UserName,
				})
				t.tokens[len(t.tokens)-1], t.tokens[tokenIndex] = t.tokens[tokenIndex], t.tokens[len(t.tokens)-1]
				t.tokens = t.tokens[:len(t.tokens)-1]
			}

		case <-t.ShutdownChannel:
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

// Send adds compatibility with EmailClient, fromName/fromAddr are discarded, message.Text is used, addresses are Chat IDs as strings.
func (t *TelegramDaemon) Send(fromName, fromAddr string, message *Message, address ...string) error {
	for _, addr := range address {
		ChatID, err := strconv.ParseInt(addr, 10, 64)
		if err != nil {
			return err
		}
		msg := tg.NewMessage(ChatID, message.Text)
		_, err = t.bot.Send(msg)
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
