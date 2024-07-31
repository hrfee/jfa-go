package logmessages

const (
	FailedLogging = "Failed to start log wrapper: %v\n"

	NoConfig      = "Couldn't find default config file"
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

	UsingOmbi         = "Starting Ombi client"
	UsingJellyseerr   = "Starting Jellyseerr client"
	UsingEmby         = "Using Emby server type (EXPERIMENTAL: PWRs are not available, and support is limited.)"
	UsingJellyfin     = "Using Jellyfin server type"
	UsingJellyfinAuth = "Using Jellyfin for authentication"
	UsingLocalAuth    = "Using local username/pw authentication (NOT RECOMMENDED)"

	AuthJellyfin       = "Authenticated with Jellyfin @ \"%s\""
	FailedAuthJellyfin = "Failed to authenticate with Jellyfin @ \"%s\" (code %d): %v"

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
)
