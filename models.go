package main

type stringResponse struct {
	Response string `json:"response" example:"message"`
	Error    string `json:"error" example:"errorDescription"`
}

type boolResponse struct {
	Success bool `json:"success" example:"false"`
	Error   bool `json:"error" example:"true"`
}

type newUserDTO struct {
	Username string `json:"username" example:"jeff" binding:"required"`  // User's username
	Password string `json:"password" example:"guest" binding:"required"` // User's password
	Email    string `json:"email" example:"jeff@jellyf.in"`              // User's email address
	Code     string `json:"code" example:"abc0933jncjkcjj"`              // Invite code (required on /newUser)
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

type generateInviteDTO struct {
	Days          int    `json:"days" example:"1"`                   // Number of days
	Hours         int    `json:"hours" example:"2"`                  // Number of hours
	Minutes       int    `json:"minutes" example:"3"`                // Number of minutes
	UserExpiry    bool   `json:"user-expiry"`                        // Whether or not user expiry is enabled
	UserDays      int    `json:"user-days,omitempty" example:"1"`    // Number of days till user expiry
	UserHours     int    `json:"user-hours,omitempty" example:"2"`   // Number of hours till user expiry
	UserMinutes   int    `json:"user-minutes,omitempty" example:"3"` // Number of minutes till user expiry
	Email         string `json:"email" example:"jeff@jellyf.in"`     // Send invite to this address
	MultipleUses  bool   `json:"multiple-uses" example:"true"`       // Allow multiple uses
	NoLimit       bool   `json:"no-limit" example:"false"`           // No invite use limit
	RemainingUses int    `json:"remaining-uses" example:"5"`         // Remaining invite uses
	Profile       string `json:"profile" example:"DefaultProfile"`   // Name of profile to apply on this invite
	Label         string `json:"label" example:"For Friends"`        // Optional label for the invite
}

type inviteProfileDTO struct {
	Invite  string `json:"invite" example:"slakdaslkdl2342"` // Invite to apply to
	Profile string `json:"profile" example:"DefaultProfile"` // Profile to use
}

type profileDTO struct {
	Admin         bool   `json:"admin" example:"false"`   // Whether profile has admin rights or not
	LibraryAccess string `json:"libraries" example:"all"` // Number of libraries profile has access to
	FromUser      string `json:"fromUser" example:"jeff"` // The user the profile is based on
}

type getProfilesDTO struct {
	Profiles       map[string]profileDTO `json:"profiles"`
	DefaultProfile string                `json:"default_profile"`
}

type profileChangeDTO struct {
	Name string `json:"name" example:"DefaultProfile" binding:"required"` // Name of the profile
}

type newProfileDTO struct {
	Name       string `json:"name" example:"DefaultProfile" binding:"required"`  // Name of the profile
	ID         string `json:"id" example:"kasdjlaskjd342342" binding:"required"` // ID of user to source settings from
	Homescreen bool   `json:"homescreen" example:"true"`                         // Whether to store homescreen layout or not
}

type inviteDTO struct {
	Code           string     `json:"code" example:"sajdlj23423j23"`         // Invite code
	Days           int        `json:"days" example:"1"`                      // Number of days till expiry
	Hours          int        `json:"hours" example:"2"`                     // Number of hours till expiry
	Minutes        int        `json:"minutes" example:"3"`                   // Number of minutes till expiry
	UserExpiry     bool       `json:"user-expiry"`                           // Whether or not user expiry is enabled
	UserDays       int        `json:"user-days,omitempty" example:"1"`       // Number of days till user expiry
	UserHours      int        `json:"user-hours,omitempty" example:"2"`      // Number of hours till user expiry
	UserMinutes    int        `json:"user-minutes,omitempty" example:"3"`    // Number of minutes till user expiry
	Created        string     `json:"created" example:"01/01/20 12:00"`      // Date of creation
	Profile        string     `json:"profile" example:"DefaultProfile"`      // Profile used on this invite
	UsedBy         [][]string `json:"used-by,omitempty"`                     // Users who have used this invite
	NoLimit        bool       `json:"no-limit,omitempty"`                    // If true, invite can be used any number of times
	RemainingUses  int        `json:"remaining-uses,omitempty"`              // Remaining number of uses (if applicable)
	Email          string     `json:"email,omitempty"`                       // Email the invite was sent to (if applicable)
	NotifyExpiry   bool       `json:"notify-expiry,omitempty"`               // Whether to notify the requesting user of expiry or not
	NotifyCreation bool       `json:"notify-creation,omitempty"`             // Whether to notify the requesting user of account creation or not
	Label          string     `json:"label,omitempty" example:"For Friends"` // Optional label for the invite
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
	ID         string `json:"id" example:"fdgsdfg45534fa"`              // userID of user
	Name       string `json:"name" example:"jeff"`                      // Username of user
	Email      string `json:"email,omitempty" example:"jeff@jellyf.in"` // Email address of user (if available)
	LastActive string `json:"last_active"`                              // Time of last activity on Jellyfin
	Admin      bool   `json:"admin" example:"false"`                    // Whether or not the user is Administrator
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

type errorListDTO map[string]map[string]string

type configDTO map[string]interface{}

// Below are for sending config

type meta struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	DependsTrue  string `json:"depends_true,omitempty"`
	DependsFalse string `json:"depends_false,omitempty"`
}

type setting struct {
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	Required        bool        `json:"required"`
	RequiresRestart bool        `json:"requires_restart"`
	Type            string      `json:"type"` // Type (string, number, bool, etc.)
	Value           interface{} `json:"value"`
	Options         [][2]string `json:"options,omitempty"`
	DependsTrue     string      `json:"depends_true,omitempty"`  // If specified, this field is enabled when the specified bool setting is enabled.
	DependsFalse    string      `json:"depends_false,omitempty"` // If specified, opposite behaviour of DependsTrue.
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
	Content   string                 `json:"content"`
	Variables []string               `json:"variables"`
	Values    map[string]interface{} `json:"values"`
	HTML      string                 `json:"html"`
	Plaintext string                 `json:"plaintext"`
}

type getEmailDTO struct {
	Lang string `json:"lang" example:"en-us"` // Language code. If not given, defaults ot one specified in settings.
}
