package types

import "time"

type UserInfo struct {
	Upload   int64 `json:"upload,omitempty" yaml:"upload,omitempty"`
	Download int64 `json:"download,omitempty" yaml:"download,omitempty"`
	Total    int64 `json:"total,omitempty" yaml:"total,omitempty"`
	Expire   int64 `json:"expire,omitempty" yaml:"expire,omitempty"`
}

type Subscription struct {
	ID           string    `json:"id" yaml:"id"`
	Name         string    `json:"name" yaml:"name"`
	URL          string    `json:"url,omitempty" yaml:"url,omitempty"`
	Type         string    `json:"type,omitempty"`
	UserInfo     *UserInfo `json:"user_info,omitempty" yaml:"user_info,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
	ProviderName string    `json:"provider_name,omitempty"`
}
