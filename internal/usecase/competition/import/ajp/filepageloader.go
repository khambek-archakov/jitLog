package ajp

import (
	"context"
	"fmt"
	"os"
	"time"
)

// staleFileWarningAge is how old a --from-file copy can be before Load
// warns about it on stderr — it still gets used, just flagged.
const staleFileWarningAge = 72 * time.Hour

// FilePageLoader reads a previously saved copy of the page instead of
// making any network request at all. It exists for when Cloudflare's own
// challenge blocks the server's own IP: a human opens the real page in
// their own browser (solving that challenge honestly, as a person, not
// this importer bypassing anything), saves it, and copies it over — see
// deploy/systemd/README.md's "Импорт из файла". Reports exactly the same
// (body, 200, nil) shape HTTPPageLoader does on success, so Run itself
// never needs to know which one it's talking to.
type FilePageLoader struct {
	path string
}

func NewFilePageLoader(path string) *FilePageLoader {
	return &FilePageLoader{path: path}
}

func (l *FilePageLoader) Load(_ context.Context) (string, int, error) {
	info, err := os.Stat(l.path)
	if err != nil {
		return "", 0, fmt.Errorf("%s: %w", l.path, err)
	}

	if msg := staleFileWarning(l.path, info.ModTime(), time.Now()); msg != "" {
		fmt.Fprintln(os.Stderr, msg)
	}

	data, err := os.ReadFile(l.path)
	if err != nil {
		return "", 0, fmt.Errorf("%s: %w", l.path, err)
	}

	return string(data), 200, nil
}

// staleFileWarning is pulled out as its own pure function purely so the
// 3-day cutoff is testable without touching the filesystem.
func staleFileWarning(path string, modTime, now time.Time) string {
	if now.Sub(modTime) <= staleFileWarningAge {
		return ""
	}

	return fmt.Sprintf("%s: файл старше 3 суток", path)
}
