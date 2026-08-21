package service

import (
	"fmt"
	"strings"

	"mihomoTui/internal/api"
	"mihomoTui/internal/models"
	"mihomoTui/internal/repository"
)

// RuleService manages rule-related business logic.
type RuleService struct {
	repo repository.RuleRepository
}

// NewRuleService creates a new RuleService.
func NewRuleService(repo repository.RuleRepository) *RuleService {
	return &RuleService{repo: repo}
}

// GetRules retrieves all rules from config.
func (s *RuleService) GetRules() ([]models.RuleDisplay, error) {
	return s.repo.GetRules()
}

// AddRule validates and adds a new rule, then triggers a Mihomo config reload.
func (s *RuleService) AddRule(ruleType, payload, proxy string) error {
	ruleType = strings.TrimSpace(ruleType)
	payload = strings.TrimSpace(payload)
	proxy = strings.TrimSpace(proxy)

	if ruleType == "" {
		return fmt.Errorf("rule type cannot be empty")
	}
	if proxy == "" {
		return fmt.Errorf("proxy policy cannot be empty")
	}

	configPath, err := s.repo.AddRule(ruleType, payload, proxy)
	if err != nil {
		return err
	}

	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return nil
}

// DeleteRule removes a rule by index and triggers a config reload.
func (s *RuleService) DeleteRule(index int) error {
	configPath, err := s.repo.DeleteRule(index)
	if err != nil {
		return err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return nil
}

// MoveRule moves a rule from one position to another and reloads config.
func (s *RuleService) MoveRule(fromIdx, toIdx int) error {
	configPath, err := s.repo.MoveRule(fromIdx, toIdx)
	if err != nil {
		return err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return nil
}

// UpdateRule updates a rule at index and reloads config.
func (s *RuleService) UpdateRule(index int, ruleType, payload, proxy string) error {
	configPath, err := s.repo.UpdateRule(index, ruleType, payload, proxy)
	if err != nil {
		return err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return nil
}

// BatchAddRules imports multiple rules in batch and reloads config once.
func (s *RuleService) BatchAddRules(rules []models.RuleDisplay) (int, error) {
	if len(rules) == 0 {
		return 0, nil
	}
	configPath, err := s.repo.BatchAddRules(rules)
	if err != nil {
		return 0, err
	}
	if configPath != "" && api.Client != nil {
		_ = api.Client.ReloadConfig(configPath)
	}
	return len(rules), nil
}
