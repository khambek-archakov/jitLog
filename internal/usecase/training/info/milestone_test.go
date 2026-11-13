package info

import "testing"

// TestIsMilestone is a white-box test (package info, not info_test) so it
// can exercise isMilestone directly, by name, per the spec's own worked
// set: 9 no; 10 yes; 11 no; 24 no; 25 yes; 50 yes; 99 no; 100 yes; 125
// no; 150 yes; 175 no; 200 yes; 250 no; 300 yes; 400 yes.
func TestIsMilestone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int
		want bool
	}{
		{9, false},
		{10, true},
		{11, false},
		{24, false},
		{25, true},
		{50, true},
		{99, false},
		{100, true},
		{125, false},
		{150, true},
		{175, false},
		{200, true},
		{250, false},
		{300, true},
		{400, true},
	}

	for _, tc := range tests {
		if got := isMilestone(tc.n); got != tc.want {
			t.Errorf("isMilestone(%d) = %v, want %v", tc.n, got, tc.want)
		}
	}
}
