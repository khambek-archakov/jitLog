package chain

// continuations groups every active-draft handoff — training's and
// schedule's (both the create-draft and the edit-draft kind), profile's
// own edit-draft, and competition's (both kinds too) — behind one Handler.
// These do NOT split by domain the way training/schedule/competition's own
// sub-packages do: each one claims *any* input (not just a matching
// callback) once its own draft exists, so whichever draft a user is
// "inside" must keep claiming every update ahead of every trigger until
// it's finished or cancelled. The order here is fixed and must not change:
// training's draft, training's edit-draft, schedule's draft, schedule's
// edit-draft, profile's edit-draft, competition's draft, competition's
// edit-draft.
func continuations(d Dependencies) Handler {
	return sequence{
		&draftContinuation{create: d.TrainingCreate, drafts: d.TrainingDraft},
		&editDraftContinuation{update: d.TrainingUpdate, edits: d.TrainingEditDraft},
		&scheduleDraftContinuation{create: d.ScheduleCreate, drafts: d.ScheduleDraft},
		&scheduleEditDraftContinuation{update: d.ScheduleUpdate, edits: d.ScheduleEditDraft},
		&profileEditDraftContinuation{profile: d.Profile, edits: d.ProfileEditDraft},
		&competitionDraftContinuation{create: d.CompetitionCreate, drafts: d.CompetitionDraft},
		&competitionEditDraftContinuation{update: d.CompetitionUpdate, edits: d.CompetitionEditDraft},
	}
}
