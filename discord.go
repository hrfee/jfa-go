package main

import (
	"fmt"
	"strings"
	"time"

	dg "github.com/bwmarrin/discordgo"
	"github.com/timshannon/badgerhold/v4"
)

type DiscordDaemon struct {
	Stopped                                                    bool
	ShutdownChannel                                            chan string
	bot                                                        *dg.Session
	username                                                   string
	tokens                                                     map[string]VerifToken  // Map of pins to tokens.
	verifiedTokens                                             map[string]DiscordUser // Map of token pins to discord users.
	channelID, channelName, inviteChannelID, inviteChannelName string
	guildID                                                    string
	serverChannelName, serverName                              string
	users                                                      map[string]DiscordUser // Map of user IDs to users. Added to on first interaction, and loaded from app.storage.discord on start.
	roleID                                                     string
	app                                                        *appContext
	commandHandlers                                            map[string]func(s *dg.Session, i *dg.InteractionCreate, lang string)
	commandIDs                                                 []string
	commandDescriptions                                        []*dg.ApplicationCommand
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
	d.bot.AddHandler(d.messageHandler)

	d.bot.AddHandler(d.commandHandler)

	d.bot.Identify.Intents = dg.IntentsGuildMessages | dg.IntentsDirectMessages | dg.IntentsGuildMembers | dg.IntentsGuildInvites
	if err := d.bot.Open(); err != nil {
		d.app.err.Printf("Discord: Failed to start daemon: %v", err)
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
		d.app.err.Printf("Discord: Failed to get roles: %v", err)
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
		d.app.err.Printf("Discord: Failed to get members: %v", err)
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
		d.app.debug.Printf("Discord: registering lang choice \"%s\":\"%s\"\n", d.app.storage.lang.Telegram[code].Meta.Name, code)
		d.commandDescriptions[1].Options[0].Choices[i] = &dg.ApplicationCommandOptionChoice{
			Name:  d.app.storage.lang.Telegram[code].Meta.Name,
			Value: code,
		}
		i++
	}

	profiles := d.app.storage.GetProfiles()
	d.commandDescriptions[3].Options[3].Choices = make([]*dg.ApplicationCommandOptionChoice, len(profiles))
	for i, profile := range profiles {
		d.app.debug.Printf("Discord: registering profile choice \"%s\"", profile.Name)
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
			d.app.err.Printf("Discord: Cannot create command \"%s\": %v", cmd.Name, err)
		} else {
			d.app.debug.Printf("Discord: registered command \"%s\"", cmd.Name)
			d.commandIDs[i] = command.ID
		}
	}
}

func (d *DiscordDaemon) deregisterCommands() {
	existingCommands, err := d.bot.ApplicationCommands(d.bot.State.User.ID, d.guildID)
	if err != nil {
		d.app.err.Printf("Discord: Failed to get commands: %v", err)
		return
	}
	for _, cmd := range existingCommands {
		if err := d.bot.ApplicationCommandDelete(d.bot.State.User.ID, d.guildID, cmd.ID); err != nil {
			d.app.err.Printf("Discord: Failed to deregister command: %v", err)
		}
	}
}

// UpdateCommands updates commands which have defined lists of options, to be used when changes occur.
func (d *DiscordDaemon) UpdateCommands() {
	// Reload Profile List
	profiles := d.app.storage.GetProfiles()
	d.commandDescriptions[3].Options[3].Choices = make([]*dg.ApplicationCommandOptionChoice, len(profiles))
	for i, profile := range profiles {
		d.app.debug.Printf("Discord: registering profile choice \"%s\"", profile.Name)
		d.commandDescriptions[3].Options[3].Choices[i] = &dg.ApplicationCommandOptionChoice{
			Name:  profile.Name,
			Value: profile.Name,
		}
	}
	cmd, err := d.bot.ApplicationCommandEdit(d.bot.State.User.ID, d.guildID, d.commandIDs[3], d.commandDescriptions[3])
	if err != nil {
		d.app.err.Printf("Discord: Failed to update profile list: %v\n", err)
	} else {
		d.commandIDs[3] = cmd.ID
	}
}

func (d *DiscordDaemon) commandHandler(s *dg.Session, i *dg.InteractionCreate) {
	if h, ok := d.commandHandlers[i.ApplicationCommandData().Name]; ok {
		if i.GuildID != "" && d.channelName != "" {
			if d.channelID == "" {
				channel, err := s.Channel(i.ChannelID)
				if err != nil {
					d.app.err.Printf("Discord: Couldn't get channel, will monitor all: %v", err)
					d.channelName = ""
				}
				if channel.Name == d.channelName {
					d.channelID = channel.ID
				}
			}
			if d.channelID != i.ChannelID {
				d.app.debug.Printf("Discord: Ignoring message as not in specified channel")
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
		d.app.err.Printf("Discord: Failed to create private channel with \"%s\": %v", i.Interaction.Member.User.Username, err)
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
		d.app.err.Printf("Discord: Failed to send reply: %v", err)
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
			d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", i.Interaction.Member.User.Username, err)
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
		d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", i.Interaction.Member.User.Username, err)
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
			d.app.err.Printf("Discord: Failed to send reply: %v", err)
			return
		}
	}
}

func (d *DiscordDaemon) cmdInvite(s *dg.Session, i *dg.InteractionCreate, lang string) {
	channel, err := s.UserChannelCreate(i.Interaction.Member.User.ID)
	if err != nil {
		d.app.err.Printf("Discord: Failed to create private channel with \"%s\": %v", i.Interaction.Member.User.Username, err)
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
	//	Check whether requestor is linked to the admin account
	requesterEmail, ok := d.app.storage.GetEmailsKey(requester.JellyfinID)
	if !ok {
		d.app.err.Printf("Failed to verify admin")
	}
	if !requesterEmail.Admin {
		d.app.err.Printf("User is not admin")
		//add response message
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
		Label:         fmt.Sprintf("Discord: %s", RenderDiscordUsername(recipient)),
	}
	if profileName != "" {
		if _, ok := d.app.storage.GetProfileKey(profileName); ok {
			invite.Profile = profileName
		}
	}

	if recipient != nil && d.app.config.Section("invite_emails").Key("enabled").MustBool(false) {
		d.app.debug.Printf("%s: Sending invite message", invite.Code)
		invname, err := d.bot.GuildMember(d.guildID, recipient.ID)
		invite.SendTo = invname.User.Username
		msg, err := d.app.email.constructInvite(invite.Code, invite, d.app, false)
		if err != nil {
			invite.SendTo = fmt.Sprintf("Failed to send to %s", RenderDiscordUsername(recipient))
			d.app.err.Printf("%s: Failed to construct invite message: %v", invite.Code, err)
			err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
				Type: dg.InteractionResponseChannelMessageWithSource,
				Data: &dg.InteractionResponseData{
					Content: d.app.storage.lang.Telegram[lang].Strings.get("sentInviteFailure"),
					Flags:   64, // Ephemeral
				},
			})
			if err != nil {
				d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", RenderDiscordUsername(requester), err)
			}
		} else {
			var err error
			err = d.app.discord.SendDM(msg, recipient.ID)
			if err != nil {
				invite.SendTo = fmt.Sprintf("Failed to send to %s", RenderDiscordUsername(recipient))
				d.app.err.Printf("%s: %s: %v", invite.Code, invite.SendTo, err)
				err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
					Type: dg.InteractionResponseChannelMessageWithSource,
					Data: &dg.InteractionResponseData{
						Content: d.app.storage.lang.Telegram[lang].Strings.get("sentInviteFailure"),
						Flags:   64, // Ephemeral
					},
				})
				if err != nil {
					d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", RenderDiscordUsername(requester), err)
				}
			} else {
				d.app.info.Printf("%s: Sent invite email to \"%s\"", invite.Code, RenderDiscordUsername(recipient))
				err := s.InteractionRespond(i.Interaction, &dg.InteractionResponse{
					Type: dg.InteractionResponseChannelMessageWithSource,
					Data: &dg.InteractionResponseData{
						Content: d.app.storage.lang.Telegram[lang].Strings.get("sentInvite"),
						Flags:   64, // Ephemeral
					},
				})
				if err != nil {
					d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", RenderDiscordUsername(requester), err)
				}
			}
		}
	}
	//if profile != "" {
	d.app.storage.SetInvitesKey(invite.Code, invite)
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
	case "!" + d.app.config.Section("discord").Key("start_command").MustString("start"):
		d.msgStart(s, m, lang)
	case "!lang":
		d.msgLang(s, m, sects, lang)
	default:
		d.msgPIN(s, m, sects, lang)
	}
}

func (d *DiscordDaemon) msgStart(s *dg.Session, m *dg.MessageCreate, lang string) {
	channel, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		d.app.err.Printf("Discord: Failed to create private channel with \"%s\": %v", m.Author.Username, err)
		return
	}
	user := d.MustGetUser(channel.ID, m.Author.ID, m.Author.Discriminator, m.Author.Username)
	d.users[m.Author.ID] = user

	_, err = d.bot.ChannelMessageSendReply(m.ChannelID, d.app.storage.lang.Telegram[lang].Strings.get("discordDMs"), m.Reference())
	if err != nil {
		d.app.err.Printf("Discord: Failed to send reply to \"%s\": %v", m.Author.Username, err)
		return
	}

	content := d.app.storage.lang.Telegram[lang].Strings.get("startMessage") + "\n"
	content += d.app.storage.lang.Telegram[lang].Strings.template("languageMessage", tmpl{"command": "!lang"})
	_, err = s.ChannelMessageSend(channel.ID, content)
	if err != nil {
		d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", m.Author.Username, err)
		return
	}
}

func (d *DiscordDaemon) msgLang(s *dg.Session, m *dg.MessageCreate, sects []string, lang string) {
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
		for _, u := range d.app.storage.GetDiscord() {
			if u.ID == m.Author.ID {
				u.Lang = sects[1]
				d.app.storage.SetDiscordKey(u.JellyfinID, u)
				user = u
				break
			}
		}
		d.users[m.Author.ID] = user
	}
}

func (d *DiscordDaemon) msgPIN(s *dg.Session, m *dg.MessageCreate, sects []string, lang string) {
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
	user, ok := d.tokens[sects[0]]
	if !ok || time.Now().After(user.Expiry) {
		_, err := s.ChannelMessageSend(
			m.ChannelID,
			d.app.storage.lang.Telegram[lang].Strings.get("invalidPIN"),
		)
		if err != nil {
			d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", m.Author.Username, err)
		}
		delete(d.tokens, sects[0])
		return
	}
	_, err := s.ChannelMessageSend(
		m.ChannelID,
		d.app.storage.lang.Telegram[lang].Strings.get("pinSuccess"),
	)
	if err != nil {
		d.app.err.Printf("Discord: Failed to send message to \"%s\": %v", m.Author.Username, err)
	}
	dcUser := d.users[m.Author.ID]
	dcUser.JellyfinID = user.JellyfinID
	d.verifiedTokens[sects[0]] = dcUser
	delete(d.tokens, sects[0])
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
func (d *DiscordDaemon) UserVerified(pin string) (user DiscordUser, ok bool) {
	user, ok = d.verifiedTokens[pin]
	// delete(d.verifiedTokens, pin)
	return
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

// DeleteVerifiedUser removes the token with the given PIN.
func (d *DiscordDaemon) DeleteVerifiedUser(pin string) {
	delete(d.verifiedTokens, pin)
}
