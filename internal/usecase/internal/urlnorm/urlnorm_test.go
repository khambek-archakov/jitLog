package urlnorm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/khambek-archakov/jitLog/internal/usecase/internal/urlnorm"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{
			name: "lowercases the host",
			in:   "https://EXAMPLE.com/page",
			want: "https://example.com/page",
			ok:   true,
		},
		{
			name: "strips utm params but keeps real ones",
			in:   "https://example.com/page?utm_source=telegram&utm_medium=bot&id=5",
			want: "https://example.com/page?id=5",
			ok:   true,
		},
		{
			name: "strips the fragment",
			in:   "https://example.com/page#section",
			want: "https://example.com/page",
			ok:   true,
		},
		{
			name: "strips a trailing slash",
			in:   "https://example.com/page/",
			want: "https://example.com/page",
			ok:   true,
		},
		{
			name: "bare domain with trailing slash normalizes to no slash",
			in:   "https://example.com/",
			want: "https://example.com",
			ok:   true,
		},
		{
			name: "http is accepted",
			in:   "http://example.com",
			want: "http://example.com",
			ok:   true,
		},
		{
			name: "missing scheme is rejected",
			in:   "example.com/page",
			ok:   false,
		},
		{
			name: "non-http(s) scheme is rejected",
			in:   "ftp://example.com",
			ok:   false,
		},
		{
			name: "garbage is rejected",
			in:   "not a url",
			ok:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := urlnorm.Normalize(tc.in)

			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
