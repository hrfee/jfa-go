package jellyseerr

import "time"

type UserField string

const (
	FieldDisplayName UserField = "displayName"
	FieldEmail       UserField = "email"
)

type User struct {
	UserTemplate                         // Note: You can set this with User.UserTemplate = value.
	Warnings                   []any     `json:"warnings"`
	ID                         int       `json:"id"`
	Email                      string    `json:"email"`
	PlexUsername               string    `json:"plexUsername"`
	JellyfinUsername           string    `json:"jellyfinUsername"`
	Username                   string    `json:"username"`
	RecoveryLinkExpirationDate any       `json:"recoveryLinkExpirationDate"`
	PlexID                     string    `json:"plexId"`
	JellyfinUserID             string    `json:"jellyfinUserId"`
	JellyfinDeviceID           string    `json:"jellyfinDeviceId"`
	JellyfinAuthToken          string    `json:"jellyfinAuthToken"`
	PlexToken                  string    `json:"plexToken"`
	Avatar                     string    `json:"avatar"`
	CreatedAt                  time.Time `json:"createdAt"`
	UpdatedAt                  time.Time `json:"updatedAt"`
	RequestCount               int       `json:"requestCount"`
	DisplayName                string    `json:"displayName"`
}

type UserTemplate struct {
	Permissions     Permissions `json:"permissions"`
	UserType        int         `json:"userType"`
	MovieQuotaLimit any         `json:"movieQuotaLimit"`
	MovieQuotaDays  any         `json:"movieQuotaDays"`
	TvQuotaLimit    any         `json:"tvQuotaLimit"`
	TvQuotaDays     any         `json:"tvQuotaDays"`
}

type PageInfo struct {
	Pages    int `json:"pages"`
	PageSize int `json:"pageSize"`
	Results  int `json:"results"`
	Page     int `json:"page"`
}

type GetUsersDTO struct {
	Page    PageInfo `json:"pageInfo"`
	Results []User   `json:"results"`
}

type permissionsDTO struct {
	Permissions Permissions `json:"permissions"`
}

type Permissions int

type NotificationTypes struct {
	Discord    int `json:"discord"`
	Email      int `json:"email"`
	Pushbullet int `json:"pushbullet"`
	Pushover   int `json:"pushover"`
	Slack      int `json:"slack"`
	Telegram   int `json:"telegram"`
	Webhook    int `json:"webhook"`
	Webpush    int `json:"webpush"`
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
	PgpKey                   any    `json:"pgpKey"`
	DiscordID                string `json:"discordId"`
	PushbulletAccessToken    any    `json:"pushbulletAccessToken"`
	PushoverApplicationToken any    `json:"pushoverApplicationToken"`
	PushoverUserKey          any    `json:"pushoverUserKey"`
	TelegramChatID           string `json:"telegramChatId"`
}

type NotificationsTemplate struct {
	EmailEnabled         bool              `json:"emailEnabled"`
	DiscordEnabled       bool              `json:"discordEnabled"`
	DiscordEnabledTypes  int               `json:"discordEnabledTypes"`
	PushoverSound        any               `json:"pushoverSound"`
	TelegramEnabled      bool              `json:"telegramEnabled"`
	TelegramSendSilently any               `json:"telegramSendSilently"`
	WebPushEnabled       bool              `json:"webPushEnabled"`
	NotifTypes           NotificationTypes `json:"notificationTypes"`
}
