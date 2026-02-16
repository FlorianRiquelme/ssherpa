package sshkey

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		key      SSHKey
		expected string
	}{
		{
			name: "file source with filename",
			key: SSHKey{
				Filename: "id_ed25519",
				Source:   SourceFile,
			},
			expected: "id_ed25519",
		},
		{
			name: "agent source with comment",
			key: SSHKey{
				Comment: "florian@work",
				Source:  SourceAgent,
			},
			expected: "agent:florian@work",
		},
		{
			name: "agent source without comment",
			key: SSHKey{
				Source: SourceAgent,
			},
			expected: "agent:",
		},
		{
			name: "1password source with filename",
			key: SSHKey{
				Filename: "production-key",
				Source:   Source1Password,
			},
			expected: "1p:production-key",
		},
		{
			name: "unknown source falls back to filename",
			key: SSHKey{
				Filename: "some-key",
				Source:   KeySource(999),
			},
			expected: "some-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.key.DisplayName())
		})
	}
}

func TestSourceBadge(t *testing.T) {
	tests := []struct {
		name     string
		source   KeySource
		expected string
	}{
		{
			name:     "file source",
			source:   SourceFile,
			expected: "[file]",
		},
		{
			name:     "agent source",
			source:   SourceAgent,
			expected: "[agent]",
		},
		{
			name:     "1password source",
			source:   Source1Password,
			expected: "[1password]",
		},
		{
			name:     "unknown source",
			source:   KeySource(999),
			expected: "[unknown]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := SSHKey{Source: tt.source}
			assert.Equal(t, tt.expected, key.SourceBadge())
		})
	}
}

func TestKeySourceString(t *testing.T) {
	tests := []struct {
		name     string
		source   KeySource
		expected string
	}{
		{
			name:     "file",
			source:   SourceFile,
			expected: "file",
		},
		{
			name:     "agent",
			source:   SourceAgent,
			expected: "agent",
		},
		{
			name:     "1password",
			source:   Source1Password,
			expected: "1password",
		},
		{
			name:     "unknown",
			source:   KeySource(999),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.source.String())
		})
	}
}
