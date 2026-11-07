package pluginconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fza/sdd-helpers/source/hooks/internal/pluginconfig"
	"github.com/fza/sdd-helpers/source/hooks/test"
)

func TestRead(t *testing.T) {
	const own = `{"pluginConfigs":{"sdd-helpers@sdd-helpers":{"options":`

	cases := []struct {
		name    string
		user    string
		managed string
		want    pluginconfig.Options
		wantErr bool
	}{
		{name: "no settings files leave the defaults", want: pluginconfig.Defaults()},
		{
			name: "user settings switch a program off",
			user: own + `{"capture_enabled":false,"capture_remove_draft":false}}}}`,
			want: pluginconfig.Options{GroundingGateEnabled: true},
		},
		{
			name:    "managed settings win per key",
			user:    own + `{"grounding_gate_enabled":false,"capture_enabled":false}}}}`,
			managed: own + `{"grounding_gate_enabled":true}}}}`,
			want:    pluginconfig.Options{GroundingGateEnabled: true, CaptureRemoveDraft: true},
		},
		{
			name: "a switch reads as a boolean or the dialog's string",
			user: own + `{"capture_enabled":"false"}}}}`,
			want: pluginconfig.Options{GroundingGateEnabled: true, CaptureRemoveDraft: true},
		},
		{
			name: "an empty value is unset",
			user: own + `{"capture_enabled":"","grounding_gate_enabled":null}}}}`,
			want: pluginconfig.Defaults(),
		},
		{
			name: "another plugin's options change nothing",
			user: `{"pluginConfigs":{"other@other":{"options":{"capture_enabled":false}}}}`,
			want: pluginconfig.Defaults(),
		},
		{name: "a switch of another word is an error", user: own + `{"capture_enabled":"sometimes"}}}}`, wantErr: true},
		{name: "a settings file that is no JSON is an error", user: `{"pluginConfigs":`, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			user := filepath.Join(root, "settings.json")
			managed := filepath.Join(root, "managed-settings.json")

			if tc.user != "" {
				test.WriteFile(t, user, tc.user)
			}

			if tc.managed != "" {
				test.WriteFile(t, managed, tc.managed)
			}

			got, err := pluginconfig.ReadWith([]string{user, managed}, nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("an invalid settings file, and only one, should be an error, got %v", err)
			}

			if !tc.wantErr && got != tc.want {
				t.Errorf("options should fold the settings in precedence order, got %+v", got)
			}
		})
	}
}

func TestAnUnreadableSettingsFileIsAnError(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "settings.json")

	err := os.Mkdir(user, 0o700)
	if err != nil {
		t.Fatal(err)
	}

	_, err = pluginconfig.ReadWith([]string{user}, nil)
	if err == nil {
		t.Error("a settings file that cannot be read should be an error, never the defaults")
	}
}

func TestPathsNeverIncludeProjectSettings(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "windows"} {
		for _, path := range pluginconfig.Paths("/home/u", goos, nil) {
			if filepath.Base(filepath.Dir(path)) == ".claude" && filepath.Dir(filepath.Dir(path)) != "/home/u" {
				t.Errorf("only user and managed settings should be read, got %s", path)
			}
		}

		if pluginconfig.Paths("/home/u", goos, nil)[0] != "/home/u/.claude/settings.json" {
			t.Errorf("user settings should come first")
		}
	}
}
