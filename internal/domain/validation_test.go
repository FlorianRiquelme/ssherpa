package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerValidate(t *testing.T) {
	tests := []struct {
		name    string
		server  *Server
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid server passes",
			server: &Server{
				Host:        "example.com",
				User:        "admin",
				Port:        22,
				DisplayName: "Production Server",
			},
			wantErr: false,
		},
		{
			name: "empty host fails",
			server: &Server{
				Host:        "",
				User:        "admin",
				Port:        22,
				DisplayName: "Test Server",
			},
			wantErr: true,
			errMsg:  "host",
		},
		{
			name: "port 0 defaults OK",
			server: &Server{
				Host:        "example.com",
				User:        "admin",
				Port:        0,
				DisplayName: "Test Server",
			},
			wantErr: false,
		},
		{
			name: "port -1 fails",
			server: &Server{
				Host:        "example.com",
				User:        "admin",
				Port:        -1,
				DisplayName: "Test Server",
			},
			wantErr: true,
			errMsg:  "port",
		},
		{
			name: "port 70000 fails",
			server: &Server{
				Host:        "example.com",
				User:        "admin",
				Port:        70000,
				DisplayName: "Test Server",
			},
			wantErr: true,
			errMsg:  "port",
		},
		{
			name: "empty display name fails",
			server: &Server{
				Host:        "example.com",
				User:        "admin",
				Port:        22,
				DisplayName: "",
			},
			wantErr: true,
			errMsg:  "display name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.server.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProjectValidate(t *testing.T) {
	tests := []struct {
		name    string
		project *Project
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid project passes",
			project: &Project{
				Name:        "My Project",
				Description: "A test project",
			},
			wantErr: false,
		},
		{
			name: "empty name fails",
			project: &Project{
				Name:        "",
				Description: "A test project",
			},
			wantErr: true,
			errMsg:  "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCredentialValidate(t *testing.T) {
	tests := []struct {
		name       string
		credential *Credential
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid credential passes",
			credential: &Credential{
				Name:        "My SSH Key",
				Type:        CredentialKeyFile,
				KeyFilePath: "/path/to/key",
			},
			wantErr: false,
		},
		{
			name: "empty name fails",
			credential: &Credential{
				Name:        "",
				Type:        CredentialKeyFile,
				KeyFilePath: "/path/to/key",
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "KeyFile type with empty KeyFilePath fails",
			credential: &Credential{
				Name:        "My SSH Key",
				Type:        CredentialKeyFile,
				KeyFilePath: "",
			},
			wantErr: true,
			errMsg:  "key file path",
		},
		{
			name: "SSHAgent type with empty KeyFilePath passes (not needed for agent auth)",
			credential: &Credential{
				Name:        "SSH Agent",
				Type:        CredentialSSHAgent,
				KeyFilePath: "",
			},
			wantErr: false,
		},
		{
			name: "Password type with empty KeyFilePath passes",
			credential: &Credential{
				Name:        "Password Auth",
				Type:        CredentialPassword,
				KeyFilePath: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.credential.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
