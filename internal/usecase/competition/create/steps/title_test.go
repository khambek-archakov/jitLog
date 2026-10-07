package steps_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/create/steps"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackCompetitionAdd mirrors steps' own private constant.
const callbackCompetitionAdd = "competition:add"

func TestTitleStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID int64 = 777

	u := &model.User{ID: 42}

	tests := []struct {
		name     string
		d        *model.UserCompetitionDraft
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockdraftRepo)
		expected func(t assert.TestingT, d *model.UserCompetitionDraft, err error)
	}{
		{
			name: "kickoff asks the title question",
			d:    &model.UserCompetitionDraft{ID: 1},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackCompetitionAdd},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, d *model.UserCompetitionDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "stray callback is just acknowledged",
			d:    &model.UserCompetitionDraft{ID: 1},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, d *model.UserCompetitionDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:    "no message, no callback — no-op",
			d:       &model.UserCompetitionDraft{ID: 1},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t assert.TestingT, d *model.UserCompetitionDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "blank title is re-asked",
			d:    &model.UserCompetitionDraft{ID: 1},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "   "},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, d *model.UserCompetitionDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.Title)
			},
		},

		{
			name: "too-long title is re-asked",
			d:    &model.UserCompetitionDraft{ID: 1},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: stringOfLen(101)},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, d *model.UserCompetitionDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.Title)
			},
		},

		{
			name: "failed to persist title",
			d:    &model.UserCompetitionDraft{ID: 1},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "Moscow Open"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().UpdateDraft(gomock.Any(), gomock.Any()).Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, d *model.UserCompetitionDraft, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "valid title moves to the date question",
			d:    &model.UserCompetitionDraft{ID: 1},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "  Moscow Open  "},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().UpdateDraft(gomock.Any(), gomock.Any()).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, d *model.UserCompetitionDraft, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "Moscow Open", *d.Title)
				assert.Equal(t, model.UserCompetitionDraftStepAwaitingDate, d.Step)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockSender := NewMocksender(ctrl)
			mockRepo := NewMockdraftRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			step := steps.NewTitle(mockSender, mockRepo)

			err := step.Handle(context.Background(), u, tc.d, tc.in)

			tc.expected(t, tc.d, err)
		})
	}
}

func stringOfLen(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = 'a'
	}

	return string(b)
}
