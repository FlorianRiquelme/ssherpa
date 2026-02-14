package sshkey

import (
	"crypto/rsa"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// ParseKeyFile reads and parses an SSH private key file.
// Returns SSHKey with metadata or error if file is not a valid SSH key.
// For encrypted keys, sets Encrypted=true and attempts to read metadata from .pub file.
func ParseKeyFile(path string) (*SSHKey, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	// Check if it looks like a private key (header sniffing)
	header := string(data)
	if len(header) > 100 {
		header = header[:100]
	}

	isPEMKey := strings.Contains(header, "-----BEGIN")
	isOpenSSHKey := strings.Contains(header, "openssh-key-v1")

	if !isPEMKey && !isOpenSSHKey {
		return nil, fmt.Errorf("not an SSH private key (no PEM or OpenSSH header)")
	}

	// Try to parse the private key
	signer, err := ssh.ParsePrivateKey(data)

	key := &SSHKey{
		Path:     path,
		Filename: filepath.Base(path),
		Source:   SourceFile,
	}

	// If parsing fails, check if it's due to passphrase protection
	if err != nil {
		if isPassphraseMissing(err) {
			key.Encrypted = true
			// Try to get metadata from .pub file instead
			return parseFromPubFile(key, path)
		}
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	// Extract public key metadata
	pubKey := signer.PublicKey()
	key.Type = extractKeyType(pubKey.Type())
	key.Fingerprint = ssh.FingerprintSHA256(pubKey)
	key.Bits = extractKeyBits(pubKey)
	key.Encrypted = false

	// Try to read comment from .pub file
	key.Comment = ReadPubKeyComment(path + ".pub")

	return key, nil
}

// ReadPubKeyComment reads the comment field from an SSH public key file.
// Returns empty string if file doesn't exist or can't be parsed.
func ReadPubKeyComment(pubPath string) string {
	data, err := os.ReadFile(pubPath)
	if err != nil {
		return ""
	}

	pubKey, comment, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		return ""
	}

	// If no comment in the authorized_keys format, return empty
	if comment == "" {
		// Sometimes the comment is in the line after the key data
		// Format: "ssh-ed25519 AAAAC3... comment@host"
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			return strings.Join(parts[2:], " ")
		}
	}

	// Verify we got a valid public key
	if pubKey == nil {
		return ""
	}

	return comment
}

// parseFromPubFile attempts to extract key metadata from .pub file when private key is encrypted
func parseFromPubFile(key *SSHKey, privKeyPath string) (*SSHKey, error) {
	pubPath := privKeyPath + ".pub"
	data, err := os.ReadFile(pubPath)
	if err != nil {
		return nil, fmt.Errorf("encrypted key without .pub file: %w", err)
	}

	pubKey, comment, _, _, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse .pub file: %w", err)
	}

	key.Type = extractKeyType(pubKey.Type())
	key.Fingerprint = ssh.FingerprintSHA256(pubKey)
	key.Bits = extractKeyBits(pubKey)

	// Extract comment from authorized_keys format
	if comment != "" {
		key.Comment = comment
	} else {
		// Fallback: parse from line format
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			key.Comment = strings.Join(parts[2:], " ")
		}
	}

	return key, nil
}

// extractKeyType removes "ssh-" prefix from key type string
func extractKeyType(sshType string) string {
	// ssh.PublicKey.Type() returns things like "ssh-ed25519", "ssh-rsa", etc.
	// We want just "ed25519", "rsa", etc.
	return strings.TrimPrefix(sshType, "ssh-")
}

// extractKeyBits returns the key size in bits
func extractKeyBits(pubKey ssh.PublicKey) int {
	switch key := pubKey.(type) {
	case ssh.CryptoPublicKey:
		// For RSA keys, extract bit length
		if rsaPub, ok := key.CryptoPublicKey().(*rsa.PublicKey); ok {
			return rsaPub.N.BitLen()
		}
	}

	// For other key types, return standard sizes
	switch pubKey.Type() {
	case "ssh-ed25519":
		return 256
	case "ssh-rsa":
		// If we couldn't extract from the key itself, return 0
		return 0
	case "ecdsa-sha2-nistp256":
		return 256
	case "ecdsa-sha2-nistp384":
		return 384
	case "ecdsa-sha2-nistp521":
		return 521
	default:
		return 0
	}
}

// isPassphraseMissing checks if the error is due to a missing passphrase
func isPassphraseMissing(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Common error messages for encrypted keys
	return strings.Contains(errStr, "passphrase") ||
		strings.Contains(errStr, "encrypted") ||
		strings.Contains(errStr, "cannot decode") ||
		strings.Contains(errStr, "password")
}
