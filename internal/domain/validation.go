package domain

import (
	"errors"
	"fmt"
)

// Validate checks if the Server has valid required fields.
func (s *Server) Validate() error {
	if s.Host == "" {
		return errors.New("server host is required")
	}
	if s.Port != 0 && (s.Port < 1 || s.Port > 65535) {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", s.Port)
	}
	if s.DisplayName == "" {
		return errors.New("server display name is required")
	}
	return nil
}

// Validate checks if the Project has valid required fields.
func (p *Project) Validate() error {
	if p.Name == "" {
		return errors.New("project name is required")
	}
	return nil
}

// Validate checks if the Credential has valid required fields.
func (c *Credential) Validate() error {
	if c.Name == "" {
		return errors.New("credential name is required")
	}
	if c.Type == CredentialKeyFile && c.KeyFilePath == "" {
		return errors.New("key file path is required when credential type is Key File")
	}
	return nil
}
