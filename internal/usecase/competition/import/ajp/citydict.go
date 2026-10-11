package ajp

import (
	"fmt"
	"os"
	"strings"
)

// LoadCityDictionary reads a plain "English=Russian" per-line file (blank
// lines and lines starting with # are ignored) into a lookup map — see
// configs/ajp_cities.txt for the shipped starting dictionary.
func LoadCityDictionary(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read city dictionary: %w", err)
	}

	dict := make(map[string]string)

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		dict[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return dict, nil
}

// translateCity looks city up in dict, falling back to city itself
// unchanged (AJP's own English spelling) when there's no entry — ok
// reports whether a translation was found at all, so the caller can flag
// "город без перевода" without the fallback itself looking like success.
func translateCity(dict map[string]string, city string) (translated string, ok bool) {
	if ru, found := dict[city]; found {
		return ru, true
	}

	return city, false
}
