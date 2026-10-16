// Package timezone is the single source of truth for the IANA zones the
// reminder feature's timezone picker offers — schedule/list (enabling
// reminders) and profile (changing zone later) both build their picker
// keyboard from Rows and validate a tapped callback through ByCode, so the
// set of allowed zones never drifts between the two screens.
package timezone

import "github.com/khambek-archakov/jitLog/internal/usecase/dto"

// Zone is one of the picker's options — IANA (what gets stored on
// model.User.Timezone), Code (the short token riding in callback data,
// since a raw IANA name is needlessly long for that) and Label (what the
// button/profile line shows).
type Zone struct {
	IANA  string
	Code  string
	Label string
}

// zones is deliberately this short, fixed list — no GeoNames, no city
// picker, no further zones. See the reminder feature's own spec for why.
// Kaliningrad was removed by explicit request (2026-10-16).
var zones = []Zone{
	{IANA: "Europe/Moscow", Code: "msk", Label: "Москва +3"},
	{IANA: "Europe/Samara", Code: "sam", Label: "Самара, Баку, Ереван +4"},
	{IANA: "Asia/Yekaterinburg", Code: "yek", Label: "Екатеринбург, Алматы +5"},
}

// ByCode resolves a short callback token (e.g. "msk") back to its Zone.
// Any other value — a stale/malformed callback — must be rejected by the
// caller rather than falling back to some default zone.
func ByCode(code string) (Zone, bool) {
	for _, z := range zones {
		if z.Code == code {
			return z, true
		}
	}

	return Zone{}, false
}

// LabelForIANA looks up the display label for a stored IANA zone name —
// used by profile's status line. A name outside the known zones (in
// practice, only possible if the column were ever hand-edited, or a user
// picked a zone before it was removed from the list) has no label to show.
func LabelForIANA(iana string) (string, bool) {
	for _, z := range zones {
		if z.IANA == iana {
			return z.Label, true
		}
	}

	return "", false
}

// Rows lays the zones out one per row, in Zone's own declared order —
// callbackPrefix is prepended to each zone's Code to build that button's
// callback data, letting two different screens (schedule:tz:{code} vs
// profile:tz:{code}) reuse this exact same layout without colliding.
func Rows(callbackPrefix string) dto.Keyboard {
	kb := make(dto.Keyboard, 0, len(zones))

	for _, z := range zones {
		kb = append(kb, dto.Row(dto.Button{Label: z.Label, Data: callbackPrefix + z.Code}))
	}

	return kb
}
