package service

import (
	"context"
	"fmt"
	"strings"

	"mihomoTui/internal/api"
	"mihomoTui/internal/models"
	"mihomoTui/internal/repository"
	"mihomoTui/internal/subscription"
	"mihomoTui/internal/types"
)

// ProviderService manages proxy and rule providers.
type ProviderService struct {
	repo   repository.ProviderRepository
	subMgr *subscription.Manager
}

// NewProviderService creates a new ProviderService.
func NewProviderService(repo repository.ProviderRepository, subMgr *subscription.Manager) *ProviderService {
	return &ProviderService{
		repo:   repo,
		subMgr: subMgr,
	}
}

// GetProxyProviders returns all proxy providers.
func (s *ProviderService) GetProxyProviders() ([]models.ProviderInfo, error) {
	return s.repo.GetProxyProviders()
}

// AddProxyProvider adds a remote proxy provider and reloads config.
func (s *ProviderService) AddProxyProvider(name, url string) error {
	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if url == "" {
		return fmt.Errorf("provider URL cannot be empty")
	}

	configPath, err := s.repo.AddProxyProvider(name, url)
	if err != nil {
		return err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return nil
}

// ImportLocal imports a local subscription file, persists it to config, and reloads Mihomo.
func (s *ProviderService) ImportLocal(name, localPath string) (*types.ConfigDocument, error) {
	name = strings.TrimSpace(name)
	localPath = strings.TrimSpace(localPath)
	if name == "" {
		return nil, fmt.Errorf("provider name cannot be empty")
	}
	if localPath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	if s.subMgr == nil {
		return nil, fmt.Errorf("subscription manager is not initialized")
	}

	doc, err := s.subMgr.ImportFromLocal(localPath)
	if err != nil {
		return nil, err
	}

	configPath, err := s.repo.AddLocalProxyProvider(name, localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to save local provider to config: %w", err)
	}

	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}

	return doc, nil
}

// ImportURL downloads and validates a remote subscription, saves to config, and reloads Mihomo.
func (s *ProviderService) ImportURL(ctx context.Context, name, url string) (*types.ConfigDocument, error) {
	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	if name == "" {
		return nil, fmt.Errorf("provider name cannot be empty")
	}
	if url == "" {
		return nil, fmt.Errorf("provider URL cannot be empty")
	}

	if s.subMgr == nil {
		return nil, fmt.Errorf("subscription manager is not initialized")
	}

	doc, err := s.subMgr.ImportFromURL(ctx, url)
	if err != nil {
		return nil, err
	}

	configPath, err := s.repo.AddProxyProvider(name, url)
	if err != nil {
		return nil, fmt.Errorf("failed to save provider to config: %w", err)
	}

	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}

	return doc, nil
}

// RemoveProxyProvider removes a proxy provider and reloads config.
func (s *ProviderService) RemoveProxyProvider(name string) error {
	if err := s.repo.RemoveProxyProvider(name); err != nil {
		return err
	}
	return nil
}

// GetRuleProviders returns all rule providers.
func (s *ProviderService) GetRuleProviders() ([]models.RuleProviderEntry, error) {
	return s.repo.GetRuleProviders()
}

// AddRuleProvider adds a rule provider and reloads config.
func (s *ProviderService) AddRuleProvider(name, url, behavior string, interval int) error {
	configPath, err := s.repo.AddRuleProvider(name, url, behavior, interval)
	if err != nil {
		return err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return nil
}

// UpdateRuleProvider updates a rule provider and reloads config.
func (s *ProviderService) UpdateRuleProvider(name, url, behavior string, interval int) error {
	configPath, err := s.repo.UpdateRuleProvider(name, url, behavior, interval)
	if err != nil {
		return err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return nil
}

// DeleteRuleProvider deletes a rule provider and reloads config.
func (s *ProviderService) DeleteRuleProvider(name string) error {
	configPath, err := s.repo.DeleteRuleProvider(name)
	if err != nil {
		return err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return nil
}

// BatchAddRuleProviders imports multiple rule providers in batch and reloads config once.
func (s *ProviderService) BatchAddRuleProviders(providers []models.RuleProviderEntry) (int, error) {
	if len(providers) == 0 {
		return 0, nil
	}
	configPath, err := s.repo.BatchAddRuleProviders(providers)
	if err != nil {
		return 0, err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return len(providers), nil
}

// RefreshRuleProvider triggers Mihomo runtime refresh for a rule provider.
func (s *ProviderService) RefreshRuleProvider(name string) error {
	if api.Client != nil {
		return api.Client.RefreshRuleProvider(name)
	}
	return nil
}
