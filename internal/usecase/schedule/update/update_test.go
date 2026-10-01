package update_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/update"
)

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	slot := &model.ScheduleSlot{ID: 7, UserID: userID, DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi}

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockslotRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no callback — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed id is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:nope"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "slot not found",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				sender.EXPECT().AnswerCallbackWithText("cb-1", "Слот не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "slot belongs to someone else",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				other := *slot
				other.UserID = userID + 1
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(&other, nil)
				sender.EXPECT().AnswerCallbackWithText("cb-1", "Слот не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch slot",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "bare id shows the edit menu",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), "✏️ Что изменить?", gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 4)
						assert.Equal(t, "schedule:edit:7:day", kb[0][0].Data)
						assert.Equal(t, "schedule:edit:7:time", kb[1][0].Data)
						assert.Equal(t, "schedule:edit:7:type", kb[2][0].Data)
						assert.Equal(t, "schedule:view:7", kb[3][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "day submenu shows a 7-day keyboard",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:day",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), "📅 Какой день недели?", gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb[0], 7)
						assert.Equal(t, "schedule:edit:7:day:1", kb[0][0].Data)
						assert.Equal(t, "schedule:edit:7:day:7", kb[0][6].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a day applies the update and shows the card",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:day:3",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				repo.EXPECT().
					UpdateSlot(gomock.Any(), int64(7), int16(3), int16(19*60), model.TrainingTypeGi).
					Return(&model.ScheduleSlot{ID: 7, UserID: userID, DayOfWeek: 3, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi}, nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "out of range day token is just acknowledged",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:day:9",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to apply day update",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:day:3",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				repo.EXPECT().
					UpdateSlot(gomock.Any(), int64(7), int16(3), int16(19*60), model.TrainingTypeGi).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "time submenu shows presets and Другое",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:time",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), "🕐 Во сколько?", gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 4)
						assert.Equal(t, "Другое", kb[2][0].Label)
						assert.Equal(t, "schedule:edit:7:time:other", kb[2][0].Data)
						assert.Equal(t, "schedule:edit:7", kb[3][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "time preset applies the update",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:time:preset:2100",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				repo.EXPECT().
					UpdateSlot(gomock.Any(), int64(7), int16(1), int16(21*60), model.TrainingTypeGi).
					Return(&model.ScheduleSlot{ID: 7, UserID: userID, DayOfWeek: 1, TimeMinutes: 21 * 60, TrainingType: model.TrainingTypeGi}, nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, 0, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed time preset token is just acknowledged",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:time:preset:bogus",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "time other sets an edit draft and prompts for free text",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:time:other",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				repo.EXPECT().SetEditDraft(gomock.Any(), userID, int64(7)).Return(nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
				sender.EXPECT().Send(chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to set edit draft",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:time:other",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				repo.EXPECT().SetEditDraft(gomock.Any(), userID, int64(7)).Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "unknown time sub-callback is just acknowledged",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:time:junk",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "type submenu shows the type keyboard",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:type",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), "Какой тип тренировки?", gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, kb dto.Keyboard) error {
						assert.Equal(t, "schedule:edit:7:type:gi", kb[0][0].Data)
						assert.Equal(t, "schedule:edit:7:type:no_gi", kb[0][1].Data)
						assert.Equal(t, "schedule:edit:7:type:open_mat", kb[1][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a type applies the update",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:type:no_gi",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				repo.EXPECT().
					UpdateSlot(gomock.Any(), int64(7), int16(1), int16(19*60), model.TrainingTypeNoGi).
					Return(&model.ScheduleSlot{ID: 7, UserID: userID, DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeNoGi}, nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(chatID, 0, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown type token is just acknowledged",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:type:bogus",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown action is just acknowledged",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7:bogus",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
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
			mockRepo := NewMockslotRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := update.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), userID, tc.in)

			tc.expected(t, err)
		})
	}
}

func TestUseCase_Continue(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	draft := &model.ScheduleEditDraft{UserID: userID, SlotID: 7}
	slot := &model.ScheduleSlot{ID: 7, UserID: userID, DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi}

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockslotRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no message — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "slot gone — draft is cleaned up",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "18:30"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch slot",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "18:30"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "valid free-text time applies the update and clears the draft",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "18:30"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				repo.EXPECT().
					UpdateSlot(gomock.Any(), int64(7), int16(1), int16(18*60+30), model.TrainingTypeGi).
					Return(&model.ScheduleSlot{ID: 7, UserID: userID, DayOfWeek: 1, TimeMinutes: 18*60 + 30, TrainingType: model.TrainingTypeGi}, nil)
				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)

				sender.EXPECT().SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unparseable free text is rejected",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "not a time"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				sender.EXPECT().Send(chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to apply update",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "18:30"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				repo.EXPECT().
					UpdateSlot(gomock.Any(), int64(7), int16(1), int16(18*60+30), model.TrainingTypeGi).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "failed to delete edit draft",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "18:30"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				repo.EXPECT().
					UpdateSlot(gomock.Any(), int64(7), int16(1), int16(18*60+30), model.TrainingTypeGi).
					Return(&model.ScheduleSlot{ID: 7, UserID: userID, DayOfWeek: 1, TimeMinutes: 18*60 + 30, TrainingType: model.TrainingTypeGi}, nil)
				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(errors.New("fail"))
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
			mockRepo := NewMockslotRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := update.New(mockSender, mockRepo)

			err := uc.Continue(context.Background(), draft, tc.in)

			tc.expected(t, err)
		})
	}
}
