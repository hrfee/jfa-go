package mediabrowser

import (
	"time"
)

type magicParse struct {
	Parsed time.Time `json:"parseme"`
}

// Time embeds time.Time with a custom JSON Unmarshal method to work with Jellyfin & Emby's time formatting.
type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(b []byte) (err error) {
	// Trim quotes from beginning and end, and any number of Zs (indicates UTC).
	for b[0] == '"' {
		b = b[1:]
	}
	for b[len(b)-1] == '"' || b[len(b)-1] == 'Z' {
		b = b[:len(b)-1]
	}
	// Trim nanoseconds and anything after, we don't care
	i := len(b) - 1
	for b[i] != '.' && i > 0 {
		i--
	}
	if i != 0 {
		b = b[:i]
	}
	t.Time, err = time.Parse("2006-01-02T15:04:05", string(b))
	return
}

type User struct {
	Name                      string        `json:"Name"`
	ServerID                  string        `json:"ServerId"`
	ID                        string        `json:"Id"`
	HasPassword               bool          `json:"HasPassword"`
	HasConfiguredPassword     bool          `json:"HasConfiguredPassword"`
	HasConfiguredEasyPassword bool          `json:"HasConfiguredEasyPassword"`
	EnableAutoLogin           bool          `json:"EnableAutoLogin"`
	LastLoginDate             Time          `json:"LastLoginDate"`
	LastActivityDate          Time          `json:"LastActivityDate"`
	Configuration             Configuration `json:"Configuration"`
	// Policy stores the user's permissions.
	Policy Policy `json:"Policy"`
}

type SessionInfo struct {
	RemoteEndpoint string `json:"RemoteEndPoint"`
	UserID         string `json:"UserId"`
}

type AuthenticationResult struct {
	User        User        `json:"User"`
	AccessToken string      `json:"AccessToken"`
	ServerID    string      `json:"ServerId"`
	SessionInfo SessionInfo `json:"SessionInfo"`
}

type Configuration struct {
	PlayDefaultAudioTrack      bool          `json:"PlayDefaultAudioTrack"`
	SubtitleLanguagePreference string        `json:"SubtitleLanguagePreference"`
	DisplayMissingEpisodes     bool          `json:"DisplayMissingEpisodes"`
	GroupedFolders             []interface{} `json:"GroupedFolders"`
	SubtitleMode               string        `json:"SubtitleMode"`
	DisplayCollectionsView     bool          `json:"DisplayCollectionsView"`
	EnableLocalPassword        bool          `json:"EnableLocalPassword"`
	OrderedViews               []interface{} `json:"OrderedViews"`
	LatestItemsExcludes        []interface{} `json:"LatestItemsExcludes"`
	MyMediaExcludes            []interface{} `json:"MyMediaExcludes"`
	HidePlayedInLatest         bool          `json:"HidePlayedInLatest"`
	RememberAudioSelections    bool          `json:"RememberAudioSelections"`
	RememberSubtitleSelections bool          `json:"RememberSubtitleSelections"`
	EnableNextEpisodeAutoPlay  bool          `json:"EnableNextEpisodeAutoPlay"`
}
type Policy struct {
	IsAdministrator                  bool          `json:"IsAdministrator"`
	IsHidden                         bool          `json:"IsHidden"`
	IsDisabled                       bool          `json:"IsDisabled"`
	BlockedTags                      []interface{} `json:"BlockedTags"`
	EnableUserPreferenceAccess       bool          `json:"EnableUserPreferenceAccess"`
	AccessSchedules                  []interface{} `json:"AccessSchedules"`
	BlockUnratedItems                []interface{} `json:"BlockUnratedItems"`
	EnableRemoteControlOfOtherUsers  bool          `json:"EnableRemoteControlOfOtherUsers"`
	EnableSharedDeviceControl        bool          `json:"EnableSharedDeviceControl"`
	EnableRemoteAccess               bool          `json:"EnableRemoteAccess"`
	EnableLiveTvManagement           bool          `json:"EnableLiveTvManagement"`
	EnableLiveTvAccess               bool          `json:"EnableLiveTvAccess"`
	EnableMediaPlayback              bool          `json:"EnableMediaPlayback"`
	EnableAudioPlaybackTranscoding   bool          `json:"EnableAudioPlaybackTranscoding"`
	EnableVideoPlaybackTranscoding   bool          `json:"EnableVideoPlaybackTranscoding"`
	EnablePlaybackRemuxing           bool          `json:"EnablePlaybackRemuxing"`
	ForceRemoteSourceTranscoding     bool          `json:"ForceRemoteSourceTranscoding"`
	EnableContentDeletion            bool          `json:"EnableContentDeletion"`
	EnableContentDeletionFromFolders []interface{} `json:"EnableContentDeletionFromFolders"`
	EnableContentDownloading         bool          `json:"EnableContentDownloading"`
	EnableSyncTranscoding            bool          `json:"EnableSyncTranscoding"`
	EnableMediaConversion            bool          `json:"EnableMediaConversion"`
	EnabledDevices                   []interface{} `json:"EnabledDevices"`
	EnableAllDevices                 bool          `json:"EnableAllDevices"`
	EnabledChannels                  []interface{} `json:"EnabledChannels"`
	EnableAllChannels                bool          `json:"EnableAllChannels"`
	EnabledFolders                   []string      `json:"EnabledFolders"`
	EnableAllFolders                 bool          `json:"EnableAllFolders"`
	InvalidLoginAttemptCount         int           `json:"InvalidLoginAttemptCount"`
	LoginAttemptsBeforeLockout       int           `json:"LoginAttemptsBeforeLockout"`
	MaxActiveSessions                int           `json:"MaxActiveSessions"`
	EnablePublicSharing              bool          `json:"EnablePublicSharing"`
	BlockedMediaFolders              []interface{} `json:"BlockedMediaFolders"`
	BlockedChannels                  []interface{} `json:"BlockedChannels"`
	RemoteClientBitrateLimit         int           `json:"RemoteClientBitrateLimit"`
	AuthenticationProviderID         string        `json:"AuthenticationProviderId"`
	PasswordResetProviderID          string        `json:"PasswordResetProviderId"`
	SyncPlayAccess                   string        `json:"SyncPlayAccess"`
}
