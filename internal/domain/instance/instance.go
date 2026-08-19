package instance

import "time"

// Instance identifies one AkôFlow control-plane installation.
type Instance struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Organization string    `json:"organization,omitempty"`
	Location     string    `json:"location,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
