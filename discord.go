package main

import (
	"fmt"
	"strings"

	dg "github.com/bwmarrin/discordgo"
)

type DiscordToken struct {
	Token     string
	ChannelID string
	UserID    string
	Username  string
}

type DiscordDaemon struct {
	Stopped         bool
	ShutdownChannel chan string
	bot             *dg.Session
	username        string
	tokens          map[string]DiscordToken // map of user IDs to tokens.
	verifiedTokens  []DiscordToken
	languages       map[string]string // Store of languages for user channelIDs. Added to on first interaction, and loaded from app.storage.discord on start.
	app             *appContext
}

func newDiscordDaemon(app *appContext) (*DiscordDaemon, error) {
	token := app.config.Section("discord").Key("token").String()
	if token == "" {
		return nil, fmt.Errorf("token was blank")
	}
	bot, err := dg.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	dd := &DiscordDaemon{
		Stopped:         false,
		ShutdownChannel: make(chan string),
		bot:             bot,
		tokens:          map[string]DiscordToken{},
		verifiedTokens:  []DiscordToken{},
		languages:       map[string]string{},
		app:             app,
	}
	for _, user := range app.storage.discord {
		if user.Lang != "" {
			dd.languages[user.ChannelID] = user.Lang
		}
	}
	return dd, nil
}

func (d *DiscordDaemon) NewAuthToken(channelID, userID, username string) DiscordToken {
	pin := genAuthToken()
	token := DiscordToken{
		Token:     pin,
		ChannelID: channelID,
		UserID:    userID,
		Username:  username,
	}
	return token
}

func (d *DiscordDaemon) run() {
	d.bot.AddHandler(d.messageHandler)
	d.bot.Identify.Intents = dg.IntentsGuildMessages | dg.IntentsDirectMessages
	if err := d.bot.Open(); err != nil {
		d.app.err.Printf("Discord: Failed to start daemon: %v", err)
		return
	}
	d.username = d.bot.State.User.Username
	defer d.bot.Close()
	<-d.ShutdownChannel
	d.ShutdownChannel <- "Down"
	return
}

func (d *DiscordDaemon) Shutdown() {
	d.Stopped = true
	d.ShutdownChannel <- "Down"
	<-d.ShutdownChannel
	close(d.ShutdownChannel)
}

func (d *DiscordDaemon) messageHandler(s *dg.Session, m *dg.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	sects := strings.Split(m.Content, " ")
	if len(sects) == 0 {
		return
	}
	lang := d.app.storage.lang.chosenTelegramLang
	if storedLang, ok := d.languages[m.Author.ID]; ok {
		lang = storedLang
	}
	switch msg := sects[0]; msg {
	case d.app.config.Section("telegram").Key("start_command").MustString("!start"):
		d.commandStart(s, m, lang)
	default:
		d.commandPIN(s, m, lang)
	}
}

func (d *DiscordDaemon) commandStart(s *dg.Session, m *dg.MessageCreate, lang string) {
	channel, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		d.app.err.Printf("Discord: Failed to create private channel with \"%s\": %v", m.Author.Username, err)
		return
	}
	token := d.NewAuthToken(channel.ID, m.Author.ID, m.Author.Username)
	d.tokens[m.Author.ID] = token
	content := d.app.storage.lang.Telegram[lang].Strings.get("startMessage") + "\n"
	content += d.app.storage.lang.Telegram[lang].Strings.get("languageMessage")
	_, err = s.ChannelMessageSend(channel.ID, content)
	if err != nil {
		d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", m.Author.Username, err)
		return
	}
}

func (d *DiscordDaemon) commandPIN(s *dg.Session, m *dg.MessageCreate, lang string) {
	token, ok := d.tokens[m.Author.ID]
	if !ok || token.Token != m.Content {
		_, err := s.ChannelMessageSendReply(
			m.ChannelID,
			d.app.storage.lang.Telegram[lang].Strings.get("invalidPIN"),
			m.Reference(),
		)
		if err != nil {
			d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", m.Author.Username, err)
		}
		return
	}
	_, err := s.ChannelMessageSendReply(
		m.ChannelID,
		d.app.storage.lang.Telegram[lang].Strings.get("pinSuccess"),
		m.Reference(),
	)
	if err != nil {
		d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", m.Author.Username, err)
	}
	d.verifiedTokens = append(d.verifiedTokens, token)
	delete(d.tokens, m.Author.ID)
}
