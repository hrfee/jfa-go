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

type deleteUserDTO struct {
	Users  []string `json:"users" binding:"required"` // List of usernames to delete
	Notify bool     `json:"notify"`                   // Whether to notify users of deletion
	Reason string   `json:"reason"`                   // Account deletion reason (for notification)
}

type generateInviteDTO struct {
	Days          int    `json:"days" example:"1"`                 // Number of days
	Hours         int    `json:"hours" example:"2"`                // Number of hours
	Minutes       int    `json:"minutes" example:"3"`              // Number of minutes
	Email         string `json:"email" example:"jeff@jellyf.in"`   // Send invite to this address
	MultipleUses  bool   `json:"multiple-uses" example:"true"`     // Allow multiple uses
	NoLimit       bool   `json:"no-limit" example:"false"`         // No invite use limit
	RemainingUses int    `json:"remaining-uses" example:"5"`       // Remaining invite uses
	Profile       string `json:"profile" example:"DefaultProfile"` // Name of profile to apply on this invite
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
	Code           string     `json:"code" example:"sajdlj23423j23"`    // Invite code
	Days           int        `json:"days" example:"1"`                 // Number of days till expiry
	Hours          int        `json:"hours" example:"2"`                // Number of hours till expiry
	Minutes        int        `json:"minutes" example:"3"`              // Number of minutes till expiry
	Created        string     `json:"created" example:"01/01/20 12:00"` // Date of creation
	Profile        string     `json:"profile" example:"DefaultProfile"` // Profile used on this invite
	UsedBy         [][]string `json:"used-by,omitempty"`                // Users who have used this invite
	NoLimit        bool       `json:"no-limit,omitempty"`               // If true, invite can be used any number of times
	RemainingUses  int        `json:"remaining-uses,omitempty"`         // Remaining number of uses (if applicable)
	Email          string     `json:"email,omitempty"`                  // Email the invite was sent to (if applicable)
	NotifyExpiry   bool       `json:"notify-expiry,omitempty"`          // Whether to notify the requesting user of expiry or not
	NotifyCreation bool       `json:"notify-creation,omitempty"`        // Whether to notify the requesting user of account creation or not
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

type errorListDTO map[string]map[string]string

type configDTO map[string]interface{}

