package main

import "time"

type stringResponse struct {
	Response string `json:"response" example:"message"`
	Error    string `json:"error" example:"errorDescription"`
}

type boolResponse struct {
	Success bool `json:"success" example:"false"`
	Error   bool `json:"error" example:"true"`
}

type newUserDTO struct {
	Username        string `json:"username" example:"jeff" binding:"required"`  // User's username
	Password        string `json:"password" example:"guest" binding:"required"` // User's password
	Email           string `json:"email" example:"jeff@jellyf.in"`              // User's email address
	Code            string `json:"code" example:"abc0933jncjkcjj"`              // Invite code (required on /newUser)
	TelegramPIN     string `json:"telegram_pin" example:"A1-B2-3C"`             // Telegram verification PIN (if used)
	TelegramContact bool   `json:"telegram_contact"`                            // Whether or not to use telegram for notifications/pwrs
	DiscordPIN      string `json:"discord_pin" example:"A1-B2-3C"`              // Discord verification PIN (if used)
	DiscordContact  bool   `json:"discord_contact"`                             // Whether or not to use discord for notifications/pwrs
	MatrixPIN       string `json:"matrix_pin" example:"A1-B2-3C"`               // Matrix verification PIN (if used)
	MatrixContact   bool   `json:"matrix_contact"`                              // Whether or not to use matrix for notifications/pwrs
	CaptchaID       string `json:"captcha_id"`                                  // Captcha ID (if enabled)
	CaptchaText     string `json:"captcha_text"`                                // Captcha text (if enabled)
	Profile         string `json:"profile"`                                     // Profile (for admins only)
}

type newUserResponse struct {
	User  bool   `json:"user" binding:"required"` // Whether user was created successfully
	Email bool   `json:"email"`                   // Whether welcome email was successfully sent (always true if feature is disabled
	Error string `json:"error"`                   // Optional error message.
}

type deleteUserDTO struct {
	Users  []string `json:"users" binding:"required"` // List of usernames to delete
	Notify bool     `json:"notify"`                   // Whether to notify users of deletion
	Reason string   `json:"reason"`                   // Account deletion reason (for notification)
}

type enableDisableUserDTO struct {
	Users   []string `json:"users" binding:"required"` // List of usernames to delete
	Enabled bool     `json:"enabled"`                  // True = enable users, False = disable.
	Notify  bool     `json:"notify"`                   // Whether to notify users of deletion
	Reason  string   `json:"reason"`                   // Account deletion reason (for notification)
}

type generateInviteDTO struct {
	Months        int    `json:"months" example:"0"`                    // Number of months
	Days          int    `json:"days" example:"1"`                      // Number of days
	Hours         int    `json:"hours" example:"2"`                     // Number of hours
	Minutes       int    `json:"minutes" example:"3"`                   // Number of minutes
	UserExpiry    bool   `json:"user-expiry"`                           // Whether or not user expiry is enabled
	UserMonths    int    `json:"user-months,omitempty" example:"1"`     // Number of months till user expiry
	UserDays      int    `json:"user-days,omitempty" example:"1"`       // Number of days till user expiry
	UserHours     int    `json:"user-hours,omitempty" example:"2"`      // Number of hours till user expiry
	UserMinutes   int    `json:"user-minutes,omitempty" example:"3"`    // Number of minutes till user expiry
	SendTo        string `json:"send-to" example:"jeff@jellyf.in"`      // Send invite to this address or discord name
	MultipleUses  bool   `json:"multiple-uses" example:"true"`          // Allow multiple uses
	NoLimit       bool   `json:"no-limit" example:"false"`              // No invite use limit
	RemainingUses int    `json:"remaining-uses" example:"5"`            // Remaining invite uses
	Profile       string `json:"profile" example:"DefaultProfile"`      // Name of profile to apply on this invite
	Label         string `json:"label" example:"For Friends"`           // Optional label for the invite
	UserLabel     string `json:"user_label,omitempty" example:"Friend"` // Label to apply to users created w/ this invite.
}

type inviteProfileDTO struct {
	Invite  string `json:"invite" example:"slakdaslkdl2342"` // Invite to apply to
	Profile string `json:"profile" example:"DefaultProfile"` // Profile to use
}

type profileDTO struct {
	Admin            bool   `json:"admin" example:"false"`            // Whether profile has admin rights or not
	LibraryAccess    string `json:"libraries" example:"all"`          // Number of libraries profile has access to
	FromUser         string `json:"fromUser" example:"jeff"`          // The user the profile is based on
	Ombi             bool   `json:"ombi"`                             // Whether or not Ombi settings are stored in this profile.
	ReferralsEnabled bool   `json:"referrals_enabled" example:"true"` // Whether or not the profile has referrals enabled, and has a template invite stored.
}

type getProfilesDTO struct {
	Profiles       map[string]profileDTO `json:"profiles"`
	DefaultProfile string                `json:"default_profile"`
}

type profileChangeDTO struct {
	Name string `json:"name" example:"DefaultProfile" binding:"required"` // Name of the profile
}

type newProfileDTO struct {
	Name       string `json:"name" example:"DefaultProfile" binding:"required"`        // Name of the profile
	ID         string `json:"id" example:"ZXhhbXBsZTEyMzQ1Njc4OQo" binding:"required"` // ID of user to source settings from
	Homescreen bool   `json:"homescreen" example:"true"`                               // Whether to store homescreen layout or not
	OmbiID     string `json:"ombi_id" example:"ZXhhbXBsZTEyMzQ1Njc4OQo"`               // ID of Ombi user to source settings from (optional)
}

type inviteDTO struct {
	Code           string           `json:"code" example:"sajdlj23423j23"`         // Invite code
	Months         int              `json:"months" example:"1"`                    // Number of months till expiry
	Days           int              `json:"days" example:"1"`                      // Number of days till expiry
	Hours          int              `json:"hours" example:"2"`                     // Number of hours till expiry
	Minutes        int              `json:"minutes" example:"3"`                   // Number of minutes till expiry
	UserExpiry     bool             `json:"user-expiry"`                           // Whether or not user expiry is enabled
	UserMonths     int              `json:"user-months,omitempty" example:"1"`     // Number of months till user expiry
	UserDays       int              `json:"user-days,omitempty" example:"1"`       // Number of days till user expiry
	UserHours      int              `json:"user-hours,omitempty" example:"2"`      // Number of hours till user expiry
	UserMinutes    int              `json:"user-minutes,omitempty" example:"3"`    // Number of minutes till user expiry
	Created        int64            `json:"created" example:"1617737207510"`       // Date of creation
	Profile        string           `json:"profile" example:"DefaultProfile"`      // Profile used on this invite
	UsedBy         map[string]int64 `json:"used-by,omitempty"`                     // Users who have used this invite mapped to their creation time in Epoch/Unix time
	NoLimit        bool             `json:"no-limit,omitempty"`                    // If true, invite can be used any number of times
	RemainingUses  int              `json:"remaining-uses,omitempty"`              // Remaining number of uses (if applicable)
	SendTo         string           `json:"send_to,omitempty"`                     // Email/Discord username the invite was sent to (if applicable)
	NotifyExpiry   bool             `json:"notify-expiry,omitempty"`               // Whether to notify the requesting user of expiry or not
	NotifyCreation bool             `json:"notify-creation,omitempty"`             // Whether to notify the requesting user of account creation or not
	Label          string           `json:"label,omitempty" example:"For Friends"` // Optional label for the invite
	UserLabel      string           `json:"user_label,omitempty" example:"Friend"` // Label to apply to users created w/ this invite.
}

type getInvitesDTO struct {
	Profiles []string    `json:"profiles"` // List of profiles (name only)
	Invites  []inviteDTO `json:"invites"`  // List of invites
}

// fake DTO, if i actually used this the code would be a lot longer
type setNotifyValues map[string]struct {
	NotifyExpiry   bool `json:"notify-expiry,omitempty"`   // Whether to notify the requesting user of expiry or not
	NotifyCreation bool `json:"notify-creation,omitempty"` // Whether to notify the requesting user of account creation or not
}

type setNotifyDTO map[string]setNotifyValues

type deleteInviteDTO struct {
	Code string `json:"code" example:"skjadajd43234s"` // Code of invite to delete
}

type respUser struct {
	ID                    string `json:"id" example:"fdgsdfg45534fa"`              // userID of user
	Name                  string `json:"name" example:"jeff"`                      // Username of user
	Email                 string `json:"email,omitempty" example:"jeff@jellyf.in"` // Email address of user (if available)
	NotifyThroughEmail    bool   `json:"notify_email"`
	LastActive            int64  `json:"last_active" example:"1617737207510"` // Time of last activity on Jellyfin
	Admin                 bool   `json:"admin" example:"false"`               // Whether or not the user is Administrator
	Expiry                int64  `json:"expiry" example:"1617737207510"`      // Expiry time of user as Epoch/Unix time.
	Disabled              bool   `json:"disabled"`                            // Whether or not the user is disabled.
	Telegram              string `json:"telegram"`                            // Telegram username (if known)
	NotifyThroughTelegram bool   `json:"notify_telegram"`
	Discord               string `json:"discord"`    // Discord username (if known)
	DiscordID             string `json:"discord_id"` // Discord user ID for creating links.
	NotifyThroughDiscord  bool   `json:"notify_discord"`
	Matrix                string `json:"matrix"` // Matrix ID (if known)
	NotifyThroughMatrix   bool   `json:"notify_matrix"`
	Label                 string `json:"label"`          // Label of user, shown next to their name.
	AccountsAdmin         bool   `json:"accounts_admin"` // Whether or not the user is a jfa-go admin.
	ReferralsEnabled      bool   `json:"referrals_enabled"`
}

type getUsersDTO struct {
	UserList []respUser `json:"users"`
}

type ombiUser struct {
	Name string `json:"name,omitempty" example:"jeff"` // Name of Ombi user
	ID   string `json:"id" example:"djgkjdg7dkjfsj8"`  // userID of Ombi user
}

type ombiUsersDTO struct {
	Users []ombiUser `json:"users"`
}

type modifyEmailsDTO map[string]string

type userSettingsDTO struct {
	From       string   `json:"from"`       // Whether to apply from "user" or "profile"
	Profile    string   `json:"profile"`    // Name of profile (if from = "profile")
	ApplyTo    []string `json:"apply_to"`   // Users to apply settings to
	ID         string   `json:"id"`         // ID of user (if from = "user")
	Homescreen bool     `json:"homescreen"` // Whether to apply homescreen layout or not
}

type announcementDTO struct {
	Users   []string `json:"users"`   // List of User IDs to send announcement to
	Subject string   `json:"subject"` // Email subject
	Message string   `json:"message"` // Email content (markdown supported)
}

type announcementTemplate struct {
	Name    string `json:"name"`    // Name of template
	Subject string `json:"subject"` // Email subject
	Message string `json:"message"` // Email content (markdown supported)
}

type getAnnouncementsDTO struct {
	Announcements []string `json:"announcements"` // list of announcement names.
}

type errorListDTO map[string]map[string]string

type configDTO map[string]interface{}

// Below are for sending config

type meta struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Advanced     bool   `json:"advanced,omitempty"`
	DependsTrue  string `json:"depends_true,omitempty"`
	DependsFalse string `json:"depends_false,omitempty"`
}

type setting struct {
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	Required        bool        `json:"required"`
	Advanced        bool        `json:"advanced,omitempty"`
	RequiresRestart bool        `json:"requires_restart"`
	Type            string      `json:"type"` // Type (string, number, bool, etc.)
	Value           interface{} `json:"value"`
	Options         [][2]string `json:"options,omitempty"`
	DependsTrue     string      `json:"depends_true,omitempty"`  // If specified, this field is enabled when the specified bool setting is enabled.
	DependsFalse    string      `json:"depends_false,omitempty"` // If specified, opposite behaviour of DependsTrue.
	Style           string      `json:"style,omitempty"`
}

type section struct {
	Meta     meta               `json:"meta"`
	Order    []string           `json:"order"`
	Settings map[string]setting `json:"settings"`
}

type settings struct {
	Order    []string           `json:"order"`
	Sections map[string]section `json:"sections"`
}

type langDTO map[string]string

type emailListDTO map[string]emailListEl

type emailListEl struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type emailSetDTO struct {
	Content string `json:"content"`
}

type emailTestDTO struct {
	Address string `json:"address"`
}

type customEmailDTO struct {
	Content      string                 `json:"content"`
	Variables    []string               `json:"variables"`
	Conditionals []string               `json:"conditionals"`
	Values       map[string]interface{} `json:"values"`
	HTML         string                 `json:"html"`
	Plaintext    string                 `json:"plaintext"`
}

type extendExpiryDTO struct {
	Users     []string `json:"users"`               // List of user IDs to apply to.
	Months    int      `json:"months" example:"1"`  // Number of months to add.
	Days      int      `json:"days" example:"1"`    // Number of days to add.
	Hours     int      `json:"hours" example:"2"`   // Number of hours to add.
	Minutes   int      `json:"minutes" example:"3"` // Number of minutes to add.
	Timestamp int64    `json:"timestamp"`           // Optional, exact time to expire at. Overrides other fields.
}

type checkUpdateDTO struct {
	New    bool   `json:"new"` // Whether or not there's a new update.
	Update Update `json:"update"`
}

type telegramPinDTO struct {
	Token    string `json:"token" example:"A1-B2-3C"`
	Username string `json:"username"`
}

type telegramSetDTO struct {
	Token string `json:"token" example:"A1-B2-3C"`
	ID    string `json:"id"` // Jellyfin ID of user.
}

type SetContactMethodsDTO struct {
	ID       string `json:"id"`
	Email    bool   `json:"email"`
	Discord  bool   `json:"discord"`
	Telegram bool   `json:"telegram"`
	Matrix   bool   `json:"matrix"`
}

type DiscordUserDTO struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	ID        string `json:"id"`
}

type DiscordUsersDTO struct {
	Users []DiscordUserDTO `json:"users"`
}

type DiscordConnectUserDTO struct {
	JellyfinID string `json:"jf_id"`
	DiscordID  string `json:"discord_id"`
}

type DiscordInviteDTO struct {
	InviteURL string `json:"invite"`
	IconURL   string `json:"icon"`
}

type MatrixSendPINDTO struct {
	UserID string `json:"user_id"`
}

type MatrixCheckPINDTO struct {
	PIN string `json:"pin"`
}

type MatrixConnectUserDTO struct {
	JellyfinID string `json:"jf_id"`
	UserID     string `json:"user_id"`
}

type MatrixLoginDTO struct {
	Homeserver string `json:"homeserver"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

type ResetPasswordDTO struct {
	PIN         string `json:"pin"`
	Password    string `json:"password"`
	CaptchaText string `json:"captcha_text"`
}

type AdminPasswordResetDTO struct {
	Users []string `json:"users"` // List of Jellyfin user IDs
}

type AdminPasswordResetRespDTO struct {
	Link   string `json:"link"`   // Only returned if one of the given users doesn't have a contact method set, or only one user was requested.
	Manual bool   `json:"manual"` // Whether or not the admin has to send the link manually or not.
}

// InternalPWR stores a local version of a password reset PIN used for resets triggered by the admin when reset links are enabled.
type InternalPWR struct {
	PIN      string    `json:"pin"`
	Username string    `json:"username"`
	ID       string    `json:"id"`
	Expiry   time.Time `json:"expiry"`
}

type LogDTO struct {
	Log string `json:"log"`
}

type setAccountsAdminDTO map[string]bool

type genCaptchaDTO struct {
	ID string `json:"id"`
}

type forUserDTO struct {
	ID string `json:"id"` // Jellyfin ID
}

// ReCaptchaRequestDTO is sent to /api/siteverify, and includes the identifier of the CAPTCHA data is requested for.
type ReCaptchaRequestDTO struct {
	Secret   string `json:"secret"`
	Response string `json:"response"`
}

// ReCaptchaResponseDTO is returned upon POST to reCAPTCHA /api/siteverify, and gives details upon the use of a CAPTCHA.
type ReCaptchaResponseDTO struct {
	Success            bool     `json:"success"`
	ChallengeTimestamp string   `json:"challenge_ts"` // ISO yyyy-MM-dd'T'HH:mm:ssZZ
	Hostname           string   `json:"hostname"`
	ErrorCodes         []string `json:"error-codes"`
}

// MyDetailsDTO is sent to the user page to personalize it for the user.
type MyDetailsDTO struct {
	Id            string                      `json:"id"`
	Username      string                      `json:"username"`
	Expiry        int64                       `json:"expiry"`
	Admin         bool                        `json:"admin"`
	AccountsAdmin bool                        `json:"accounts_admin"`
	Disabled      bool                        `json:"disabled"`
	Email         *MyDetailsContactMethodsDTO `json:"email,omitempty"`
	Discord       *MyDetailsContactMethodsDTO `json:"discord,omitempty"`
	Telegram      *MyDetailsContactMethodsDTO `json:"telegram,omitempty"`
	Matrix        *MyDetailsContactMethodsDTO `json:"matrix,omitempty"`
	HasReferrals  bool                        `json:"has_referrals,omitempty"`
}

type MyDetailsContactMethodsDTO struct {
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type ModifyMyEmailDTO struct {
	Email string `json:"email"`
}

type ConfirmationTarget int

const (
	UserEmailChange ConfirmationTarget = iota
	NoOp
)

type GetMyPINDTO struct {
	PIN string `json:"pin"`
}

type ChangeMyPasswordDTO struct {
	Old string `json:"old"`
	New string `json:"new"`
}

type GetMyReferralRespDTO struct {
	Code          string `json:"code"`
	RemainingUses int    `json:"remaining_uses"`
	NoLimit       bool   `json:"no_limit"`
	Expiry        int64  `json:"expiry"` // Come back after this time to get a new referral (if UseExpiry, a new one can't be made).
	UseExpiry     bool   `json:"use_expiry"`
}

type EnableDisableReferralDTO struct {
	Users []string `json:"users"`
}

type ActivityDTO struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	UserID         string `json:"user_id"`
	Username       string `json:"username"`
	SourceType     string `json:"source_type"`
	Source         string `json:"source"`
	SourceUsername string `json:"source_username"`
	InviteCode     string `json:"invite_code"`
	Value          string `json:"value"`
	Time           int64  `json:"time"`
	IP             string `json:"ip"`
}

type GetActivitiesDTO struct {
	Type      []string `json:"type"` // Types of activity to get. Leave blank for all.
	Limit     int      `json:"limit"`
	Page      int      `json:"page"` // zero-indexed
	Ascending bool     `json:"ascending"`
}

type GetActivitiesRespDTO struct {
	Activities []ActivityDTO `json:"activities"`
	LastPage   bool          `json:"last_page"`
}

type GetActivityCountDTO struct {
	Count uint64 `json:"count"`
}

type CreateBackupDTO struct {
	Size string `json:"size"`
	Name string `json:"name"`
	Path string `json:"path"`
	Date int64  `json:"date"`
}

type GetBackupsDTO struct {
	Backups []CreateBackupDTO `json:"backups"`
}
