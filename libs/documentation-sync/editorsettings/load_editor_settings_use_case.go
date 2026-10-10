package editorsettings

import (
	"path/filepath"

	"lore-master/libs/documentation-sync/workspacesettings"
)

// FolderSettingsPath is the workspace folder's own settings file.
func FolderSettingsPath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".vscode", "settings.json")
}

// LoadEditorSettings resolves the configuration from the user's settings and the
// workspace folder's, the folder overriding the user (#308). userSettingsPath may be
// empty. When no layer sets loreMaster.outputs the result is not Configured and holds
// no settings, so the caller can fall back to .lore-master.yaml. Otherwise the settings
// are completed and validated as a file's are, and an error names the file and key.
func LoadEditorSettings(workspaceRoot, userSettingsPath string) (Result, error) {
	var layers []Layer
	for _, path := range []string{userSettingsPath, FolderSettingsPath(workspaceRoot)} {
		if path == "" {
			continue
		}
		layer, err := ReadLayer(path)
		if err != nil {
			return Result{}, err
		}
		layers = append(layers, layer)
	}

	var sources []string
	for _, layer := range layers {
		if len(layer.Values) > 0 {
			sources = append(sources, layer.Source)
		}
	}
	values := mergeLayers(layers)
	if _, configured := values[keyOutputs]; !configured {
		return Result{Sources: sources}, nil
	}

	settings, err := toSettings(values)
	if err != nil {
		return Result{}, err
	}
	settings, firstSync, err := workspacesettings.ResolveSettings(settings)
	if err != nil {
		return Result{}, err
	}

	return Result{Configured: true, FirstSync: firstSync, Settings: settings, Sources: sources}, nil
}
