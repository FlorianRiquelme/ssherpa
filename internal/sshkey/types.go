package sshkey

// KeySource represents where an SSH key was discovered.
type KeySource int

const (
	// SourceFile means the key was discovered in ~/.ssh/ directory
	SourceFile KeySource = iota
	// SourceAgent means the key was discovered in SSH agent
	SourceAgent
	// Source1Password means the key was discovered in 1Password backend
	Source1Password
)

// SSHKey represents a discovered SSH key with metadata.
type SSHKey struct {
	// Path is the filesystem path (empty for agent-only or 1P keys)
	Path string
	// Filename is just the filename (e.g. "id_ed25519")
	Filename string
	// Type is the key algorithm: "ed25519", "rsa", "ecdsa", "dsa" (lowercase, no "ssh-" prefix)
	Type string
	// Fingerprint is the SHA256 fingerprint (e.g. "SHA256:abc123...")
	Fingerprint string
	// Comment is from .pub file or agent (empty if unavailable)
	Comment string
	// Source is where the key was found
	Source KeySource
	// Bits is the key size in bits (0 if unknown, relevant for RSA)
	Bits int
	// Encrypted is true if private key is passphrase-protected
	Encrypted bool
	// Missing is true if referenced in config but file not on disk
	Missing bool
	// MissingPath is the path that was referenced but not found (only set when Missing=true)
	MissingPath string
}

// DisplayName returns a human-friendly display name for the key.
// For file keys: returns filename
// For agent keys: returns "agent:<comment>"
// For 1Password keys: returns "1p:<filename>"
func (k SSHKey) DisplayName() string {
	switch k.Source {
	case SourceFile:
		return k.Filename
	case SourceAgent:
		return "agent:" + k.Comment
	case Source1Password:
		return "1p:" + k.Filename
	default:
		return k.Filename
	}
}

// SourceBadge returns a badge string indicating the source.
func (k SSHKey) SourceBadge() string {
	switch k.Source {
	case SourceFile:
		return "[file]"
	case SourceAgent:
		return "[agent]"
	case Source1Password:
		return "[1password]"
	default:
		return "[unknown]"
	}
}
