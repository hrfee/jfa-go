package mediabrowser

import (
	"encoding/json"
	"fmt"
	"strings"
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
	str := strings.TrimSuffix(strings.TrimPrefix(string(b), "\""), "\"")
	// Trim nanoseconds to always have 6 digits, so overall length is always the same.
	if str[len(str)-1] == 'Z' {
		str = str[:26] + "Z"
	} else {
		str = str[:26]
	}
	// decent method
	t.Time, err = time.Parse("2006-01-02T15:04:05.000000Z", str)
	if err == nil {
		return
	}
	t.Time, err = time.Parse("2006-01-02T15:04:05.000000", str)
	if err == nil {
		return
	}
	// emby method
	t.Time, err = time.Parse("2006-01-02T15:04:05.0000000+00:00", str)
	if err == nil {
		return
	}
	fmt.Println("THIRDERR", err)
	// magic method
	// some stored dates from jellyfin have no timezone at the end, if not we assume UTC
	if str[len(str)-1] != 'Z' {
		str += "Z"
	}
	timeJSON := []byte("{ \"parseme\": \"" + str + "\" }")
	var parsed magicParse
	// Magically turn it into a time.Time
	err = json.Unmarshal(timeJSON, &parsed)
	t.Time = parsed.Parsed
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
