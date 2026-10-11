package editorsettings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSettings(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFolderOverridesUserAndUserFillsTheGaps(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(t.TempDir(), "settings.json")
	writeSettings(t, user, `{
  // set once, every workspace inherits it
  "loreMaster.confluence.baseUrl": "https://user.atlassian.net/wiki",
  "loreMaster.defaults.mermaidMode": "code",
  "loreMaster.ignore": ["user-only/"],
}`)
	writeSettings(t, FolderSettingsPath(root), `{
  "loreMaster.ignore": ["folder-only/"],
  "loreMaster.outputs": [{ "platform": "confluence", "space": "DOC", "parentPageId": "123", "titlePrefix": "Lore" }]
}`)

	result, err := LoadEditorSettings(root, user)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Configured || result.FirstSync {
		t.Fatalf("configured=%v firstSync=%v", result.Configured, result.FirstSync)
	}
	output := result.Settings.Outputs[0]
	if output.BaseURL != "https://user.atlassian.net/wiki" {
		t.Fatalf("baseUrl = %q, want the user's", output.BaseURL)
	}
	if output.MermaidMode != "code" {
		t.Fatalf("mermaidMode = %q, want the user's default", output.MermaidMode)
	}
	if got := strings.Join(result.Settings.Ignore, ","); got != "folder-only/" {
		t.Fatalf("ignore = %q, want the folder's list to replace the user's", got)
	}
	if len(result.Sources) != 2 {
		t.Fatalf("sources = %v", result.Sources)
	}
}

func TestAnOutputsBaseURLBeatsTheUserDefault(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(t.TempDir(), "settings.json")
	writeSettings(t, user, `{"loreMaster.confluence.baseUrl": "https://user.atlassian.net/wiki"}`)
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.outputs": [{"baseUrl": "https://other.example.com/wiki", "space": "X", "parentPageId": "1", "titlePrefix": "P"}]}`)

	result, err := LoadEditorSettings(root, user)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Settings.Outputs[0].BaseURL; got != "https://other.example.com/wiki" {
		t.Fatalf("baseUrl = %q", got)
	}
}

func TestNoOutputsMeansNotConfigured(t *testing.T) {
	root := t.TempDir()
	result, err := LoadEditorSettings(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Configured {
		t.Fatal("an empty workspace reported as configured")
	}
}

func TestASecretKeyIsRefusedAndNamed(t *testing.T) {
	root := t.TempDir()
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.confluence.token": "abc"}`)

	_, err := LoadEditorSettings(root, "")
	if err == nil || !strings.Contains(err.Error(), "looks like a secret") {
		t.Fatalf("err = %v", err)
	}
}

func TestAnUnknownKeyAndAMisspeltFieldAreErrorsNamingTheirFile(t *testing.T) {
	root := t.TempDir()
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.outputz": []}`)
	if _, err := LoadEditorSettings(root, ""); err == nil || !strings.Contains(err.Error(), "loreMaster.outputz") {
		t.Fatalf("err = %v", err)
	}

	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.outputs": [{"platfrom": "confluence"}]}`)
	if _, err := LoadEditorSettings(root, ""); err == nil || !strings.Contains(err.Error(), "platfrom") {
		t.Fatalf("err = %v", err)
	}
}

func TestExtensionOnlySettingsAreIgnored(t *testing.T) {
	root := t.TempDir()
	writeSettings(t, FolderSettingsPath(root), `{"loreMaster.pages.label": "title", "loreMaster.watchDebounceSeconds": 5, "loreMaster.engine.path": ""}`)
	if _, err := LoadEditorSettings(root, ""); err != nil {
		t.Fatal(err)
	}
}

func TestUserSettingsPathPerSystem(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(key string) string { return values[key] }
	}
	if got := UserSettingsPath("linux", env(map[string]string{UserSettingsOverrideVariable: "/x/s.json"}), "/home/a"); got != "/x/s.json" {
		t.Fatalf("override = %q", got)
	}
	if got, want := UserSettingsPath("windows", env(map[string]string{"APPDATA": "C:/Users/a/AppData/Roaming"}), ""), filepath.Join("C:/Users/a/AppData/Roaming", "Code", "User", "settings.json"); got != want {
		t.Fatalf("windows = %q, want %q", got, want)
	}
	if got, want := UserSettingsPath("linux", env(nil), "/home/a"), filepath.Join("/home/a", ".config", "Code", "User", "settings.json"); got != want {
		t.Fatalf("linux = %q, want %q", got, want)
	}
	if got := UserSettingsPath("windows", env(nil), ""); got != "" {
		t.Fatalf("unknown location = %q", got)
	}
}
