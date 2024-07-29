package jellyseerr

import "time"

type User struct {
	Permissions                int       `json:"permissions"`
	Warnings                   []any     `json:"warnings"`
	ID                         int       `json:"id"`
	Email                      string    `json:"email"`
	PlexUsername               string    `json:"plexUsername"`
	JellyfinUsername           string    `json:"jellyfinUsername"`
	Username                   string    `json:"username"`
	RecoveryLinkExpirationDate any       `json:"recoveryLinkExpirationDate"`
	UserType                   int       `json:"userType"`
	PlexID                     string    `json:"plexId"`
	JellyfinUserID             string    `json:"jellyfinUserId"`
	JellyfinDeviceID           string    `json:"jellyfinDeviceId"`
	JellyfinAuthToken          string    `json:"jellyfinAuthToken"`
	PlexToken                  string    `json:"plexToken"`
	Avatar                     string    `json:"avatar"`
	MovieQuotaLimit            any       `json:"movieQuotaLimit"`
	MovieQuotaDays             any       `json:"movieQuotaDays"`
	TvQuotaLimit               any       `json:"tvQuotaLimit"`
	TvQuotaDays                any       `json:"tvQuotaDays"`
	CreatedAt                  time.Time `json:"createdAt"`
	UpdatedAt                  time.Time `json:"updatedAt"`
	RequestCount               int       `json:"requestCount"`
	DisplayName                string    `json:"displayName"`
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
