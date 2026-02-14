package sshkey

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// DiscoverAgentKeys discovers SSH keys loaded in the SSH agent via SSH_AUTH_SOCK.
// Returns empty slice (not error) if agent is unavailable or unreachable.
// Agent unavailability is a normal condition, not an error.
func DiscoverAgentKeys() ([]SSHKey, error) {
	socketPath := os.Getenv("SSH_AUTH_SOCK")
	if socketPath == "" {
		return []SSHKey{}, nil
	}
	return discoverKeysFromSocket(socketPath, SourceAgent)
}

// DiscoverKeysFromSocket discovers SSH keys from a specific agent socket path.
// The source parameter tags discovered keys (e.g. Source1Password for 1Password's IdentityAgent).
// Returns empty slice (not error) if socket is unavailable.
func DiscoverKeysFromSocket(socketPath string, source KeySource) ([]SSHKey, error) {
	return discoverKeysFromSocket(socketPath, source)
}

// discoverKeysFromSocket connects to an SSH agent socket and lists keys.
func discoverKeysFromSocket(socketPath string, source KeySource) ([]SSHKey, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return []SSHKey{}, nil
	}
	defer conn.Close()

	agentClient := agent.NewClient(conn)

	agentKeys, err := agentClient.List()
	if err != nil {
		return []SSHKey{}, nil
	}

	keys := make([]SSHKey, 0, len(agentKeys))
	for _, agentKey := range agentKeys {
		pubKey, err := ssh.ParsePublicKey(agentKey.Marshal())
		if err != nil {
			continue
		}

		key := SSHKey{
			Type:        extractKeyType(pubKey.Type()),
			Fingerprint: ssh.FingerprintSHA256(pubKey),
			Comment:     agentKey.Comment,
			Source:      source,
			Bits:        extractKeyBits(pubKey),
		}

		keys = append(keys, key)
	}

	return keys, nil
}
