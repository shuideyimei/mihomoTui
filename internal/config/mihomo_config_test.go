package config

import (
	"testing"
)

func TestParseProcessCmdlines(t *testing.T) {
	sampleOutput := `
/Library/PrivilegedHelperTools/io.github.clash-verge-rev.clash-verge-rev.service
/Applications/Clash Verge.app/Contents/MacOS/verge-mihomo -d /Users/test/Library/Application Support/clash-verge -f /Users/test/Library/Application Support/clash-verge/clash-verge.yaml -ext-ctl-unix /tmp/verge.sock
/usr/local/bin/mihomo -d /etc/mihomo -ext-ctl 127.0.0.1:9090
/bin/bash -c ps -A -o command | grep -iE 'mihomo|clash'
mihomoTui
`
	paths := parseProcessCmdlines(sampleOutput)
	if len(paths) == 0 {
		t.Fatalf("expected extracted paths, got 0")
	}

	foundYaml := false
	foundDir := false
	for _, p := range paths {
		if p == "/Users/test/Library/Application Support/clash-verge/clash-verge.yaml" {
			foundYaml = true
		}
		if p == "/Users/test/Library/Application Support/clash-verge" {
			foundDir = true
		}
		if p == "/etc/mihomo" {
			foundDir = true
		}
	}

	if !foundYaml {
		t.Errorf("failed to extract -f clash-verge.yaml from cmdline, got %v", paths)
	}
	if !foundDir {
		t.Errorf("failed to extract -d directory from cmdline, got %v", paths)
	}
}

func TestFindMihomoConfigPathOnHost(t *testing.T) {
	path := FindMihomoConfigPath()
	t.Logf("Discovered mihomo config path: %s", path)
	if path != "" {
		if err := ValidateConfigFile(path); err != nil {
			t.Errorf("discovered path is invalid: %v", err)
		}
	}
}

func TestLoadHostConfig(t *testing.T) {
	groups, err := GetAllProxyGroupsFromConfig()
	if err != nil {
		t.Logf("GetAllProxyGroupsFromConfig: %v", err)
	} else {
		t.Logf("Found %d proxy groups", len(groups))
	}

	rules, err := GetRulesFromConfig()
	if err != nil {
		t.Logf("GetRulesFromConfig: %v", err)
	} else {
		t.Logf("Found %d rules", len(rules))
	}

	rps, err := GetRuleProvidersFromConfig()
	if err != nil {
		t.Logf("GetRuleProvidersFromConfig: %v", err)
	} else {
		t.Logf("Found %d rule providers", len(rps))
	}
}
