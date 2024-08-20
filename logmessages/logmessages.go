package logmessages

/* Log strings for (almost) all the program.
 * Helps avoid writing redundant, slightly different
 * strings constantly.
 * Also would help if I were to ever set up translation
 * for logs. Mostly split by file, but obviously there's
 * re-use, and occasionally related stuff is grouped.
 */
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

	NoConfig        = "Couldn't find default config file"
	Write           = "Wrote to \"%s\""
	FailedWriting   = "Failed to write to \"%s\": %v"
	FailedCreateDir = "Failed to create directory \"%s\": %v"
	FailedReading   = "Failed to read from \"%s\": %v"
	FailedOpen      = "Failed to open \"%s\": %v"
	FailedStat      = "Failed to stat \"%s\": %v"
	PathNotFound    = "Path \"%s\" not found"

	CopyConfig       = "Copied default configuration to \"%s\""
	FailedCopyConfig = "Failed to copy default configuration to \"%s\": %v"
	LoadConfig       = "Loaded config file \"%s\""
	FailedLoadConfig = "Failed to load config file \"%s\": %v"
	ModifyConfig     = "Config saved to \"%s\""

	SocketPath          = "Socket Path: \"%s\""
	FailedSocketConnect = "Couldn't establish socket connection at \"%s\": %v"
	SocketCheckRunning  = "Make sure jfa-go is running."
	FailedSocketRead    = "Couldn't read message on socket \"%s\": %v"
	SocketWrite         = "Command sent."
	FailedSocketWrite   = "Coudln't write message on socket \"%s\": %v"

	FailedLangLoad = "Failed to load language files: %v"

	UsingTLS = "Using TLS/HTTP2"

	UsingOmbi         = "Starting " + Ombi + " client"
	UsingJellyseerr   = "Starting " + Jellyseerr + " client"
	UsingEmby         = "Using Emby server type (EXPERIMENTAL: PWRs are not available, and support is limited.)"
	UsingJellyfin     = "Using " + Jellyfin + " server type"
	UsingJellyfinAuth = "Using " + Jellyfin + " for authentication"
	UsingLocalAuth    = "Using local username/pw authentication (NOT RECOMMENDED)"

	AuthJellyfin       = "Authenticated with " + Jellyfin + " @ \"%s\""
	FailedAuthJellyfin = "Failed to authenticate with " + Jellyfin + " @ \"%s\" (code %d): %v"
	FailedAuth         = "Failed to authenticate with %s @ \"%s\" (code %d): %v"

	Unauthorized          = "unauthorized, check credentials"
	Forbidden             = "forbidden, the user may not have correct permissions"
	NotFound              = "not found"
	TimedOut              = "timed out"
	FailedGenericWithCode = "failed (code %d)"

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

	QuitReceived             = "Restart/Quit signal received, please be patient."
	Quitting                 = "Shutting down..."
	Restarting               = "Restarting..."
	FailedHardRestartWindows = "hard restarts not available on windows"
	Quit                     = "Server shut down."
	FailedQuit               = "Server shutdown failed: %v"

	// api-activities.go
	FailedDBReadActivities = "Failed to read activities from DB: %v"

	// api-backups.go
	IgnoreInvalidFilename = "Invalid filename \"%s\", ignoring: %v"
	GetUpload             = "Retrieved uploaded file \"%s\""
	FailedGetUpload       = "Failed to retrieve file from form data: %v"

	// api-invites.go
	DeleteOldInvite      = "Deleting old invite \"%s\""
	DeleteInvite         = "Deleting invite \"%s\""
	FailedDeleteInvite   = "Failed to delete invite \"%s\": %v"
	GenerateInvite       = "Generating new invite"
	FailedGenerateInvite = "Failed to generate new invite: %v"
	InvalidInviteCode    = "Invalid invite code \"%s\""

	FailedSendToTooltipNoUser    = "Failed: \"%s\" not found"
	FailedSendToTooltipMultiUser = "Failed: \"%s\" linked to multiple users"

	FailedParseTime = "Failed to parse time value: %v"

	FailedGetContactMethod = "Failed to get contact method for \"%s\", make sure one is set."

	SetAdminNotify = "Set \"%s\" to %t for admin address \"%s\""

	// *jellyseerr*.go
	FailedGetUsers = "Failed to get user(s) from %s: %v"
	// FIXME: Once done, look back at uses of FailedGetUsers for places where this would make more sense.
	FailedGetUser                        = "Failed to get user \"%s\" from %s: %v"
	FailedGetJellyseerrNotificationPrefs = "Failed to get user \"%s\"'s notification prefs from " + Jellyseerr + ": %v"
	FailedSyncContactMethods             = "Failed to sync contact methods with %s: %v"
	ImportJellyseerrUser                 = "Triggered import for " + Jellyseerr + " user \"%s\" (New ID: %d)"
	FailedImportUser                     = "Failed to get or trigger import for %s user \"%s\": %v"

	// api-messages.go
	FailedGetCustomMessage   = "Failed to get custom message \"%s\""
	SetContactPrefForService = "Set contact preference for %s (\"%s\"): %t"

	// Matrix
	InvalidPIN       = "Invalid PIN \"%s\""
	ExpiredPIN       = "Expired PIN \"%s\""
	InvalidPassword  = "Invalid Password"
	UnauthorizedPIN  = "Unauthorized PIN \"%s\""
	FailedCreateRoom = "Failed to create room: %v"

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
	Lang                          = "language"
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
	FailedSetDiscordMemberRole = "Failed to apply/remove " + Discord + " member role: %v"

	FailedSetEmailAddress = "Failed to set email address for %s user \"%s\": %v"

	AdditionalErrors = "Additional errors from %s: %v"

	IncorrectCaptcha = "captcha incorrect"

	ExtendCreateExpiry = "Extended or created expiry for user \"%s\""

	UserEmailAdjusted = "Email for user \"%s\" adjusted"
	UserAdminAdjusted = "Admin state for user \"%s\" set to %t"
	UserLabelAdjusted = "Label for user \"%s\" set to \"%s\""

	// api.go
	ApplyUpdate       = "Applied update"
	FailedApplyUpdate = "Failed to apply update: %v"
	UpdateManual      = "update is manual"

	// backups.go
	DeleteOldBackup       = "Deleted old backup \"%s\""
	FailedDeleteOldBackup = "Failed to delete old backup \"%s\": %v"
	CreateBackup          = "Created database backup \"%+v\""
	FailedCreateBackup    = "Faled to create database backup: %v"
	MoveOldDB             = "Moved existing database to \"%s\""
	FailedMoveOldDB       = "Failed to move existing database to \"%s\": %v"
	RestoreDB             = "Restored database from \"%s\""
	FailedRestoreDB       = "Failed to resotre database from \"%s\": %v"

	// config.go
	EnableAllPWRMethods = "No PWR method preferences set in [user_page], all will be enabled"
	InitProxy           = "Initialized proxy @ \"%s\""
	FailedInitProxy     = "Failed to initialize proxy @ \"%s\": %v\nStartup will pause for a bit to grab your attention."
	NoURLSuffix         = `Warning: Given "jfa_url"/"External jfa-go URL" value does not include "url_base" value!`
	BadURLBase          = `Warning: Given URL Base "%s" may conflict with the applications subpaths.`
	NoExternalHost      = `No "External jfa-go URL" provided, set one in Settings > General.`
	LoginWontSave       = ` Your login won't save until you do.`

	// discord.go
	StartDaemon                      = "Started %s daemon"
	FailedStartDaemon                = "Failed to start %s daemon: %v"
	FailedGetDiscordGuildMembers     = "Failed to get " + Discord + " guild members: %v"
	FailedGetDiscordGuild            = "Failed to get " + Discord + " guild: %v"
	FailedGetDiscordRoles            = "Failed to get " + Discord + " roles: %v"
	FailedCreateDiscordInviteChannel = "Failed to create " + Discord + " invite channel: %v"
	InviteChannelEmpty               = "no invite channel set in settings"
	FailedGetDiscordChannels         = "Failed to get " + Discord + " channel(s): %v"
	FailedGetDiscordChannel          = "Failed to get " + Discord + " channel \"%s\": %v"
	MonitorAllDiscordChannels        = "Will monitor all " + Discord + " channels"
	FailedCreateDiscordDMChannel     = "Failed to create " + Discord + " private DM channel with \"%s\": %v"
	RegisterDiscordChoice            = "Registered " + Discord + " %s choice \"%s\""
	FailedRegisterDiscordChoices     = "Failed to register " + Discord + " %s choices: %v"
	FailedDeregDiscordChoice         = "Failed to deregister " + Discord + " %s choice \"%s\": %v"
	RegisterDiscordCommand           = "Registered " + Discord + " command \"%s\""
	FailedRegisterDiscordCommand     = "Failed to register " + Discord + " command \"%s\": %v"
	FailedGetDiscordCommands         = "Failed to get " + Discord + " commands: %v"
	FailedDeregDiscordCommand        = "Failed to deregister " + Discord + " command \"%s\": %v"

	FailedReply   = "Failed to reply to %s message from \"%s\": %v"
	FailedMessage = "Failed to send %s message to \"%s\": %v"

	IgnoreOutOfChannelMessage = "Ignoring out-of-channel %s message"

	FailedGenerateDiscordInvite = "Failed to generate " + Discord + " invite: %v"

	// email.go
	FailedInitSMTP        = "Failed to initialize SMTP mailer: %v"
	FailedGeneratePWRLink = "Failed to generate PWR link: %v"

	// housekeeping-d.go
	hk                   = "Housekeeping: "
	hkcu                 = hk + "cleaning up "
	HousekeepingEmail    = hkcu + Email + " addresses"
	HousekeepingDiscord  = hkcu + Discord + " IDs"
	HousekeepingTelegram = hkcu + Telegram + " IDs"
	HousekeepingMatrix   = hkcu + Matrix + " IDs"
	HousekeepingCaptcha  = hkcu + "PWR Captchas"
	HousekeepingActivity = hkcu + "Activity log"
	HousekeepingInvites  = hkcu + "Invites"
	ActivityLogTxnTooBig = hk + "Activity log delete transaction was too big, going one-by-one"

	// matrix*.go
	FailedSyncMatrix             = "Failed to sync " + Matrix + " daemon: %v"
	FailedCreateMatrixRoom       = "Failed to create " + Matrix + " room with user \"%s\": %v"
	MatrixOLMLog                 = "Matrix/OLM: %v"
	MatrixOLMTraceLog            = "Matrix/OLM [TRACE]:"
	FailedDecryptMatrixMessage   = "Failed to decrypt " + Matrix + " E2EE'd message: %v"
	FailedEnableMatrixEncryption = "Failed to enable encryption in " + Matrix + " room \"%s\": %v"

	// NOTE: "migrations.go" is the one file where log messages are not part of logmessages/logmessages.go.

	// pwreset.go
	PWRExpired = "PWR for user \"%s\" already expired @ %s, check system time!"

	// router.go
	UseDefaultHTML      = "Using default HTML \"%s\""
	UseCustomHTML       = "Using custom HTML \"%s\""
	FailedLoadTemplates = "Failed to load %s templates: %v"
	Internal            = "internal"
	External            = "external"
	RegisterPprof       = "Registered pprof"
	SwaggerWarning      = "Warning: Swagger should not be used on a public instance."

	// storage.go
	ConnectDB       = "Connected to DB \"%s\""
	FailedConnectDB = "Failed to open/connect to database \"%s\": %v"

	// updater.go
	NoUpdate           = "No new updates available"
	FoundUpdate        = "Found update"
	FailedGetUpdateTag = "Failed to get latest tag: %v"
	FailedGetUpdate    = "Failed to get update: %v"
	UpdateTagDetails   = "Update/Tag details: %+v"

	// user-auth.go
	UserPage                     = "userpage"
	UserPageRequiresJellyfinAuth = "Jellyfin login must be enabled for user page access."

	// user-d.go
	CheckUserExpiries                = "Checking for user expiry"
	DeleteExpiryForOldUser           = "Deleting expiry for old user \"%s\""
	DeleteExpiredUser                = "Deleting expired user \"%s\""
	DisableExpiredUser               = "Disabling expired user \"%s\""
	FailedDeleteOrDisableExpiredUser = "Failed to delete/disable expired user \"%s\": %v"

	// views.go
	FailedServerPush      = "Failed to use HTTP/2 Server Push: %v"
	IgnoreBotPWR          = "Ignore PWR magic link visit from bot"
	ReCAPTCHA             = "ReCAPTCHA"
	FailedGenerateCaptcha = "Failed to generate captcha: %v"
	CaptchaNotFound       = "Captcha \"%s\" not found in invite \"%s\""
	FailedVerifyReCAPTCHA = "Failed to verify reCAPTCHA: %v"
	InvalidHostname       = "invalid hostname (wanted \"%s\", got \"%s\")"

	// webhooks.go
	WebhookRequest = "Webhook request send to \"%s\" (%d): %v"
)

const (
	FailedGetCookies      = "Failed to get cookie(s) \"%s\": %v"
	FailedParseJWT        = "Failed to parse JWT: %v"
	FailedCastJWT         = "JWT claims unreadable"
	InvalidJWT            = "JWT was invalidated, of incorrect type or has expired"
	LocallyInvalidatedJWT = "JWT is listed as invalidated"
	FailedSignJWT         = "Failed to sign JWT: %v"

	RequestingToken       = "Token requested (%s)"
	TokenLoginAttempt     = "login attempt"
	TokenRefresh          = "refresh token"
	UserTokenLoginAttempt = UserPage + " " + TokenLoginAttempt
	UserTokenRefresh      = UserPage + " " + TokenRefresh
	GenerateToken         = "Token generated for user \"%s\""
	FailedGenerateToken   = "Failed to generate token: %v"

	FailedAuthRequest = "Failed to authorize request: %v"
	InvalidAuthHeader = "invalid auth header"
	NonAdminToken     = "token not for admin use"
	NonAdminUser      = "user \"%s\" not admin"
	InvalidUserOrPass = "invalid user/pass"
	EmptyUserOrPass   = "invalid user/pass"
	UserDisabled      = "user is disabled"
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

	FailedConstructExpiryMessage = "Failed to construct expiry message for \"%s\": %v"
	FailedSendExpiryMessage      = "Failed to send expiry message for \"%s\" to \"%s\": %v"
	SentExpiryMessage            = "Sent expiry message for \"%s\" to \"%s\""

	FailedConstructAnnouncementMessage = "Failed to construct announcement message for \"%s\": %v"
	FailedSendAnnouncementMessage      = "Failed to send announcement message for \"%s\" to \"%s\": %v"
	SentAnnouncementMessage            = "Sent announcement message for \"%s\" to \"%s\""
)
