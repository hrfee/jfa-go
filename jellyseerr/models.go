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
