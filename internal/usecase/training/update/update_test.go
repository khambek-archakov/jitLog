package update_test

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
	"github.com/khambek-archakov/jitLog/internal/usecase/training/update"
)

const chatID, userID, messageID int64 = 777, 42, 555

var date = time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

func baseTraining() *model.Training {
	return &model.Training{ID: 7, UserID: userID, Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 60}
}

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MocktrainingRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no callback — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed id is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:noop"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "training not found",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)

				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Тренировка не найдена.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "training belongs to someone else",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				t := baseTraining()
				t.UserID = userID + 1
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(t, nil)

				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Тренировка не найдена.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch training",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "bare id shows the edit menu",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "✏️ Что изменить?", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 6)
						assert.Equal(t, "training:edit:7:date", kb[0][0].Data)
						assert.Equal(t, "training:edit:7:type", kb[1][0].Data)
						assert.Equal(t, "training:edit:7:duration", kb[2][0].Data)
						assert.Equal(t, "training:edit:7:rounds", kb[3][0].Data)
						assert.Equal(t, "training:edit:7:notes", kb[4][0].Data)
						assert.Equal(t, "training:view:7", kb[5][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "date sub-action shows the calendar",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7:date",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						// Cancel row goes straight back to the card via info.
						last := kb[len(kb)-1]
						assert.Equal(t, "training:view:7", last[0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a calendar day updates the date and shows the card",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "training:edit:7:date:pick:2026-03-20",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				repo.EXPECT().
					UpdateTraining(gomock.Any(), int64(7), gomock.Any(), model.TrainingTypeGi, int32(60), gomock.Nil(), gomock.Nil()).
					DoAndReturn(func(_ context.Context, _ int64, date time.Time, tt model.TrainingType, d int32, _ *int16, n *string) (*model.Training, error) {
						assert.Equal(t, "2026-03-20", date.Format("2006-01-02"))

						return &model.Training{ID: 7, Date: date, TrainingType: tt, DurationMinutes: d}, nil
					})

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed calendar day is just acknowledged",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7:date:pick:not-a-date",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "type sub-action shows the type keyboard",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7:type",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "Какой тип тренировки?", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						assert.Equal(t, "training:edit:7:type:gi", kb[0][0].Data)
						assert.Equal(t, "training:edit:7:type:no_gi", kb[0][1].Data)
						assert.Equal(t, "training:edit:7:type:open_mat", kb[1][0].Data)
						assert.Equal(t, "training:edit:7", kb[2][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a type updates it and shows the card",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "training:edit:7:type:no_gi",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				repo.EXPECT().
					UpdateTraining(gomock.Any(), int64(7), date, model.TrainingTypeNoGi, int32(60), gomock.Nil(), gomock.Nil()).
					Return(&model.Training{ID: 7, Date: date, TrainingType: model.TrainingTypeNoGi, DurationMinutes: 60}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "🥷 No-Gi")
						assert.Equal(t, "training:edit:7", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown type token is just acknowledged",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7:type:bogus",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "duration sub-action shows the duration keyboard",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "training:edit:7:duration",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "⏱ Сколько длилась тренировка?", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						assert.Equal(t, "training:edit:7:duration:60", kb[0][0].Data)
						assert.Equal(t, "training:edit:7:duration:other", kb[1][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a quick duration updates it and shows the card",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "training:edit:7:duration:90",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				repo.EXPECT().
					UpdateTraining(gomock.Any(), int64(7), date, model.TrainingTypeGi, int32(90), gomock.Nil(), gomock.Nil()).
					Return(&model.Training{ID: 7, Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 90}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "duration other sets a pending edit draft and nudges for text",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7:duration:other",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				repo.EXPECT().
					SetEditDraft(gomock.Any(), userID, int64(7), model.TrainingEditFieldDuration).
					Return(nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().Send(gomock.Any(), chatID, "Напиши длительность в минутах, например: 45").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "notes sets a pending edit draft and nudges for text",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7:notes",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				repo.EXPECT().
					SetEditDraft(gomock.Any(), userID, int64(7), model.TrainingEditFieldNotes).
					Return(nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().Send(gomock.Any(), chatID, "Напиши новую заметку:").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "rounds sub-action shows the rounds keyboard",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "training:edit:7:rounds",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "Сколько было раундов?", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						assert.Equal(t, "training:edit:7:rounds:3", kb[0][0].Data)
						assert.Equal(t, "training:edit:7:rounds:5", kb[0][1].Data)
						assert.Equal(t, "training:edit:7:rounds:7", kb[0][2].Data)
						assert.Equal(t, "training:edit:7:rounds:10", kb[0][3].Data)
						assert.Equal(t, "training:edit:7:rounds:other", kb[1][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a quick rounds count updates it and shows the card",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "training:edit:7:rounds:5",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				var rounds int16 = 5

				repo.EXPECT().
					UpdateTraining(gomock.Any(), int64(7), date, model.TrainingTypeGi, int32(60), &rounds, gomock.Nil()).
					Return(&model.Training{ID: 7, Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 60, Rounds: &rounds}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "rounds other sets a pending edit draft and nudges for text",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7:rounds:other",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				repo.EXPECT().
					SetEditDraft(gomock.Any(), userID, int64(7), model.TrainingEditFieldRounds).
					Return(nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					Send(gomock.Any(), chatID, "Напиши количество раундов цифрами, например: 6").
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown rounds token is just acknowledged",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7:rounds:bogus",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown action is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7:bogus"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockSender := NewMocksender(ctrl)
			mockRepo := NewMocktrainingRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := update.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), userID, tc.in)

			tc.expected(t, err)
		})
	}
}

func TestUseCase_Continue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		d        *model.TrainingEditDraft
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MocktrainingRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no message — no-op",
			d:       &model.TrainingEditDraft{UserID: userID, TrainingID: 7, Field: model.TrainingEditFieldNotes},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "training gone meanwhile just clears the draft",
			d:    &model.TrainingEditDraft{UserID: userID, TrainingID: 7, Field: model.TrainingEditFieldNotes},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "note"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unparsable duration is re-asked",
			d:    &model.TrainingEditDraft{UserID: userID, TrainingID: 7, Field: model.TrainingEditFieldDuration},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "ninety"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "valid duration finishes the edit",
			d:    &model.TrainingEditDraft{UserID: userID, TrainingID: 7, Field: model.TrainingEditFieldDuration},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: " 45 "},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				repo.EXPECT().
					UpdateTraining(gomock.Any(), int64(7), date, model.TrainingTypeGi, int32(45), gomock.Nil(), gomock.Nil()).
					Return(&model.Training{ID: 7, Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 45}, nil)

				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)

				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "notes finishes the edit",
			d:    &model.TrainingEditDraft{UserID: userID, TrainingID: 7, Field: model.TrainingEditFieldNotes},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "  new note  "},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				repo.EXPECT().
					UpdateTraining(gomock.Any(), int64(7), date, model.TrainingTypeGi, int32(60), gomock.Nil(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ time.Time, _ model.TrainingType, _ int32, _ *int16, notes *string) (*model.Training, error) {
						require.NotNil(t, notes)
						assert.Equal(t, "new note", *notes)

						return &model.Training{ID: 7, Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 60, Notes: notes}, nil
					})

				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)

				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "zero rounds is allowed — no sparring that day is a valid answer",
			d:    &model.TrainingEditDraft{UserID: userID, TrainingID: 7, Field: model.TrainingEditFieldRounds},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "0"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				var zero int16

				repo.EXPECT().
					UpdateTraining(gomock.Any(), int64(7), date, model.TrainingTypeGi, int32(60), &zero, gomock.Nil()).
					Return(&model.Training{ID: 7, Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 60, Rounds: &zero}, nil)

				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)

				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "valid rounds finishes the edit",
			d:    &model.TrainingEditDraft{UserID: userID, TrainingID: 7, Field: model.TrainingEditFieldRounds},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: " 6 "},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				var six int16 = 6

				repo.EXPECT().
					UpdateTraining(gomock.Any(), int64(7), date, model.TrainingTypeGi, int32(60), &six, gomock.Nil()).
					Return(&model.Training{ID: 7, Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 60, Rounds: &six}, nil)

				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)

				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "rounds of 50 or more is re-asked",
			d:    &model.TrainingEditDraft{UserID: userID, TrainingID: 7, Field: model.TrainingEditFieldRounds},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "50"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "negative rounds is re-asked",
			d:    &model.TrainingEditDraft{UserID: userID, TrainingID: 7, Field: model.TrainingEditFieldRounds},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "-1"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unparsable rounds is re-asked",
			d:    &model.TrainingEditDraft{UserID: userID, TrainingID: 7, Field: model.TrainingEditFieldRounds},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "six"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to update training",
			d:    &model.TrainingEditDraft{UserID: userID, TrainingID: 7, Field: model.TrainingEditFieldDuration},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "45"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(baseTraining(), nil)

				repo.EXPECT().
					UpdateTraining(gomock.Any(), int64(7), date, model.TrainingTypeGi, int32(45), gomock.Nil(), gomock.Nil()).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockSender := NewMocksender(ctrl)
			mockRepo := NewMocktrainingRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := update.New(mockSender, mockRepo)

			err := uc.Continue(context.Background(), tc.d, tc.in)

			tc.expected(t, err)
		})
	}
}
