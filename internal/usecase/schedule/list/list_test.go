package list_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/list"
)

const (
	chatID, userID int64 = 777, 42
	messageID            = 555
)

func oneSlot() []*model.ScheduleSlot {
	return []*model.ScheduleSlot{{DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi}}
}

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		u        *model.User
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no callback — no-op",
			u:       &model.User{ID: userID},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "schedule:list shows the list, no toggle with no slots",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:list"},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {
				repo.EXPECT().ListSlots(gomock.Any(), userID).Return(nil, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, messageID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						for _, row := range kb {
							for _, b := range row {
								assert.NotEqual(t, "schedule:reminders:toggle", b.Data)
							}
						}

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "schedule:list shows the reminder toggle once a slot exists",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:list"},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {
				repo.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, messageID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Напоминания выключены")

						found := false
						for _, row := range kb {
							for _, b := range row {
								if b.Data == "schedule:reminders:toggle" {
									found = true
									assert.Equal(t, "🔔 Напоминание: выкл", b.Label)
								}
							}
						}
						assert.True(t, found)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to list slots",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:list"},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {
				repo.EXPECT().ListSlots(gomock.Any(), userID).Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "toggling on with a timezone already set enables immediately",
			u:    &model.User{ID: userID, Timezone: ptr("Europe/Moscow")},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:reminders:toggle"},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {
				users.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, u *model.User) error {
						assert.True(t, u.RemindersEnabled)

						return nil
					})
				repo.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, messageID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "Напоминания включены. Часовой пояс: Москва +3")

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "toggling on with no timezone shows the picker instead, leaves reminders off",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:reminders:toggle"},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, messageID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "выбери свой часовой пояс")

						require.Len(t, kb, 4) // one row per zone + cancel
						assert.Equal(t, "schedule:tz:set:msk", kb[0][0].Data)
						assert.Equal(t, "schedule:tz:set:sam", kb[1][0].Data)
						assert.Equal(t, "schedule:tz:set:yek", kb[2][0].Data)
						assert.Equal(t, "schedule:tz:cancel", kb[3][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "toggling off clears reminders_enabled, leaves timezone untouched",
			u:    &model.User{ID: userID, Timezone: ptr("Europe/Moscow"), RemindersEnabled: true},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:reminders:toggle"},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {
				users.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, u *model.User) error {
						assert.False(t, u.RemindersEnabled)
						require.NotNil(t, u.Timezone)
						assert.Equal(t, "Europe/Moscow", *u.Timezone)

						return nil
					})
				repo.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, messageID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a zone sets timezone and enables reminders",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:tz:set:yek"},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {
				users.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, u *model.User) error {
						require.NotNil(t, u.Timezone)
						assert.Equal(t, "Asia/Yekaterinburg", *u.Timezone)
						assert.True(t, u.RemindersEnabled)

						return nil
					})
				repo.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, messageID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "Часовой пояс: Екатеринбург, Алматы +5")

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "re-tapping an already-acted-on zone button is a no-op repeat, not an error",
			u:    &model.User{ID: userID, Timezone: ptr("Asia/Yekaterinburg"), RemindersEnabled: true},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:tz:set:yek"},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {
				users.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
				repo.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, messageID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "an unrecognized zone code is rejected",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:tz:set:utc"},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cancelling the picker just redraws the list, no writes",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:tz:cancel"},
			prepare: func(sender *Mocksender, repo *MockslotRepo, users *MockuserRepo) {
				repo.EXPECT().ListSlots(gomock.Any(), userID).Return(nil, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, messageID, gomock.Any(), gomock.Any()).Return(nil)
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
			mockUsers := NewMockuserRepo(ctrl)

			tc.prepare(mockSender, mockRepo, mockUsers)

			uc := list.New(mockSender, mockRepo, mockUsers)

			err := uc.Handle(context.Background(), tc.u, tc.in)

			tc.expected(t, err)
		})
	}
}

func ptr(s string) *string {
	return &s
}
