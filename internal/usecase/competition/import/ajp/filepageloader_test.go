package ajp_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/usecase/competition/import/ajp"
)

func TestFilePageLoader_Load(t *testing.T) {
	t.Parallel()

	t.Run("reads the file back verbatim, status 200", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "page.html")
		require.NoError(t, os.WriteFile(path, []byte("<html>hello</html>"), 0o600))

		loader := ajp.NewFilePageLoader(path)

		body, status, err := loader.Load(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 200, status)
		assert.Equal(t, "<html>hello</html>", body)
	})

	t.Run("missing file is an error, not a panic", func(t *testing.T) {
		t.Parallel()

		loader := ajp.NewFilePageLoader(filepath.Join(t.TempDir(), "does-not-exist.html"))

		_, _, err := loader.Load(context.Background())
		assert.Error(t, err)
	})

	t.Run("empty file reads back as an empty body with no error", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "empty.html")
		require.NoError(t, os.WriteFile(path, nil, 0o600))

		loader := ajp.NewFilePageLoader(path)

		body, status, err := loader.Load(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 200, status)
		assert.Empty(t, body)
		// Run itself turns an empty body into the same structural
		// monitoring failure a missing section#upcoming-events gets —
		// see TestRun_fromFile_monitoringFailures.
	})
}

// failingTransport fails every request it ever sees — used to prove file
// mode never reaches for the network at all, not even by accident.
type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("no network calls are allowed in file mode")
}

func TestFilePageLoader_NoNetworkCalls(t *testing.T) {
	original := http.DefaultTransport
	http.DefaultTransport = failingTransport{}

	t.Cleanup(func() { http.DefaultTransport = original })

	path := filepath.Join(t.TempDir(), "page.html")
	require.NoError(t, os.WriteFile(path, []byte("<html>hello</html>"), 0o600))

	loader := ajp.NewFilePageLoader(path)

	body, status, err := loader.Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 200, status)
	assert.Equal(t, "<html>hello</html>", body)
}

// captureStderr swaps os.Stderr for a pipe for the duration of fn and
// returns everything written to it — used to prove the staleness warning
// (and nothing else) actually lands on stderr.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stderr

	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stderr = w

	fn()

	require.NoError(t, w.Close())

	os.Stderr = original

	out, err := io.ReadAll(r)
	require.NoError(t, err)

	return string(out)
}

func TestStaleFileWarning(t *testing.T) {
	// Relative to the real clock, not a fixed calendar date — Load's own
	// staleness check (see filepageloader.go) compares the file's mtime
	// against time.Now() for real, so pinning this fixture to a specific
	// past date eventually drifts more than staleFileWarningAge away from
	// "now" and makes the "fresh" case below wrongly look stale.
	now := time.Now()

	path := filepath.Join(t.TempDir(), "page.html")
	require.NoError(t, os.WriteFile(path, []byte("<html></html>"), 0o600))

	t.Run("fresh file — no warning on stderr, still loads fine", func(t *testing.T) {
		fresh := now.Add(-1 * time.Hour)
		require.NoError(t, os.Chtimes(path, fresh, fresh))

		loader := ajp.NewFilePageLoader(path)

		var status int

		stderr := captureStderr(t, func() {
			_, s, err := loader.Load(context.Background())
			require.NoError(t, err)
			status = s
		})

		assert.Equal(t, 200, status)
		assert.Empty(t, stderr)
	})

	t.Run("file older than 3 days still loads, just gets flagged on stderr", func(t *testing.T) {
		old := now.Add(-96 * time.Hour)
		require.NoError(t, os.Chtimes(path, old, old))

		loader := ajp.NewFilePageLoader(path)

		var body string

		stderr := captureStderr(t, func() {
			b, status, err := loader.Load(context.Background())
			require.NoError(t, err)
			assert.Equal(t, 200, status)
			body = b
		})

		assert.Equal(t, "<html></html>", body)
		assert.Contains(t, stderr, "файл старше 3 суток")
	})
}
