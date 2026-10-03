package chain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// TestDomainWiring is a smoke test over Chain's own internal assembly, not
// over training/schedule's internal routing logic (that's covered
// exhaustively by each sub-package's own domain_test.go). Training and
// schedule each have several same-shaped dependencies (trainingInfo/
// scheduleInfo, trainingUpdate/scheduleUpdate, ... all just
// Handle(ctx, userID, in) error) that the Go compiler cannot catch if
// swapped in chain.New's own wiring — only a real call through
// chain.New(...).Handle(...) proves the right concrete mock receives the
// call.
func TestDomainWiring(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	tests := []struct {
		name string
		in   dto.Input
		call func(m mocks, in dto.Input)
	}{
		{
			name: "training:view:{id} reaches trainingInfo, not scheduleInfo",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:view:7"},
			call: func(m mocks, in dto.Input) {
				m.info.EXPECT().Handle(gomock.Any(), userID, in).Return(nil)
			},
		},
		{
			name: "training:history:page:{n} reaches trainingHistory",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:history:page:0"},
			call: func(m mocks, in dto.Input) {
				m.history.EXPECT().Handle(gomock.Any(), userID, in).Return(nil)
			},
		},
		{
			name: "training:edit:* reaches trainingUpdate, not scheduleUpdate",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7"},
			call: func(m mocks, in dto.Input) {
				m.update.EXPECT().Handle(gomock.Any(), userID, in).Return(nil)
			},
		},
		{
			name: "training:delete:* reaches trainingDelete, not scheduleDelete",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:delete:7"},
			call: func(m mocks, in dto.Input) {
				m.delete.EXPECT().Handle(gomock.Any(), userID, in).Return(nil)
			},
		},
		{
			name: "schedule:view:{id} reaches scheduleInfo, not trainingInfo",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:view:7"},
			call: func(m mocks, in dto.Input) {
				m.scheduleInfo.EXPECT().Handle(gomock.Any(), userID, in).Return(nil)
			},
		},
		{
			name: "schedule:edit:* reaches scheduleUpdate, not trainingUpdate",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7"},
			call: func(m mocks, in dto.Input) {
				m.scheduleUpdate.EXPECT().Handle(gomock.Any(), userID, in).Return(nil)
			},
		},
		{
			name: "schedule:delete:* reaches scheduleDelete, not trainingDelete",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:delete:7"},
			call: func(m mocks, in dto.Input) {
				m.scheduleDelete.EXPECT().Handle(gomock.Any(), userID, in).Return(nil)
			},
		},
		{
			name: "schedule:list reaches scheduleList",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:list"},
			call: func(m mocks, in dto.Input) {
				m.scheduleList.EXPECT().Handle(gomock.Any(), userID, in).Return(nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := run(t, u, tc.in, func(m mocks) {
				noActiveDrafts(m, userID)
				tc.call(m, tc.in)
			})

			assert.NoError(t, err)
		})
	}
}
