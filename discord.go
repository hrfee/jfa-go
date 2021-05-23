package main

import (
	"fmt"
	"strings"

	dg "github.com/bwmarrin/discordgo"
)

type DiscordDaemon struct {
	Stopped                                                    bool
	ShutdownChannel                                            chan string
	bot                                                        *dg.Session
	username                                                   string
	tokens                                                     []string
	verifiedTokens                                             map[string]DiscordUser // Map of tokens to discord users.
	channelID, channelName, inviteChannelID, inviteChannelName string
	guildID                                                    string
	serverChannelName, serverName                              string
	users                                                      map[string]DiscordUser // Map of user IDs to users. Added to on first interaction, and loaded from app.storage.discord on start.
	app                                                        *appContext
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
		tokens:          []string{},
		verifiedTokens:  map[string]DiscordUser{},
		users:           map[string]DiscordUser{},
		app:             app,
	}
	for _, user := range app.storage.discord {
		dd.users[user.ID] = user
	}

	return dd, nil
}

// NewAuthToken generates an 8-character pin in the form "A1-2B-CD".
func (d *DiscordDaemon) NewAuthToken() string {
	pin := genAuthToken()
	d.tokens = append(d.tokens, pin)
	return pin
}

func (d *DiscordDaemon) NewUnknownUser(channelID, userID, discrim, username string) DiscordUser {
	user := DiscordUser{
		ChannelID:     channelID,
		ID:            userID,
		Username:      username,
		Discriminator: discrim,
	}
	return user
}

func (d *DiscordDaemon) MustGetUser(channelID, userID, discrim, username string) DiscordUser {
	if user, ok := d.users[userID]; ok {
		return user
	}
	return d.NewUnknownUser(channelID, userID, discrim, username)
}

func (d *DiscordDaemon) run() {
	d.bot.AddHandler(d.messageHandler)
	d.bot.Identify.Intents = dg.IntentsGuildMessages | dg.IntentsDirectMessages | dg.IntentsGuildMembers | dg.IntentsGuildInvites
	if err := d.bot.Open(); err != nil {
		d.app.err.Printf("Discord: Failed to start daemon: %v", err)
		return
	}
	// Sometimes bot.State isn't populated quick enough
	for d.bot.State == nil {
		continue
	}
	d.username = d.bot.State.User.Username
	// Choose the last guild (server), for now we don't really support multiple anyway
	d.guildID = d.bot.State.Guilds[len(d.bot.State.Guilds)-1].ID
	guild, err := d.bot.Guild(d.guildID)
	if err != nil {
		d.app.err.Printf("Discord: Failed to get guild: %v", err)
	}
	d.serverChannelName = guild.Name
	d.serverName = guild.Name
	if channel := d.app.config.Section("discord").Key("channel").String(); channel != "" {
		d.channelName = channel
		d.serverChannelName += "/" + channel
	}
	if d.app.config.Section("discord").Key("provide_invite").MustBool(false) {
		if invChannel := d.app.config.Section("discord").Key("invite_channel").String(); invChannel != "" {
			d.inviteChannelName = invChannel
		}
	}
	defer d.bot.Close()
	<-d.ShutdownChannel
	d.ShutdownChannel <- "Down"
	return
}

// NewTempInvite creates an invite link, and returns the invite URL, as well as the URL for the server icon.
func (d *DiscordDaemon) NewTempInvite(ageSeconds, maxUses int) (inviteURL, iconURL string) {
	var inv *dg.Invite
	var err error
	if d.inviteChannelName == "" {
		d.app.err.Println("Discord: Cannot create invite without channel specified in settings.")
		return
	}
	if d.inviteChannelID == "" {
		channels, err := d.bot.GuildChannels(d.guildID)
		if err != nil {
			d.app.err.Printf("Discord: Couldn't get channel list: %v", err)
			return
		}
		found := false
		for _, channel := range channels {
			// channel, err := d.bot.Channel(ch.ID)
			// if err != nil {
			// 	d.app.err.Printf("Discord: Couldn't get channel: %v", err)
			// 	return
			// }
			if channel.Name == d.inviteChannelName {
				d.inviteChannelID = channel.ID
				found = true
				break
			}
		}
		if !found {
			d.app.err.Printf("Discord: Couldn't find invite channel \"%s\"", d.inviteChannelName)
			return
		}
	}
	// channel, err := d.bot.Channel(d.inviteChannelID)
	// if err != nil {
	// 	d.app.err.Printf("Discord: Couldn't get invite channel: %v", err)
	// 	return
	// }
	inv, err = d.bot.ChannelInviteCreate(d.inviteChannelID, dg.Invite{
		// Guild:   d.bot.State.Guilds[len(d.bot.State.Guilds)-1],
		// Channel: channel,
		// Inviter: d.bot.State.User,
		MaxAge:    ageSeconds,
		MaxUses:   maxUses,
		Temporary: false,
	})
	if err != nil {
		d.app.err.Printf("Discord: Failed to create invite: %v", err)
		return
	}
	inviteURL = "https://discord.gg/" + inv.Code
	guild, err := d.bot.Guild(d.guildID)
	if err != nil {
		d.app.err.Printf("Discord: Failed to get guild: %v", err)
		return
	}
	iconURL = guild.IconURL()
	return
}

// Returns the user(s) roughly corresponding to the username (if they are in the guild).
// if no discriminator (#xxxx) is given in the username and there are multiple corresponding users, a list of all matching users is returned.
func (d *DiscordDaemon) GetUsers(username string) []*dg.Member {
	members, err := d.bot.GuildMembers(
		d.guildID,
		"",
		1000,
	)
	if err != nil {
		d.app.err.Printf("Discord: Failed to get members: %v", err)
		return nil
	}
	hasDiscriminator := strings.Contains(username, "#")
	var users []*dg.Member
	for _, member := range members {
		if hasDiscriminator {
			if member.User.Username+"#"+member.User.Discriminator == username {
				return []*dg.Member{member}
			}
		}
		if strings.Contains(member.User.Username, username) {
			users = append(users, member)
		}
	}
	return users
}

func (d *DiscordDaemon) NewUser(ID string) (user DiscordUser, ok bool) {
	u, err := d.bot.User(ID)
	if err != nil {
		d.app.err.Printf("Discord: Failed to get user: %v", err)
		return
	}
	user.ID = ID
	user.Username = u.Username
	user.Contact = true
	user.Discriminator = u.Discriminator
	channel, err := d.bot.UserChannelCreate(ID)
	if err != nil {
		d.app.err.Printf("Discord: Failed to create DM channel: %v", err)
		return
	}
	user.ChannelID = channel.ID
	ok = true
	return
}

func (d *DiscordDaemon) Shutdown() {
	d.Stopped = true
	d.ShutdownChannel <- "Down"
	<-d.ShutdownChannel
	close(d.ShutdownChannel)
}

func (d *DiscordDaemon) messageHandler(s *dg.Session, m *dg.MessageCreate) {
	if m.GuildID != "" && d.channelName != "" {
		if d.channelID == "" {
			channel, err := s.Channel(m.ChannelID)
			if err != nil {
				d.app.err.Printf("Discord: Couldn't get channel, will monitor all: %v", err)
				d.channelName = ""
			}
			if channel.Name == d.channelName {
				d.channelID = channel.ID
			}
		}
		if d.channelID != m.ChannelID {
			d.app.debug.Printf("Discord: Ignoring message as not in specified channel")
			return
		}
	}
	if m.Author.ID == s.State.User.ID {
		return
	}
	sects := strings.Split(m.Content, " ")
	if len(sects) == 0 {
		return
	}
	lang := d.app.storage.lang.chosenTelegramLang
	if user, ok := d.users[m.Author.ID]; ok {
		if _, ok := d.app.storage.lang.Telegram[user.Lang]; ok {
			lang = user.Lang
		}
	}
	switch msg := sects[0]; msg {
	case d.app.config.Section("discord").Key("start_command").MustString("!start"):
		d.commandStart(s, m, lang)
	case "!lang":
		d.commandLang(s, m, sects, lang)
	default:
		d.commandPIN(s, m, sects, lang)
	}
}

func (d *DiscordDaemon) commandStart(s *dg.Session, m *dg.MessageCreate, lang string) {
	channel, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		d.app.err.Printf("Discord: Failed to create private channel with \"%s\": %v", m.Author.Username, err)
		return
	}
	user := d.MustGetUser(channel.ID, m.Author.ID, m.Author.Discriminator, m.Author.Username)
	d.users[m.Author.ID] = user
	content := d.app.storage.lang.Telegram[lang].Strings.get("startMessage") + "\n"
	content += d.app.storage.lang.Telegram[lang].Strings.template("languageMessage", tmpl{"command": "!lang"})
	_, err = s.ChannelMessageSend(channel.ID, content)
	if err != nil {
		d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", m.Author.Username, err)
		return
	}
}

func (d *DiscordDaemon) commandLang(s *dg.Session, m *dg.MessageCreate, sects []string, lang string) {
	if len(sects) == 1 {
		list := "!lang <lang>\n"
		for code := range d.app.storage.lang.Telegram {
			list += fmt.Sprintf("%s: %s\n", code, d.app.storage.lang.Telegram[code].Meta.Name)
		}
		_, err := s.ChannelMessageSendReply(
			m.ChannelID,
			list,
			m.Reference(),
		)
		if err != nil {
			d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", m.Author.Username, err)
		}
		return
	}
	if _, ok := d.app.storage.lang.Telegram[sects[1]]; ok {
		var user DiscordUser
		for jfID, user := range d.app.storage.discord {
			if user.ID == m.Author.ID {
				user.Lang = sects[1]
				d.app.storage.discord[jfID] = user
				if err := d.app.storage.storeDiscordUsers(); err != nil {
					d.app.err.Printf("Failed to store Discord users: %v", err)
				}
				break
			}
		}
		d.users[m.Author.ID] = user
	}
}

func (d *DiscordDaemon) commandPIN(s *dg.Session, m *dg.MessageCreate, sects []string, lang string) {
	if _, ok := d.users[m.Author.ID]; ok {
		channel, err := s.Channel(m.ChannelID)
		if err != nil {
			d.app.err.Printf("Discord: Failed to get channel: %v", err)
			return
		}
		if channel.Type != dg.ChannelTypeDM {
			d.app.debug.Println("Discord: Ignoring message as not a DM")
			return
		}
	} else {
		d.app.debug.Println("Discord: Ignoring message as user was not found")
		return
	}
	tokenIndex := -1
	for i, token := range d.tokens {
		if sects[0] == token {
			tokenIndex = i
			break
		}
	}
	if tokenIndex == -1 {
		_, err := s.ChannelMessageSend(
			m.ChannelID,
			d.app.storage.lang.Telegram[lang].Strings.get("invalidPIN"),
		)
		if err != nil {
			d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", m.Author.Username, err)
		}
		return
	}
	_, err := s.ChannelMessageSend(
		m.ChannelID,
		d.app.storage.lang.Telegram[lang].Strings.get("pinSuccess"),
	)
	if err != nil {
		d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", m.Author.Username, err)
	}
	d.verifiedTokens[sects[0]] = d.users[m.Author.ID]
	d.tokens[len(d.tokens)-1], d.tokens[tokenIndex] = d.tokens[tokenIndex], d.tokens[len(d.tokens)-1]
	d.tokens = d.tokens[:len(d.tokens)-1]
}

func (d *DiscordDaemon) SendDM(message *Message, userID ...string) error {
	channels := make([]string, len(userID))
	for i, id := range userID {
		channel, err := d.bot.UserChannelCreate(id)
		if err != nil {
			return err
		}
		channels[i] = channel.ID
	}
	return d.Send(message, channels...)
}

func (d *DiscordDaemon) Send(message *Message, channelID ...string) error {
	msg := ""
	var embeds []*dg.MessageEmbed
	if message.Markdown != "" {
		msg, embeds = StripAltText(message.Markdown, true)
	} else {
		msg = message.Text
	}
	for _, id := range channelID {
		var err error
		if len(embeds) != 0 {
			_, err = d.bot.ChannelMessageSendComplex(
				id,
				&dg.MessageSend{
					Content: msg,
					Embed:   embeds[0],
				},
			)
			if err != nil {
				return err
			}
			for i := 1; i < len(embeds); i++ {
				_, err := d.bot.ChannelMessageSendEmbed(id, embeds[i])
				if err != nil {
					return err
				}
			}
		} else {
			_, err := d.bot.ChannelMessageSend(
				id,
				msg,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
