package domain

// CredentialType defines the authentication method for a credential.
type CredentialType int

const (
	CredentialKeyFile CredentialType = iota
	CredentialSSHAgent
	CredentialPassword
)

// String returns a human-readable representation of the credential type.
func (ct CredentialType) String() string {
	switch ct {
	case CredentialKeyFile:
		return "Key File"
	case CredentialSSHAgent:
		return "SSH Agent"
	case CredentialPassword:
		return "Password"
	default:
		return "Unknown"
	}
}

// Credential represents an authentication reference (not a secret store).
// Actual secrets live in filesystem, 1Password, or SSH agent.
type Credential struct {
	ID          string
	Name        string // human label, e.g., "Work SSH Key"
	Type        CredentialType
	KeyFilePath string // populated when Type == CredentialKeyFile
	Notes       string
}
