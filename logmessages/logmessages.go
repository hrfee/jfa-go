package logmessages

const (
	Jellyseerr = "Jellyseerr"
	Jellyfin   = "Jellyfin"
	Ombi       = "Ombi"
	Discord    = "Discord"
	Telegram   = "Telegram"
	Matrix     = "Matrix"
	Email      = "Email"

	// main.go
	FailedLogging = "Failed to start log wrapper: %v\n"

	NoConfig      = "Couldn't find default config file"
	Write         = "Wrote to \"%s\""
	FailedWriting = "Failed to write to \"%s\": %v"
	FailedReading = "Failed to read from \"%s\": %v"
	FailedOpen    = "Failed to open \"%s\": %v"

	CopyConfig       = "Copied default configuration to \"%s\""
	FailedCopyConfig = "Failed to copy default configuration to \"%s\": %v"
	LoadConfig       = "Loaded config file \"%s\""
	FailedLoadConfig = "Failed to load config file \"%s\": %v"

	SocketPath          = "Socket Path: \"%s\""
	FailedSocketConnect = "Couldn't establish socket connection at \"%s\": %v"
	SocketCheckRunning  = "Make sure jfa-go is running."
	FailedSocketRead    = "Couldn't read message on socket \"%s\": %v"
	SocketWrite         = "Command sent."
	FailedSocketWrite   = "Coudln't write message on socket \"%s\": %v"

	FailedLangLoad = "Failed to load language files: %v"

	UsingTLS = "Using TLS/HTTP2"

	UsingOmbi         = "Starting " + " + Ombi + " + " client"
	UsingJellyseerr   = "Starting " + Jellyseerr + " client"
	UsingEmby         = "Using Emby server type (EXPERIMENTAL: PWRs are not available, and support is limited.)"
	UsingJellyfin     = "Using " + Jellyfin + " server type"
	UsingJellyfinAuth = "Using " + Jellyfin + " for authentication"
	UsingLocalAuth    = "Using local username/pw authentication (NOT RECOMMENDED)"

	AuthJellyfin       = "Authenticated with " + Jellyfin + " @ \"%s\""
	FailedAuthJellyfin = "Failed to authenticate with " + Jellyfin + " @ \"%s\" (code %d): %v"

	InitDiscord        = "Initialized Discord daemon"
	FailedInitDiscord  = "Failed to initialize Discord daemon: %v"
	InitTelegram       = "Initialized Telegram daemon"
	FailedInitTelegram = "Failed to initialize Telegram daemon: %v"
	InitMatrix         = "Initialized Matrix daemon"
	FailedInitMatrix   = "Failed to initialize Matrix daemon: %v"

	InitRouter = "Initializing router"
	LoadRoutes = "Loading Routes"

	LoadingSetup = "Loading setup @ \"%s\""
	ServingSetup = "Loaded, visit \"%s\" to start."

	InvalidSSLCert = "Failed loading SSL Certificate \"%s\": %v"
	InvalidSSLKey  = "Failed loading SSL Keyfile \"%s\": %v"

	FailServeSSL = "Failure serving with SSL/TLS: %v"
	FailServe    = "Failure serving: %v"

	Serving = "Loaded @ \"%s\""

	QuitReceived = "Restart/Quit signal received, please be patient."
	Quitting     = "Shutting down..."
	Quit         = "Server shut down."
	FailedQuit   = "Server shutdown failed: %v"

	// api-activities.go
	FailedDBReadActivities = "Failed to read activities from DB: %v"

	// api-backups.go
	IgnoreInvalidFilename = "Invalid filename \"%s\", ignoring: %v"
	GetUpload             = "Retrieved uploaded file \"%s\""
	FailedGetUpload       = "Failed to retrieve file from form data: %v"

	// api-invites.go
	DeleteOldInvite    = "Deleting old invite \"%s\""
	DeleteInvite       = "Deleting invite \"%s\""
	FailedDeleteInvite = "Failed to delete invite \"%s\": %v"
	GenerateInvite     = "Generating new invite"
	InvalidInviteCode  = "Invalid invite code \"%s\""

	FailedSendToTooltipNoUser    = "Failed: \"%s\" not found"
	FailedSendToTooltipMultiUser = "Failed: \"%s\" linked to multiple users"

	FailedParseTime = "Failed to parse time value: %v"

	FailedGetContactMethod = "Failed to get contact method for \"%s\", make sure one is set."

	SetAdminNotify = "Set \"%s\" to %t for admin address \"%s\""

	// api-jellyseerr.go
	FailedGetUsers = "Failed to get user(s) from %s: %v"
	// FIXME: Once done, look back at uses of FailedGetUsers for places where this would make more sense.
	FailedGetUser                        = "Failed to get user \"%s\" from %s: %v"
	FailedGetJellyseerrNotificationPrefs = "Failed to get user's notification prefs from " + Jellyseerr + ": %v"
	FailedSyncContactMethods             = "Failed to sync contact methods with %s: %v"

	// api-messages.go
	FailedGetCustomMessage   = "Failed to get custom message \"%s\""
	SetContactPrefForService = "Set contact preference for %s (\"%s\"): %t"

	// Matrix
	InvalidPIN          = "Invalid PIN \"%s\""
	UnauthorizedPIN     = "Unauthorized PIN \"%s\""
	FailedCreateRoom    = "Failed to create room: %v"
	FailedGenerateToken = "Failed to generate token: %v"

	// api-profiles.go
	SetDefaultProfile = "Setting default profile to \"%s\""

	FailedApplyProfile            = "Failed to apply profile for %s user \"%s\": %v"
	ApplyProfile                  = "Applying settings from profile \"%s\""
	FailedGetProfile              = "Failed to find profile \"%s\""
	FailedApplyTemplate           = "Failed to apply %s template for %s user \"%s\": %v"
	FallbackToDefault             = ", using default"
	CreateProfileFromUser         = "Creating profile from user \"%s\""
	FailedGetJellyfinDisplayPrefs = "Failed to get DisplayPreferences for user \"%s\" from " + Jellyfin + ": %v"
	ProfileNoHomescreen           = "No homescreen template in profile \"%s\""
	Profile                       = "profile"
	User                          = "user"
	ApplyingTemplatesFrom         = "Applying templates from %s: \"%s\" to %d users"
	DelayingRequests              = "Delay will be added between requests (count = %d)"

	// api-userpage.go
	EmailConfirmationRequired = "User \"%s\" requires email confirmation"

	ChangePassword       = "Changed password for %s user \"%s\""
	FailedChangePassword = "Failed to change password for %s user \"%s\": %v"

	GetReferralTemplate       = "Found referral template \"%s\""
	FailedGetReferralTemplate = "Failed to find referral template \"%s\": %v"
	DeleteOldReferral         = "Deleting old referral \"%s\""
	RenewOldReferral          = "Renewing old referral \"%s\""

	// api-users.go
	CreateUser                 = "Created %s user \"%s\""
	FailedCreateUser           = "Failed to create new %s user \"%s\": %v"
	LinkUser                   = "Linked %s user \"%s\""
	FailedLinkUser             = "Failed to link %s user \"%s\" with \"%s\": %v"
	DeleteUser                 = "Deleted %s user \"%s\""
	FailedDeleteUser           = "Failed to delete %s user \"%s\": %v"
	FailedDeleteUsers          = "Failed to delete %s user(s): %v"
	UserExists                 = "user already exists"
	AccountLinked              = "account already linked and require_unique enabled"
	AccountUnverified          = "unverified"
	FailedSetDiscordMemberRole = "Failed to set " + Discord + " member role: %v"

	FailedSetEmailAddress = "Failed to set email address for %s user \"%s\": %v"

	AdditionalOmbiErrors = "Additional errors from " + Ombi + ": %v"

	IncorrectCaptcha = "captcha incorrect"

	ExtendCreateExpiry = "Extended or created expiry for user \"%s\""

	UserEmailAdjusted = "Email for user \"%s\" adjusted"
	UserAdminAdjusted = "Admin state for user \"%s\" set to %t"
	UserLabelAdjusted = "Label for user \"%s\" set to \"%s\""
)

const (
	FailedGetCookies = "Failed to get cookie(s) \"%s\": %v"
	FailedParseJWT   = "Failed to parse JWT: %v"
	FailedCastJWT    = "JWT claims unreadable"
	InvalidJWT       = "JWT was invalidated, of incorrect type or has expired"
	FailedSignJWT    = "Failed to sign JWT: %v"
)

const (
	FailedConstructExpiryAdmin = "Failed to construct expiry notification for \"%s\": %v"
	FailedSendExpiryAdmin      = "Failed to send expiry notification for \"%s\" to \"%s\": %v"
	SentExpiryAdmin            = "Sent expiry notification for \"%s\" to \"%s\""

	FailedConstructCreationAdmin = "Failed to construct creation notification for \"%s\": %v"
	FailedSendCreationAdmin      = "Failed to send creation notification for \"%s\" to \"%s\": %v"
	SentCreationAdmin            = "Sent creation notification for \"%s\" to \"%s\""

	FailedConstructInviteMessage = "Failed to construct invite message for \"%s\": %v"
	FailedSendInviteMessage      = "Failed to send invite message for \"%s\" to \"%s\": %v"
	SentInviteMessage            = "Sent invite message for \"%s\" to \"%s\""

	FailedConstructConfirmationEmail = "Failed to construct confirmation email for \"%s\": %v"
	FailedSendConfirmationEmail      = "Failed to send confirmation email for \"%s\" to \"%s\": %v"
	SentConfirmationEmail            = "Sent confirmation email for \"%s\" to \"%s\""

	FailedConstructPWRMessage = "Failed to construct PWR message for \"%s\": %v"
	FailedSendPWRMessage      = "Failed to send PWR message for \"%s\" to \"%s\": %v"
	SentPWRMessage            = "Sent PWR message for \"%s\" to \"%s\""

	FailedConstructWelcomeMessage = "Failed to construct welcome message for \"%s\": %v"
	FailedSendWelcomeMessage      = "Failed to send welcome message for \"%s\" to \"%s\": %v"
	SentWelcomeMessage            = "Sent welcome message for \"%s\" to \"%s\""

	FailedConstructEnableDisableMessage = "Failed to construct enable/disable message for \"%s\": %v"
	FailedSendEnableDisableMessage      = "Failed to send enable/disable message for \"%s\" to \"%s\": %v"
	SentEnableDisableMessage            = "Sent enable/disable message for \"%s\" to \"%s\""

	FailedConstructDeletionMessage = "Failed to construct account deletion message for \"%s\": %v"
	FailedSendDeletionMessage      = "Failed to send account deletion message for \"%s\" to \"%s\": %v"
	SentDeletionMessage            = "Sent account deletion message for \"%s\" to \"%s\""

	FailedConstructExpiryAdjustmentMessage = "Failed to construct expiry adjustment message for \"%s\": %v"
	FailedSendExpiryAdjustmentMessage      = "Failed to send expiry adjustment message for \"%s\" to \"%s\": %v"
	SentExpiryAdjustmentMessage            = "Sent expiry adjustment message for \"%s\" to \"%s\""

	FailedConstructAnnouncementMessage = "Failed to construct announcement message for \"%s\": %v"
	FailedSendAnnouncementMessage      = "Failed to send announcement message for \"%s\" to \"%s\": %v"
	SentAnnouncementMessage            = "Sent announcement message for \"%s\" to \"%s\""
)
