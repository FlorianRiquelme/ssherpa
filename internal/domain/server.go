package domain

import "time"

// Server represents an SSH server configuration with metadata.
// Domain model is storage-agnostic — no struct tags for serialization.
type Server struct {
	ID                string
	Host              string   // hostname or IP address
	User              string   // SSH username
	Port              int      // SSH port (default 22)
	IdentityFile      string   // path to SSH key file
	Proxy             string   // ProxyJump / bastion host
	Tags              []string // user-defined tags for filtering
	Notes             string   // free-form notes
	LastConnected     *time.Time
	Favorite          bool
	DisplayName       string   // human-friendly name
	VPNRequired       bool     // flag to warn before connecting
	CredentialID      string   // references a Credential by ID
	ProjectIDs        []string // server belongs to multiple projects
	RemoteProjectPath string   // remote path on server for 'ssh user@host -t "cd /path && $SHELL"'
	VaultID           string   // 1Password vault ID for write operations (empty for non-1P servers)
	Source            string   // backend that provided this server (e.g., "ssh-config", "1password")
}
