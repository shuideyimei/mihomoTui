package service

import (
	"fmt"
	"os"
	"path/filepath"

	"mihomoTui/internal/api"
	"mihomoTui/internal/events"
	"mihomoTui/internal/repository"
)

// ProfileService handles profile/configuration file operations and active profile switching.
type ProfileService struct {
	repo repository.ProfileRepository
	bus  *events.Bus
}

// NewProfileService creates a new ProfileService.
func NewProfileService(repo repository.ProfileRepository, bus *events.Bus) *ProfileService {
	return &ProfileService{
		repo: repo,
		bus:  bus,
	}
}

// ListProfiles returns all valid Mihomo config files found.
func (s *ProfileService) ListProfiles() []string {
	return s.repo.FindAllProfiles()
}

// GetActiveProfile returns the path to the currently active Mihomo config.
func (s *ProfileService) GetActiveProfile() string {
	return s.repo.GetCurrentProfilePath()
}

// ReadProfile reads the full text of a profile file.
func (s *ProfileService) ReadProfile(path string) (string, error) {
	return s.repo.ReadProfile(path)
}

// SaveProfile writes the profile content to disk and reloads it if it's the active profile.
func (s *ProfileService) SaveProfile(path, content string) error {
	if err := s.repo.WriteProfile(path, content); err != nil {
		return err
	}

	if api.Client != nil {
		// Try raw YAML reload first, fall back to file path reload
		if err := api.Client.ReloadConfigData([]byte(content)); err != nil {
			_ = api.Client.ReloadConfig(path)
		}
	}
	return nil
}

// SwitchProfile switches Mihomo to use a different configuration profile.
func (s *ProfileService) SwitchProfile(path string) error {
	if err := s.repo.ValidateProfile(path); err != nil {
		return fmt.Errorf("invalid profile file: %w", err)
	}

	s.ensureSafePaths(path)

	if api.Client == nil {
		return fmt.Errorf("API client is not initialized")
	}

	if err := api.Client.ReloadConfig(path); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	if s.bus != nil {
		s.bus.Publish(events.TopicConfigSwitched, events.ConfigSwitchedEvent{Path: path})
	}
	return nil
}

// DeleteProfile deletes an inactive configuration profile.
func (s *ProfileService) DeleteProfile(path string) error {
	return s.repo.DeleteProfile(path)
}

func (s *ProfileService) ensureSafePaths(activePath string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	cfgDir := filepath.Join(homeDir, ".config", "mihomo")
	paths, err := s.repo.EnsureSafePath(activePath, cfgDir)
	if err != nil || paths == nil {
		return
	}

	if api.Client != nil {
		_ = api.Client.PatchConfig(map[string][]string{
			"safe-paths": paths,
		})
	}
}
