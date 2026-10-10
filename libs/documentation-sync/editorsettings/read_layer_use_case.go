package editorsettings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

// ReadLayer reads one settings.json and keeps its LoreMaster keys. A missing file is an
// empty layer, not an error. The file may hold comments and trailing commas, as an
// editor's settings do. A key that looks like a secret is refused: credentials live in
// the editor's secret store, never in a settings file that is synced or committed.
func ReadLayer(path string) (Layer, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Layer{Source: path}, nil
	}
	if err != nil {
		return Layer{}, err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(stripJSONC(content), &all); err != nil {
		return Layer{}, fmt.Errorf("%s: %w", path, err)
	}

	layer := Layer{Source: path, Values: map[string]json.RawMessage{}}
	var unknown []string
	for key, value := range all {
		if !strings.HasPrefix(key, keyPrefix) || isExtensionOnly(key) {
			continue
		}
		if looksLikeSecret(key) {
			return Layer{}, fmt.Errorf("%s: %q looks like a secret; settings files are synced and committed, the editor keeps credentials in its secret store", path, key)
		}
		if !configKeys[key] {
			unknown = append(unknown, key)

			continue
		}
		layer.Values[key] = value
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)

		return Layer{}, fmt.Errorf("%s: unknown LoreMaster setting %s", path, strings.Join(unknown, ", "))
	}

	return layer, nil
}

func looksLikeSecret(key string) bool {
	last := strings.ToLower(key[strings.LastIndex(key, ".")+1:])
	for _, secret := range []string{"token", "apitoken", "password", "pat", "secret", "apikey", "credential", "credentials"} {
		if last == secret {
			return true
		}
	}

	return false
}
