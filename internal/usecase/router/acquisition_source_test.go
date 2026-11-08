package router

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcquisitionSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		want    string
		ok      bool
	}{
		{name: "a simple label", payload: "src_ads", want: "src_ads", ok: true},
		{name: "digits and underscores", payload: "src_ad_1_2", want: "src_ad_1_2", ok: true},
		{name: "a single-char label is the shortest allowed", payload: "src_a", want: "src_a", ok: true},
		{name: "a 32-char label is the longest allowed", payload: "src_" + strings.Repeat("a", 32), ok: true},
		{name: "a 33-char label is one too long", payload: "src_" + strings.Repeat("a", 33), ok: false},
		{name: "empty payload (plain /start)", payload: "", ok: false},
		{name: "missing the src_ prefix", payload: "ads", ok: false},
		{name: "uppercase is rejected", payload: "src_ADS", ok: false},
		{name: "a space is rejected", payload: "src_ad s", ok: false},
		{name: "no label after the prefix", payload: "src_", ok: false},
		{name: "some other bot's deep-link shape", payload: "ref_12345", ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := acquisitionSource(tc.payload)

			if !tc.ok {
				assert.Nil(t, got)

				return
			}

			require.NotNil(t, got)

			want := tc.want
			if want == "" {
				want = tc.payload
			}

			assert.Equal(t, want, *got)
		})
	}
}
