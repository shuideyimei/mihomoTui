package parser

import (
	"testing"

	"mihomoTui/internal/types"
)

func TestDetectFormat_ClashYAML(t *testing.T) {
	data := []byte("port: 7890\nsocks-port: 7891\nproxies:\n  - name: hk-01\n    type: ss\n")
	format := DetectFormat(data)
	if format != FormatClashYAML {
		t.Errorf("expected FormatClashYAML, got %v", format)
	}
}

func TestDetectFormat_ShadowsocksURI(t *testing.T) {
	data := []byte("ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ@example.com:8443#MyServer")
	format := DetectFormat(data)
	if format != FormatShadowsocks {
		t.Errorf("expected FormatShadowsocks, got %v", format)
	}
}

func TestDetectFormat_TrojanURI(t *testing.T) {
	data := []byte("trojan://password@us.example.com:443?security=tls#US-01")
	format := DetectFormat(data)
	if format != FormatTrojan {
		t.Errorf("expected FormatTrojan, got %v", format)
	}
}

func TestDetectFormat_V2RayURI(t *testing.T) {
	data := []byte("vmess://eyJhZGQiOiJleGFtcGxlLmNvbSIsInBvcnQiOjQ0MywiaWQiOiJ1dWlkIiwibmV0Ijoid3MifQ==")
	format := DetectFormat(data)
	if format != FormatV2Ray {
		t.Errorf("expected FormatV2Ray, got %v", format)
	}
}

func TestDetectFormat_Hysteria2URI(t *testing.T) {
	data := []byte("hysteria2://password@example.com:443?insecure=1#Hy2-Node")
	format := DetectFormat(data)
	if format != FormatHysteria2 {
		t.Errorf("expected FormatHysteria2, got %v", format)
	}
}

func TestDetectFormat_Surge(t *testing.T) {
	data := []byte("[Proxy]\n🇭🇰HK = ss, hk.example.com, 443, aes-256-gcm, password\n[Rule]\nDOMAIN-SUFFIX,google.com,Proxy")
	format := DetectFormat(data)
	if format != FormatSurge {
		t.Errorf("expected FormatSurge, got %v", format)
	}
}

func TestParse_ClashYAML(t *testing.T) {
	data := []byte(`
proxies:
  - name: hk-01
    type: ss
    server: hk.example.com
    port: 443
    cipher: aes-256-gcm
    password: secret

proxy-groups:
  - name: Proxy
    type: select
    proxies:
      - hk-01
      - DIRECT

rules:
  - DOMAIN-SUFFIX,google.com,Proxy
  - MATCH,DIRECT
`)
	doc, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(doc.Proxies) != 1 {
		t.Errorf("expected 1 proxy, got %d", len(doc.Proxies))
	}
	if len(doc.ProxyGroups) != 1 {
		t.Errorf("expected 1 proxy-group, got %d", len(doc.ProxyGroups))
	}
	if len(doc.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(doc.Rules))
	}

	proxy := doc.Proxies[0]
	if proxy["name"] != "hk-01" {
		t.Errorf("expected proxy name hk-01, got %v", proxy["name"])
	}
}

func TestParse_ShadowsocksURIs(t *testing.T) {
	data := []byte("ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ@hk.example.com:8443#HK-Node\nss://YWVzLTI1Ni1nY206cGFzczI@jp.example.com:8443#JP-Node\n")
	doc, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(doc.Proxies) != 2 {
		t.Errorf("expected 2 proxies, got %d", len(doc.Proxies))
	}
}

func TestParse_MixedProxyURIs(t *testing.T) {
	data := []byte("trojan://pass1@us.example.com:443#US-01\nss://YWVzLTI1Ni1nY206cGFzc3dvcmQ@sg.example.com:8443#SG-01\n")
	doc, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(doc.Proxies) != 2 {
		t.Errorf("expected 2 proxies, got %d", len(doc.Proxies))
	}
}

func TestParse_EmptyData(t *testing.T) {
	_, err := Parse([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestValidateConfig_Valid(t *testing.T) {
	doc := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"name": "hk-01", "type": "ss"},
		},
		ProxyGroups: []map[string]interface{}{
			{"name": "Proxy", "type": "select"},
		},
	}
	err := ValidateConfig(doc)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateConfig_ProxyNoName(t *testing.T) {
	doc := &types.ConfigDocument{
		Proxies: []map[string]interface{}{
			{"type": "ss"}, // missing name
		},
	}
	err := ValidateConfig(doc)
	if err == nil {
		t.Error("expected error for proxy without name")
	}
}
