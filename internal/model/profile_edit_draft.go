package model

import "time"

// ProfileEditDraft marks that the next free-text message from this user
// should be read as their new age — the only profile field editable via
// free text (belt is always picked from buttons). There's no Field to
// track and no separate target id: the draft's mere existence is the
// state, and its target is always the current resolved user.
type ProfileEditDraft struct {
	UserID    int64
	CreatedAt time.Time
}
