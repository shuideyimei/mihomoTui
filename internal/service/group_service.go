package service

import (
	"fmt"
	"strings"

	"mihomoTui/internal/api"
	"mihomoTui/internal/models"
	"mihomoTui/internal/repository"
)

// ProxyGroupService manages proxy group business logic.
type ProxyGroupService struct {
	repo repository.ProxyGroupRepository
}

// NewProxyGroupService creates a new ProxyGroupService.
func NewProxyGroupService(repo repository.ProxyGroupRepository) *ProxyGroupService {
	return &ProxyGroupService{repo: repo}
}

// GetGroups returns all proxy groups.
func (s *ProxyGroupService) GetGroups() ([]models.ProxyGroupEntry, error) {
	return s.repo.GetGroups()
}

// AddGroup adds a new proxy group and reloads Mihomo config.
func (s *ProxyGroupService) AddGroup(name, groupType, filter, testURL string, use, proxies []string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("group name cannot be empty")
	}

	configPath, err := s.repo.AddGroup(name, groupType, filter, testURL, use, proxies)
	if err != nil {
		return err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return nil
}

// UpdateGroup updates an existing proxy group and reloads Mihomo config.
func (s *ProxyGroupService) UpdateGroup(oldName, name, groupType, filter, testURL string, use, proxies []string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("group name cannot be empty")
	}

	configPath, err := s.repo.UpdateGroup(oldName, name, groupType, filter, testURL, use, proxies)
	if err != nil {
		return err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return nil
}

// DeleteGroup deletes a proxy group and reloads Mihomo config.
func (s *ProxyGroupService) DeleteGroup(name string) error {
	configPath, err := s.repo.DeleteGroup(name)
	if err != nil {
		return err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return nil
}

// GetGroupNames returns list of all group names.
func (s *ProxyGroupService) GetGroupNames() ([]string, error) {
	return s.repo.GetGroupNames()
}

// GetGroupDef retrieves proxies and use providers for a specific group.
func (s *ProxyGroupService) GetGroupDef(groupName string) ([]string, []string, error) {
	return s.repo.GetGroupDef(groupName)
}
