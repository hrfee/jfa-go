package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	dg "github.com/bwmarrin/discordgo"
	lm "github.com/hrfee/jfa-go/logmessages"
	"github.com/timshannon/badgerhold/v4"
)

type DiscordDaemon struct {
	Stopped                       bool
	ShutdownChannel               chan string
	bot                           *dg.Session
	username                      string
	tokens                        map[string]VerifToken  // Map of pins to tokens.
	verifiedTokens                map[string]DiscordUser // Map of token pins to discord users.
	Channel, InviteChannel        struct{ ID, Name string }
	guildID                       string
	serverChannelName, serverName string
	users                         map[string]DiscordUser // Map of user IDs to users. Added to on first interaction, and loaded from app.storage.discord on start.
	roleID                        string
	app                           *appContext
	commandHandlers               map[string]func(s *dg.Session, i *dg.InteractionCreate, lang string)
	commandIDs                    []string
	commandDescriptions           []*dg.ApplicationCommand
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
		tokens:          map[string]VerifToken{},
		verifiedTokens:  map[string]DiscordUser{},
		users:           map[string]DiscordUser{},
		app:             app,
		roleID:          app.config.Section("discord").Key("apply_role").String(),
		commandHandlers: map[string]func(s *dg.Session, i *dg.InteractionCreate, lang string){},
		commandIDs:      []string{},
	}
	dd.commandHandlers[app.config.Section("discord").Key("start_command").MustString("start")] = dd.cmdStart
	dd.commandHandlers["lang"] = dd.cmdLang
	dd.commandHandlers["pin"] = dd.cmdPIN
	dd.commandHandlers["inv"] = dd.cmdInvite
	for _, user := range app.storage.GetDiscord() {
		dd.users[user.ID] = user
	}

	return dd, nil
}

// SetTransport sets the http.Transport to use for requests. Can be used to set a proxy.
func (d *DiscordDaemon) SetTransport(t *http.Transport) {
	d.bot.Client.Transport = t
}

// NewAuthToken generates an 8-character pin in the form "A1-2B-CD".
func (d *DiscordDaemon) NewAuthToken() string {
	pin := genAuthToken()
	d.tokens[pin] = VerifToken{Expiry: time.Now().Add(VERIF_TOKEN_EXPIRY_SEC * time.Second), JellyfinID: ""}
	return pin
}

// NewAssignedAuthToken generates an 8-character pin in the form "A1-2B-CD",
// and assigns it for access only with the given Jellyfin ID.
func (d *DiscordDaemon) NewAssignedAuthToken(id string) string {
	pin := genAuthToken()
	d.tokens[pin] = VerifToken{Expiry: time.Now().Add(VERIF_TOKEN_EXPIRY_SEC * time.Second), JellyfinID: id}
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
	d.bot.AddHandler(d.commandHandler)

	d.bot.Identify.Intents = dg.IntentsGuildMessages | dg.IntentsDirectMessages | dg.IntentsGuildMembers | dg.IntentsGuildInvites
	if err := d.bot.Open(); err != nil {
		d.app.err.Printf(lm.FailedStartDaemon, lm.Discord, err)
		return
	}
	// Wait for everything to populate, it's slow sometimes.
	for d.bot.State == nil {
		continue
	}
	for d.bot.State.User == nil {
		continue
	}
	d.username = d.bot.State.User.Username
	for d.bot.State.Guilds == nil {
		continue
	}
	// Choose the last guild (server), for now we don't really support multiple anyway
	d.guildID = d.bot.State.Guilds[len(d.bot.State.Guilds)-1].ID
	guild, err := d.bot.Guild(d.guildID)
	if err != nil {
		d.app.err.Printf(lm.FailedGetDiscordGuild, err)
	}
	d.serverChannelName = guild.Name
	d.serverName = guild.Name
	if channel := d.app.config.Section("discord").Key("channel").String(); channel != "" {
		d.Channel.Name = channel
		d.serverChannelName += "/" + channel
	}
	if d.app.config.Section("discord").Key("provide_invite").MustBool(false) {
		if invChannel := d.app.config.Section("discord").Key("invite_channel").String(); invChannel != "" {
			d.InviteChannel.Name = invChannel
		}
	}
	err = d.bot.UpdateGameStatus(0, "/"+d.app.config.Section("discord").Key("start_command").MustString("start"))
	defer d.deregisterCommands()
	defer d.bot.Close()

	go d.registerCommands()

	<-d.ShutdownChannel
	d.ShutdownChannel <- "Down"
	return
}

// ListRoles returns a list of available (excluding bot and @everyone) roles in a guild as a list of containing an array of the guild ID and its name.
func (d *DiscordDaemon) ListRoles() (roles [][2]string, err error) {
	var r []*dg.Role
	r, err = d.bot.GuildRoles(d.guildID)
	if err != nil {
		d.app.err.Printf(lm.FailedGetDiscordRoles, err)
		return
	}
	for _, role := range r {
		if role.Name != d.username && role.Name != "@everyone" {
			roles = append(roles, [2]string{role.ID, role.Name})
		}
	}
	// roles = make([][2]string, len(r))
	// for i, role := range r {
	// 	roles[i] = [2]string{role.ID, role.Name}
	// }
	return
}

// ApplyRole applies the member role to the given user if set.
func (d *DiscordDaemon) ApplyRole(userID string) error {
	if d.roleID == "" {
		return nil
	}
	return d.bot.GuildMemberRoleAdd(d.guildID, userID, d.roleID)
}

// RemoveRole removes the member role to the given user if set.
func (d *DiscordDaemon) RemoveRole(userID string) error {
	if d.roleID == "" {
		return nil
	}
	return d.bot.GuildMemberRoleRemove(d.guildID, userID, d.roleID)
}

// SetRoleDisabled removes the role if "disabled", and applies if "!disabled".
func (d *DiscordDaemon) SetRoleDisabled(userID string, disabled bool) (err error) {
	if disabled {
		err = d.RemoveRole(userID)
	} else {
		err = d.ApplyRole(userID)
	}
	return
}

// NewTempInvite creates an invite link, and returns the invite URL, as well as the URL for the server icon.
func (d *DiscordDaemon) NewTempInvite(ageSeconds, maxUses int) (inviteURL, iconURL string) {
	var inv *dg.Invite
	var err error
	if d.InviteChannel.Name == "" {
		d.app.err.Printf(lm.FailedCreateDiscordInviteChannel, lm.InviteChannelEmpty)
		return
	}
	if d.InviteChannel.ID == "" {
		channels, err := d.bot.GuildChannels(d.guildID)
		if err != nil {
			d.app.err.Printf(lm.FailedGetDiscordChannels, err)
			return
		}
		found := false
		for _, channel := range channels {
			// channel, err := d.bot.Channel(ch.ID)
			// if err != nil {
			// 	d.app.err.Printf(lm.FailedGetDiscordChannel, ch.ID, err)
			// 	return
			// }
			if channel.Name == d.InviteChannel.Name {
				d.InviteChannel.ID = channel.ID
				found = true
				break
			}
		}
		if !found {
			d.app.err.Printf(lm.FailedGetDiscordChannel, d.InviteChannel.Name, lm.NotFound)
			return
		}
	}
	// channel, err := d.bot.Channel(d.inviteChannelID)
	// if err != nil {
	// 	d.app.err.Printf(lm.FailedGetDiscordChannel, d.inviteChannelID, err)
	// 	return
	// }
	inv, err = d.bot.ChannelInviteCreate(d.InviteChannel.ID, dg.Invite{
		// Guild:   d.bot.State.Guilds[len(d.bot.State.Guilds)-1],
		// Channel: channel,
		// Inviter: d.bot.State.User,
		MaxAge:    ageSeconds,
		MaxUses:   maxUses,
		Temporary: false,
	})
	if err != nil {
		d.app.err.Printf(lm.FailedGenerateDiscordInvite, err)
		return
	}
	inviteURL = "https://discord.gg/" + inv.Code
	guild, err := d.bot.Guild(d.guildID)
	if err != nil {
		d.app.err.Printf(lm.FailedGetDiscordGuild, err)
		return
	}
	iconURL = guild.IconURL("256")
	return
}

// RenderDiscordUsername returns String of discord username, with support for new discriminator-less versions.
func RenderDiscordUsername[DcUser *dg.User | DiscordUser](user DcUser) string {
	u, ok := interface{}(user).(*dg.User)
	var discriminator, username string
	if ok {
		discriminator = u.Discriminator
		username = u.Username
	} else {
		u2 := interface{}(user).(DiscordUser)
		discriminator = u2.Discriminator
		username = u2.Username
	}

	if discriminator == "0" {
		return "@" + username
	}
	return username + "#" + discriminator
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
		d.app.err.Printf(lm.FailedGetDiscordGuildMembers, err)
		return nil
	}
	hasDiscriminator := strings.Contains(username, "#")
	hasAt := strings.HasPrefix(username, "@")
	if hasAt {
		username = username[1:]
	}
	var users []*dg.Member
	for _, member := range members {
		if hasDiscriminator {
			if member.User.Username+"#"+member.User.Discriminator == username {
				return []*dg.Member{member}
			}
		}
		if hasAt {
			if member.User.Username == username && member.User.Discriminator == "0" {
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
		d.app.err.Printf(lm.FailedGetUser, ID, lm.Discord, err)
		return
	}
	user.ID = ID
	user.Username = u.Username
	user.Contact = true
	user.Discriminator = u.Discriminator
	channel, err := d.bot.UserChannelCreate(ID)
	if err != nil {
		d.app.err.Printf(lm.FailedCreateDiscordDMChannel, ID, err)
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

func (d *DiscordDaemon) registerCommands() {
	d.commandDescriptions = []*dg.ApplicationCommand{
		{
			Name:        d.app.config.Section("discord").Key("start_command").MustString("start"),
			Description: "Start the Discord linking process. The bot will send further instructions.",
		},
		{
			Name:        "lang",
			Description: "Set the language for the bot.",
			Options: []*dg.ApplicationCommandOption{
				{
					Type:        dg.ApplicationCommandOptionString,
					Name:        "language",
					Description: "Language Name",
					Required:    true,
					Choices:     []*dg.ApplicationCommandOptionChoice{},
				},
			},
		},
		{
			Name:        "pin",
			Description: "Send PIN for Discord verification.",
			Options: []*dg.ApplicationCommandOption{
				{
					Type:        dg.ApplicationCommandOptionString,
					Name:        "pin",
					Description: "Verification PIN (e.g AB-CD-EF)",
					Required:    true,
				},
			},
		},
		{
			Name:        "inv",
			Description: "Send an invite to a discord user (admin only).",
			Options: []*dg.ApplicationCommandOption{
				{
					Type:        dg.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "User to Invite.",
					Required:    true,
				},
				{
					Type:        dg.ApplicationCommandOptionInteger,
					Name:        "expiry",
					Description: "Time in minutes before expiration.",
					Required:    false,
				},
				/* Label should be automatically set to something like "Discord invite for @username"
				{
					Type:        dg.ApplicationCommandOptionString,
					Name:        "label",
					Description: "Label given to this invite (shown on the Admin page)",
					Required:    false,
				}, */
				{
					Type:        dg.ApplicationCommandOptionString,
					Name:        "user_label",
					Description: "Label given to users created with this invite.",
					Required:    false,
				},
				{
					Type:        dg.ApplicationCommandOptionString,
					Name:        "profile",
					Description: "Profile to apply to the created user.",
					Required:    false,
				},
			},
		},
	}
	d.commandDescriptions[1].Options[0].Choices = make([]*dg.ApplicationCommandOptionChoice, len(d.app.storage.lang.Telegram))
	i := 0
	for code := range d.app.storage.lang.Telegram {
		d.app.debug.Printf(lm.RegisterDiscordChoice, lm.Lang, d.app.storage.lang.Telegram[code].Meta.Name+":"+code)
		d.commandDescriptions[1].Options[0].Choices[i] = &dg.ApplicationCommandOptionChoice{
			Name:  d.app.storage.lang.Telegram[code].Meta.Name,
			Value: code,
		}
		i++
	}

	profiles := d.app.storage.GetProfiles()
	d.commandDescriptions[3].Options[3].Choices = make([]*dg.ApplicationCommandOptionChoice, len(profiles))
	for i, profile := range profiles {
		d.app.debug.Printf(lm.RegisterDiscordChoice, lm.Profile, profile.Name)
		d.commandDescriptions[3].Options[3].Choices[i] = &dg.ApplicationCommandOptionChoice{
			Name:  profile.Name,
			Value: profile.Name,
		}
	}

	// d.deregisterCommands()

	d.commandIDs = make([]string, len(d.commandDescriptions))
	// cCommands, err := d.bot.ApplicationCommandBulkOverwrite(d.bot.State.User.ID, d.guildID, commands)
	// if err != nil {
	// 	d.app.err.Printf("Discord: Cannot create commands: %v", err)
	// }
	for i, cmd := range d.commandDescriptions {
		command, err := d.bot.ApplicationCommandCreate(d.bot.State.User.ID, d.guildID, cmd)
		if err != nil {
			d.app.err.Printf(lm.FailedRegisterDiscordCommand, cmd.Name, err)
		} else {
			d.app.debug.Printf(lm.RegisterDiscordCommand, cmd.Name)
			d.commandIDs[i] = command.ID
		}
	}
}

func (d *DiscordDaemon) deregisterCommands() {
	existingCommands, err := d.bot.ApplicationCommands(d.bot.State.User.ID, d.guildID)
	if err != nil {
		d.app.err.Printf(lm.FailedGetDiscordCommands, err)
		return
	}
	for _, cmd := range existingCommands {
		if err := d.bot.ApplicationCommandDelete(d.bot.State.User.ID, d.guildID, cmd.ID); err != nil {
			d.app.err.Printf(lm.FailedDeregDiscordCommand, cmd.Name, err)
		}
	}
}

// UpdateCommands updates commands which have defined lists of options, to be used when changes occur.
func (d *DiscordDaemon) UpdateCommands() {
	// Reload Profile List
	profiles := d.app.storage.GetProfiles()
	d.commandDescriptions[3].Options[3].Choices = make([]*dg.ApplicationCommandOptionChoice, len(profiles))
	for i, profile := range profiles {
		d.app.debug.Printf(lm.RegisterDiscordChoice, lm.Profile, profile.Name)
		d.commandDescriptions[3].Options[3].Choices[i] = &dg.ApplicationCommandOptionChoice{
			Name:  profile.Name,
			Value: profile.Name,
		}
	}
	cmd, err := d.bot.ApplicationCommandEdit(d.bot.State.User.ID, d.guildID, d.commandIDs[3], d.commandDescriptions[3])
	if err != nil {
		d.app.err.Printf(lm.FailedRegisterDiscordChoices, lm.Profile, err)
	} else {
		d.commandIDs[3] = cmd.ID
	}
}

func (d *DiscordDaemon) commandHandler(s *dg.Session, i *dg.InteractionCreate) {
	if h, ok := d.commandHandlers[i.ApplicationCommandData().Name]; ok {
		if i.GuildID != "" && d.Channel.Name != "" {
			if d.Channel.ID == "" {
				channel, err := s.Channel(i.ChannelID)
				if err != nil {
					d.app.err.Printf(lm.FailedGetDiscordChannel, i.ChannelID, err)
					d.app.err.Println(lm.MonitorAllDiscordChannels)
					d.Channel.Name = ""
				}
				if channel.Name == d.Channel.Name {
					d.Channel.ID = channel.ID
				}
			}
			if d.Channel.ID != i.ChannelID {
				d.app.debug.Printf(lm.IgnoreOutOfChannelMessage, lm.Discord)
				return
			}
		}
		if i.Interaction.Member.User.ID == s.State.User.ID {
			return
		}
		lang := d.app.storage.lang.chosenTelegramLang
		if user, ok := d.users[i.Interaction.Member.User.ID]; ok {
			if _, ok := d.app.storage.lang.Telegram[user.Lang]; ok {
				lang = user.Lang
			}
		}
		h(s, i, lang)
	}
}

// cmd* methods handle slash-commands, msg* methods handle ! commands.

func (d *DiscordDaemon) cmdStart(s *dg.Session, i *dg.InteractionCreate, lang string) {
	channel, err := s.UserChannelCreate(i.Interaction.Member.User.ID)
	if err != nil {
		d.app.err.Printf(lm.FailedCreateDiscordDMChannel, i.Interaction.Member.User.ID, err)
		return
	}
	user := d.MustGetUser(channel.ID, i.Interaction.Member.User.ID, i.Interaction.Member.User.Discriminator, i.Interaction.Member.User.Username)
	d.users[i.Interaction.Member.User.ID] = user

	content := d.app.storage.lang.Telegram[lang].Strings.get("discordStartMessage") + "\n"
	content += d.app.storage.lang.Telegram[lang].Strings.template("languageMessageDiscord", tmpl{"command": "/lang"})
	err = s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
		//	Type: dg.InteractionResponseChannelMessageWithSource,
		Type: dg.InteractionResponseChannelMessageWithSource,
		Data: &dg.InteractionResponseData{
			Content: content,
			Flags:   64, // Ephemeral
		},
	})
	if err != nil {
		d.app.err.Printf(lm.FailedReply, lm.Discord, i.Interaction.Member.User.ID, err)
		return
	}
}

func (d *DiscordDaemon) cmdPIN(s *dg.Session, i *dg.InteractionCreate, lang string) {
	pin := i.ApplicationCommandData().Options[0].StringValue()
	user, ok := d.tokens[pin]
	if !ok || time.Now().After(user.Expiry) {
		err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
			//	Type: dg.InteractionResponseChannelMessageWithSource,
			Type: dg.InteractionResponseChannelMessageWithSource,
			Data: &dg.InteractionResponseData{
				Content: d.app.storage.lang.Telegram[lang].Strings.get("invalidPIN"),
				Flags:   64, // Ephemeral
			},
		})
		if err != nil {
			d.app.err.Printf(lm.FailedReply, lm.Discord, i.Interaction.Member.User.ID, err)
		}
		delete(d.tokens, pin)
		return
	}
	err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
		//	Type: dg.InteractionResponseChannelMessageWithSource,
		Type: dg.InteractionResponseChannelMessageWithSource,
		Data: &dg.InteractionResponseData{
			Content: d.app.storage.lang.Telegram[lang].Strings.get("pinSuccess"),
			Flags:   64, // Ephemeral
		},
	})
	if err != nil {
		d.app.err.Printf(lm.FailedReply, lm.Discord, i.Interaction.Member.User.ID, err)
	}
	dcUser := d.users[i.Interaction.Member.User.ID]
	dcUser.JellyfinID = user.JellyfinID
	d.verifiedTokens[pin] = dcUser
	delete(d.tokens, pin)
}

func (d *DiscordDaemon) cmdLang(s *dg.Session, i *dg.InteractionCreate, lang string) {
	code := i.ApplicationCommandData().Options[0].StringValue()
	if _, ok := d.app.storage.lang.Telegram[code]; ok {
		var user DiscordUser
		for _, u := range d.app.storage.GetDiscord() {
			if u.ID == i.Interaction.Member.User.ID {
				u.Lang = code
				lang = code
				d.app.storage.SetDiscordKey(u.JellyfinID, u)
				user = u
				break
			}
		}
		d.users[i.Interaction.Member.User.ID] = user
		err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
			//	Type: dg.InteractionResponseChannelMessageWithSource,
			Type: dg.InteractionResponseChannelMessageWithSource,
			Data: &dg.InteractionResponseData{
				Content: d.app.storage.lang.Telegram[lang].Strings.template("languageSet", tmpl{"language": d.app.storage.lang.Telegram[lang].Meta.Name}),
				Flags:   64, // Ephemeral
			},
		})
		if err != nil {
			d.app.err.Printf(lm.FailedReply, lm.Discord, i.Interaction.Member.User.ID, err)
			return
		}
	}
}

func (d *DiscordDaemon) cmdInvite(s *dg.Session, i *dg.InteractionCreate, lang string) {
	channel, err := s.UserChannelCreate(i.Interaction.Member.User.ID)
	if err != nil {
		d.app.err.Printf(lm.FailedCreateDiscordDMChannel, i.Interaction.Member.User.ID, err)
		return
	}
	requester := d.MustGetUser(channel.ID, i.Interaction.Member.User.ID, i.Interaction.Member.User.Discriminator, i.Interaction.Member.User.Username)
	d.users[i.Interaction.Member.User.ID] = requester
	recipient := i.ApplicationCommandData().Options[0].UserValue(s)
	// d.app.debug.Println(invuser)
	//label := i.ApplicationCommandData().Options[2].StringValue()
	//profile := i.ApplicationCommandData().Options[3].StringValue()
	//mins, err := strconv.Atoi(i.ApplicationCommandData().Options[1].StringValue())
	//if mins > 0 {
	//	expmin = mins
	//}
	// We want the same criteria for running this command as accessing the admin page (i.e. an "admin" of some sort)
	if !(d.app.canAccessAdminPageByID(requester.JellyfinID)) {
		d.app.err.Printf(lm.FailedGenerateInvite, fmt.Sprintf(lm.NonAdminUser, requester.JellyfinID))
		s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
			Type: dg.InteractionResponseChannelMessageWithSource,
			Data: &dg.InteractionResponseData{
				Content: d.app.storage.lang.Telegram[lang].Strings.get("noPermission"),
				Flags:   64, // Ephemeral
			},
		})
		return
	}

	var expiryMinutes int64 = 30
	userLabel := ""
	profileName := ""

	for i, opt := range i.ApplicationCommandData().Options {
		if i == 0 {
			continue
		}
		switch opt.Name {
		case "expiry":
			expiryMinutes = opt.IntValue()
		case "user_label":
			userLabel = opt.StringValue()
		case "profile":
			profileName = opt.StringValue()
		}
	}

	currentTime := time.Now()

	validTill := currentTime.Add(time.Minute * time.Duration(expiryMinutes))

	invite := Invite{
		Code:          GenerateInviteCode(),
		Created:       currentTime,
		RemainingUses: 1,
		UserExpiry:    false,
		ValidTill:     validTill,
		UserLabel:     userLabel,
		Profile:       "Default",
		Label:         fmt.Sprintf("%s: %s", lm.Discord, RenderDiscordUsername(recipient)),
	}
	if profileName != "" {
		if _, ok := d.app.storage.GetProfileKey(profileName); ok {
			invite.Profile = profileName
		}
	}

	if recipient != nil && d.app.config.Section("invite_emails").Key("enabled").MustBool(false) {
		invname, err := d.bot.GuildMember(d.guildID, recipient.ID)
		invite.SendTo = invname.User.Username
		msg, err := d.app.email.constructInvite(invite.Code, invite, d.app, false)
		if err != nil {
			invite.SendTo = fmt.Sprintf(lm.FailedConstructInviteMessage, invite.Code, err)
			d.app.err.Println(invite.SendTo)
			err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
				Type: dg.InteractionResponseChannelMessageWithSource,
				Data: &dg.InteractionResponseData{
					Content: d.app.storage.lang.Telegram[lang].Strings.get("sentInviteFailure"),
					Flags:   64, // Ephemeral
				},
			})
			if err != nil {
				d.app.err.Printf(lm.FailedReply, lm.Discord, requester.ID, err)
			}
		} else {
			var err error
			err = d.app.discord.SendDM(msg, recipient.ID)
			if err != nil {
				invite.SendTo = fmt.Sprintf(lm.FailedSendInviteMessage, invite.Code, RenderDiscordUsername(recipient), err)
				d.app.err.Println(invite.SendTo)
				err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
					Type: dg.InteractionResponseChannelMessageWithSource,
					Data: &dg.InteractionResponseData{
						Content: d.app.storage.lang.Telegram[lang].Strings.get("sentInviteFailure"),
						Flags:   64, // Ephemeral
					},
				})
				if err != nil {
					d.app.err.Printf(lm.FailedReply, lm.Discord, requester.ID, err)
				}
			} else {
				d.app.info.Printf(lm.SentInviteMessage, invite.Code, RenderDiscordUsername(recipient))
				err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
					Type: dg.InteractionResponseChannelMessageWithSource,
					Data: &dg.InteractionResponseData{
						Content: d.app.storage.lang.Telegram[lang].Strings.get("sentInvite"),
						Flags:   64, // Ephemeral
					},
				})
				if err != nil {
					d.app.err.Printf(lm.FailedReply, lm.Discord, requester.ID, err)
				}
			}
		}
	}
	//if profile != "" {
	d.app.storage.SetInvitesKey(invite.Code, invite)
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

// UserVerified returns whether or not a token with the given PIN has been verified, and the user itself.
func (d *DiscordDaemon) UserVerified(pin string) (ContactMethodUser, bool) {
	u, ok := d.verifiedTokens[pin]
	// delete(d.verifiedTokens, pin)
	return &u, ok
}

// AssignedUserVerified returns whether or not a user with the given PIN has been verified, and the token itself.
// Returns false if the given Jellyfin ID does not match the one in the user.
func (d *DiscordDaemon) AssignedUserVerified(pin string, jfID string) (user DiscordUser, ok bool) {
	user, ok = d.verifiedTokens[pin]
	if ok && user.JellyfinID != jfID {
		ok = false
	}
	// delete(d.verifiedUsers, pin)
	return
}

// UserExists returns whether or not a user with the given ID exists.
func (d *DiscordDaemon) UserExists(id string) bool {
	c, err := d.app.storage.db.Count(&DiscordUser{}, badgerhold.Where("ID").Eq(id))
	return err != nil || c > 0
}

// Exists returns whether or not the given user exists.
func (d *DiscordDaemon) Exists(user ContactMethodUser) bool {
	return d.UserExists(user.MethodID().(string))
}

// DeleteVerifiedToken removes the token with the given PIN.
func (d *DiscordDaemon) DeleteVerifiedToken(PIN string) {
	delete(d.verifiedTokens, PIN)
}

func (d *DiscordDaemon) PIN(req newUserDTO) string { return req.DiscordPIN }

func (d *DiscordDaemon) Name() string { return lm.Discord }

func (d *DiscordDaemon) Required() bool {
	return d.app.config.Section("discord").Key("required").MustBool(false)
}

func (d *DiscordDaemon) UniqueRequired() bool {
	return d.app.config.Section("discord").Key("require_unique").MustBool(false)
}

func (d *DiscordDaemon) PostVerificationTasks(PIN string, u ContactMethodUser) error {
	err := d.ApplyRole(u.MethodID().(string))
	if err != nil {
		return fmt.Errorf(lm.FailedSetDiscordMemberRole, err)
	}
	return err
}

func (d *DiscordUser) Name() string                          { return RenderDiscordUsername(*d) }
func (d *DiscordUser) SetMethodID(id any)                    { d.ID = id.(string) }
func (d *DiscordUser) MethodID() any                         { return d.ID }
func (d *DiscordUser) SetJellyfin(id string)                 { d.JellyfinID = id }
func (d *DiscordUser) Jellyfin() string                      { return d.JellyfinID }
func (d *DiscordUser) SetAllowContactFromDTO(req newUserDTO) { d.Contact = req.DiscordContact }
func (d *DiscordUser) SetAllowContact(contact bool)          { d.Contact = contact }
func (d *DiscordUser) AllowContact() bool                    { return d.Contact }
func (d *DiscordUser) Store(st *Storage) {
	st.SetDiscordKey(d.Jellyfin(), *d)
}
