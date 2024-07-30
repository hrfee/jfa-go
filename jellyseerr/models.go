package jellyseerr

import "time"

type UserField string

const (
	FieldDisplayName UserField = "displayName"
	FieldEmail       UserField = "email"
)

type User struct {
	UserTemplate                         // Note: You can set this with User.UserTemplate = value.
	UserType                   int64     `json:"userType,omitempty"`
	Warnings                   []any     `json:"warnings,omitempty"`
	ID                         int64     `json:"id,omitempty"`
	Email                      string    `json:"email,omitempty"`
	PlexUsername               string    `json:"plexUsername,omitempty"`
	JellyfinUsername           string    `json:"jellyfinUsername,omitempty"`
	Username                   string    `json:"username,omitempty"`
	RecoveryLinkExpirationDate any       `json:"recoveryLinkExpirationDate,omitempty"`
	PlexID                     string    `json:"plexId,omitempty"`
	JellyfinUserID             string    `json:"jellyfinUserId,omitempty"`
	JellyfinDeviceID           string    `json:"jellyfinDeviceId,omitempty"`
	JellyfinAuthToken          string    `json:"jellyfinAuthToken,omitempty"`
	PlexToken                  string    `json:"plexToken,omitempty"`
	Avatar                     string    `json:"avatar,omitempty"`
	CreatedAt                  time.Time `json:"createdAt,omitempty"`
	UpdatedAt                  time.Time `json:"updatedAt,omitempty"`
	RequestCount               int64     `json:"requestCount,omitempty"`
	DisplayName                string    `json:"displayName,omitempty"`
}

func (u User) Name() string {
	var n string
	if u.Username != "" {
		n = u.Username
	} else if u.JellyfinUsername != "" {
		n = u.JellyfinUsername
	}
	if u.DisplayName != "" {
		n += " (" + u.DisplayName + ")"
	}
	return n
}

type UserTemplate struct {
	Permissions     Permissions `json:"permissions,omitempty"`
	MovieQuotaLimit any         `json:"movieQuotaLimit,omitempty"`
	MovieQuotaDays  any         `json:"movieQuotaDays,omitempty"`
	TvQuotaLimit    any         `json:"tvQuotaLimit,omitempty"`
	TvQuotaDays     any         `json:"tvQuotaDays,omitempty"`
}

type PageInfo struct {
	Pages    int `json:"pages,omitempty"`
	PageSize int `json:"pageSize,omitempty"`
	Results  int `json:"results,omitempty"`
	Page     int `json:"page,omitempty"`
}

type GetUsersDTO struct {
	Page    PageInfo `json:"pageInfo,omitempty"`
	Results []User   `json:"results,omitempty"`
}

type permissionsDTO struct {
	Permissions Permissions `json:"permissions,omitempty"`
}

type Permissions int

type NotificationTypes struct {
	Discord    int64 `json:"discord,omitempty"`
	Email      int64 `json:"email,omitempty"`
	Pushbullet int64 `json:"pushbullet,omitempty"`
	Pushover   int64 `json:"pushover,omitempty"`
	Slack      int64 `json:"slack,omitempty"`
	Telegram   int64 `json:"telegram,omitempty"`
	Webhook    int64 `json:"webhook,omitempty"`
	Webpush    int64 `json:"webpush,omitempty"`
}

func (nt *NotificationTypes) Empty() bool {
	return nt.Discord == 0 && nt.Email == 0 && nt.Pushbullet == 0 && nt.Pushover == 0 && nt.Slack == 0 && nt.Telegram == 0 && nt.Webhook == 0 && nt.Webpush == 0
}

type NotificationsField string

const (
	FieldDiscord         NotificationsField = "discordId"
	FieldTelegram        NotificationsField = "telegramChatId"
	FieldEmailEnabled    NotificationsField = "emailEnabled"
	FieldDiscordEnabled  NotificationsField = "discordEnabled"
	FieldTelegramEnabled NotificationsField = "telegramEnabled"
)

type Notifications struct {
	NotificationsTemplate
	PgpKey                   any    `json:"pgpKey,omitempty"`
	DiscordID                string `json:"discordId,omitempty"`
	PushbulletAccessToken    any    `json:"pushbulletAccessToken,omitempty"`
	PushoverApplicationToken any    `json:"pushoverApplicationToken,omitempty"`
	PushoverUserKey          any    `json:"pushoverUserKey,omitempty"`
	TelegramChatID           string `json:"telegramChatId,omitempty"`
}

type NotificationsTemplate struct {
	EmailEnabled         bool               `json:"emailEnabled,omitempty"`
	DiscordEnabled       bool               `json:"discordEnabled,omitempty"`
	DiscordEnabledTypes  int64              `json:"discordEnabledTypes,omitempty"`
	PushoverSound        any                `json:"pushoverSound,omitempty"`
	TelegramEnabled      bool               `json:"telegramEnabled,omitempty"`
	TelegramSendSilently any                `json:"telegramSendSilently,omitempty"`
	WebPushEnabled       bool               `json:"webPushEnabled,omitempty"`
	NotifTypes           *NotificationTypes `json:"notificationTypes,omitempty"`
}
