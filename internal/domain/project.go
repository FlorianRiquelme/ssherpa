package domain

import "time"

// Project represents a group of servers, typically detected via git remote URLs.
// Server-to-project relationship is many-to-many, tracked on Server side via ProjectIDs.
type Project struct {
	ID            string
	Name          string   // e.g., "payments-api"
	Description   string
	GitRemoteURLs []string // one method for project detection
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
