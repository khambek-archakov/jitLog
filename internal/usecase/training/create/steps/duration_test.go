package steps_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/create/steps"
)

// callbackDuration90 and callbackDurationOther mirror steps' own private
// constants — black-box tests have to know the literal values.
const (
	callbackDuration90    = "training:duration:90"
	callbackDurationOther = "training:duration:other"
)

func TestDurationStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		d       *model.TrainingDraft
		in      dto.Input
		prepare func(
			sender *Mocksender,
			repo *MockdraftRepo,
		)
		expected func(t assert.TestingT, d *model.TrainingDraft, err error)
	}{
		{
			name: "back goes to type step, duration untouched",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackBack},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.TrainingDraftStepAwaitingType, d.Step)
				assert.Nil(t, d.DurationMinutes)
			},
		},

		{
			name: "stray callback is just acknowledged",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "\"другое\" just nudges, doesn't finish",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackDurationOther},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					Send(gomock.Any(), chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.DurationMinutes)
			},
		},

		{
			name:    "no message, no callback — no-op",
			d:       &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unparsable duration is re-asked",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "ninety"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					Send(gomock.Any(), chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.DurationMinutes)
			},
		},

		{
			name: "zero duration is re-asked",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "0"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					Send(gomock.Any(), chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.DurationMinutes)
			},
		},

		{
			name: "quick button finishes the wizard — training saved, draft cleared",
			d:    &model.TrainingDraft{ID: 1, UserID: userID, Date: &date, TrainingType: model.TrainingTypeGi, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackDuration90},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				repo.EXPECT().
					CreateTraining(gomock.Any(), userID, date, model.TrainingTypeGi, int32(90), gomock.Nil()).
					Return(&model.Training{ID: 7, Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 90}, nil)

				repo.EXPECT().
					DeleteDraft(gomock.Any(), userID).
					Return(nil)

				confirmation := sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, "Вот что я умею:", gomock.Any()).
					After(confirmation).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "free-text duration finishes the wizard too",
			d:    &model.TrainingDraft{ID: 1, UserID: userID, Date: &date, TrainingType: model.TrainingTypeNoGi, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: " 60 "},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateTraining(gomock.Any(), userID, date, model.TrainingTypeNoGi, int32(60), gomock.Nil()).
					Return(&model.Training{ID: 7, Date: date, TrainingType: model.TrainingTypeNoGi, DurationMinutes: 60}, nil)

				repo.EXPECT().
					DeleteDraft(gomock.Any(), userID).
					Return(nil)

				confirmation := sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, "Вот что я умею:", gomock.Any()).
					After(confirmation).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "confirmation shows a compact summary with rounds/notes/edit/delete buttons",
			d:    &model.TrainingDraft{ID: 1, UserID: userID, Date: &date, TrainingType: model.TrainingTypeOpenMat, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackDuration90},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				repo.EXPECT().
					CreateTraining(gomock.Any(), userID, date, model.TrainingTypeOpenMat, int32(90), gomock.Nil()).
					Return(&model.Training{ID: 7, Date: date, TrainingType: model.TrainingTypeOpenMat, DurationMinutes: 90}, nil)

				repo.EXPECT().
					DeleteDraft(gomock.Any(), userID).
					Return(nil)

				confirmation := sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, text string, keyboard dto.Keyboard) error {
						assert.Contains(t, text, "✅ Тренировка сохранена!")
						assert.Contains(t, text, "15 марта")
						assert.NotContains(t, text, "2026")
						assert.Contains(t, text, "🤼 Open Mat")
						assert.Contains(t, text, "90 минут")

						require.Len(t, keyboard, 2)
						assert.Equal(t, "➕ Раунды", keyboard[0][0].Label)
						assert.Equal(t, "training:edit:7:rounds", keyboard[0][0].Data)
						assert.Equal(t, "📝 Заметка", keyboard[0][1].Label)
						assert.Equal(t, "training:edit:7:notes", keyboard[0][1].Data)
						assert.Equal(t, "✏️ Изменить", keyboard[1][0].Label)
						assert.Equal(t, "training:edit:7", keyboard[1][0].Data)
						assert.Equal(t, "🗑 Удалить", keyboard[1][1].Label)
						assert.Equal(t, "training:delete:7", keyboard[1][1].Data)

						return nil
					})

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, "Вот что я умею:", gomock.Any()).
					After(confirmation).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to create training",
			d:    &model.TrainingDraft{ID: 1, UserID: userID, Date: &date, TrainingType: model.TrainingTypeGi, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackDuration90},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateTraining(gomock.Any(), userID, date, model.TrainingTypeGi, int32(90), gomock.Nil()).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.Error(t, err)
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

			step := steps.NewDuration(mockSender, mockRepo)

			err := step.Handle(context.Background(), tc.d, tc.in)

			tc.expected(t, tc.d, err)
		})
	}
}
