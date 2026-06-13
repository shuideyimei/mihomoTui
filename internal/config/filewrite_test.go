package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWriteFileAtomicPreservesModeAndReplacesContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := WriteFileAtomic(path, []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFileAtomic failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("expected new content, got %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("expected mode 0600, got %o", got)
	}
}

func TestFindMihomoProcessConfigIgnoresUnrelatedProcesses(t *testing.T) {
	procRoot := t.TempDir()
	writeProcess := func(pid, comm string, args ...string) {
		t.Helper()
		dir := filepath.Join(procRoot, pid)
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "comm"), []byte(comm+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		cmdline := []byte{}
		for _, arg := range args {
			cmdline = append(cmdline, []byte(arg)...)
			cmdline = append(cmdline, 0)
		}
		if err := os.WriteFile(filepath.Join(dir, "cmdline"), cmdline, 0644); err != nil {
			t.Fatal(err)
		}
	}

	writeProcess("100", "unrelated", "/usr/bin/unrelated", "-d", "/tmp/wrong")
	writeProcess("101", "mihomo", "/usr/bin/mihomo", "-d", "/tmp/right")
	writeProcess("102", "mihomo-alpha", "/usr/bin/mihomo-alpha", "--directory=/tmp/other")
	writeProcess("103", "mihomo", "/usr/bin/mihomo", "-d")
	writeProcess("104", "mihomo", "/usr/bin/mihomo", "--directory=")

	got := findMihomoProcessConfigAt(procRoot)
	want := []string{"/tmp/right", "/tmp/other"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFindMihomoConfigPathPrefersExplicitPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.yaml")
	if err := os.WriteFile(path, []byte("mixed-port: 7890\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MIHOMO_CONFIG_PATH", path)

	if got := FindMihomoConfigPath(); got != path {
		t.Fatalf("expected explicit path %q, got %q", path, got)
	}
}

func TestValidateConfigFileParsesTopLevelKeys(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.yaml")
	invalid := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(valid, []byte("mixed-port: 7890\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalid, []byte("description: 'contains proxies: but is not a config'\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ValidateConfigFile(valid); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
	if err := ValidateConfigFile(invalid); err == nil {
		t.Fatal("expected non-Mihomo YAML to be rejected")
	}
}

func TestSameConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("mixed-port: 7890\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, path)
	if err != nil {
		t.Fatal(err)
	}
	if !SameConfigFile(path, relative) {
		t.Fatalf("expected %q and %q to identify the same file", path, relative)
	}
}

func TestFindAllConfigFilesDoesNotScanWorkingDirectory(t *testing.T) {
	workingDir := t.TempDir()
	homeDir := t.TempDir()
	path := filepath.Join(workingDir, "unrelated.yaml")
	if err := os.WriteFile(path, []byte("mixed-port: 7890\n"), 0644); err != nil {
		t.Fatal(err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
	t.Setenv("HOME", homeDir)
	t.Setenv("MIHOMO_CONFIG_PATH", "")

	for _, got := range FindAllConfigFiles() {
		if SameConfigFile(got, path) {
			t.Fatalf("working-directory config %q should not be discovered", path)
		}
	}
}

func TestMoveNodesToPosition(t *testing.T) {
	makeNodes := func() []*yaml.Node {
		return []*yaml.Node{
			{Value: "a"},
			{Value: "b"},
			{Value: "c"},
			{Value: "d"},
		}
	}
	values := func(nodes []*yaml.Node) []string {
		out := make([]string, len(nodes))
		for i, node := range nodes {
			out[i] = node.Value
		}
		return out
	}

	tests := []struct {
		name     string
		from, to int
		want     []string
	}{
		{name: "move down to final index", from: 0, to: 3, want: []string{"b", "c", "d", "a"}},
		{name: "move down one", from: 1, to: 2, want: []string{"a", "c", "b", "d"}},
		{name: "move up", from: 3, to: 1, want: []string{"a", "d", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := values(moveNodesToPosition(makeNodes(), tt.from, tt.to))
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
