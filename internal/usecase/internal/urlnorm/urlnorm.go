// Package urlnorm normalizes a tournament URL into a stable, comparable
// form. It lives under internal/usecase/internal (not inside the
// competition scenario itself) because stage 2's catalog — a future,
// separate usecase package — reuses the exact same function for
// competition_source.url's dedup (its UNIQUE constraint only works if
// equivalent URLs always normalize identically).
package urlnorm

import (
	"net/url"
	"strings"
)

// Normalize lowercases the host, strips any utm_* query parameters and the
// fragment, and drops a trailing slash from the path. It rejects anything
// that isn't a well-formed http(s) URL.
func Normalize(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", false
	}

	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""

	q := u.Query()
	for key := range q {
		if strings.HasPrefix(strings.ToLower(key), "utm_") {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()

	u.Path = strings.TrimSuffix(u.Path, "/")

	return u.String(), true
}
