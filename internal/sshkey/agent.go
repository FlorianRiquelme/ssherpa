package sshkey

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// DiscoverAgentKeys discovers SSH keys loaded in the SSH agent.
// Returns empty slice (not error) if agent is unavailable or unreachable.
// Agent unavailability is a normal condition, not an error.
func DiscoverAgentKeys() ([]SSHKey, error) {
	// Read SSH_AUTH_SOCK environment variable
	socketPath := os.Getenv("SSH_AUTH_SOCK")
	if socketPath == "" {
		// No agent configured - this is normal
		return []SSHKey{}, nil
	}

	// Try to connect to the agent socket
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		// Agent unreachable - this is normal (agent not running, socket stale, etc.)
		return []SSHKey{}, nil
	}
	defer conn.Close()

	// Create agent client
	agentClient := agent.NewClient(conn)

	// List keys in the agent
	agentKeys, err := agentClient.List()
	if err != nil {
		// Error listing keys - treat as agent unavailable
		return []SSHKey{}, nil
	}

	// Convert agent keys to SSHKey format
	keys := make([]SSHKey, 0, len(agentKeys))
	for _, agentKey := range agentKeys {
		// Parse the public key from marshaled format
		pubKey, err := ssh.ParsePublicKey(agentKey.Marshal())
		if err != nil {
			// Skip keys we can't parse
			continue
		}

		key := SSHKey{
			Type:        extractKeyType(pubKey.Type()),
			Fingerprint: ssh.FingerprintSHA256(pubKey),
			Comment:     agentKey.Comment,
			Source:      SourceAgent,
			Bits:        extractKeyBits(pubKey),
			// Path and Filename are empty for agent keys
		}

		keys = append(keys, key)
	}

	return keys, nil
}
