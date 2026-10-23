package profile_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/profile"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, messageID int64 = 777, 555

	name := "test"
	age := int16(28)

	baseUser := func() *model.User {
		return &model.User{ID: 42, Name: &name, Age: &age, Belt: model.BeltBlue}
	}

	tests := []struct {
		name     string
		u        *model.User
		in       dto.Input
		prepare  func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo)
		expected func(t assert.TestingT, u *model.User, err error)
	}{
		{
			name:    "no callback — no-op",
			u:       baseUser(),
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			u:    baseUser(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "profile:show renders the card in place",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:show",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Имя: test")
						assert.Contains(t, text, "Возраст: 28")
						assert.Contains(t, text, "🔵 Синий")

						require.Len(t, kb, 2)
						assert.Equal(t, "profile:edit", kb[0][0].Data)
						assert.Equal(t, "menu:back", kb[1][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "profile:edit shows the field-picker submenu",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:edit",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 4)
						assert.Equal(t, "profile:belt:edit", kb[0][0].Data)
						assert.Equal(t, "profile:age:edit", kb[1][0].Data)
						assert.Equal(t, "profile:tz:edit", kb[2][0].Data)
						assert.Equal(t, "profile:show", kb[3][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "profile:belt:edit shows every belt except the current one",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:edit",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 5) // 4 belts (blue excluded) + back row

						var labels []string
						for _, row := range kb[:4] {
							labels = append(labels, row[0].Label)
						}

						assert.NotContains(t, labels, "🔵 Синий")
						assert.Contains(t, labels, "⚪ Белый")

						last := kb[len(kb)-1]
						assert.Equal(t, "profile:edit", last[0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "a one-step promotion with trainings on the previous belt is recorded and congratulated",
			u:    baseUser(), // blue
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().
					ChangeBelt(gomock.Any(), int64(42), model.BeltPurple, gomock.Any(), true).
					Return(true, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "🟣 Пурпурный")

						return nil
					})

				blue := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

				user.EXPECT().
					ListBeltPromotions(gomock.Any(), int64(42)).
					Return([]*model.BeltPromotion{{Belt: model.BeltBlue, PromotedAt: blue}}, nil)

				trainings.EXPECT().
					ListTrainingsBetween(gomock.Any(), int64(42), blue, gomock.Any()).
					Return([]*model.Training{
						{DurationMinutes: 60}, {DurationMinutes: 30},
					}, nil)

				sender.EXPECT().
					Send(gomock.Any(), chatID, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, text string) error {
						assert.Contains(t, text, "🎉 Поздравляю с пурпурным поясом!")
						assert.Contains(t, text, "На синем поясе в JitLog: 2 тренировки, 1 ч 30 мин.")

						return nil
					})
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.BeltPurple, u.Belt)
			},
		},

		{
			name: "a one-step promotion with no trainings on the previous belt shows only the first line",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().ChangeBelt(gomock.Any(), int64(42), model.BeltPurple, gomock.Any(), true).Return(true, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)

				blue := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				user.EXPECT().
					ListBeltPromotions(gomock.Any(), int64(42)).
					Return([]*model.BeltPromotion{{Belt: model.BeltBlue, PromotedAt: blue}}, nil)
				trainings.EXPECT().ListTrainingsBetween(gomock.Any(), int64(42), blue, gomock.Any()).Return(nil, nil)

				sender.EXPECT().
					Send(gomock.Any(), chatID, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, text string) error {
						assert.Equal(t, "🎉 Поздравляю с пурпурным поясом!", text)

						return nil
					})
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "the previous belt has no recorded promotion — defensively, only the first line",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().ChangeBelt(gomock.Any(), int64(42), model.BeltPurple, gomock.Any(), true).Return(true, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)

				user.EXPECT().ListBeltPromotions(gomock.Any(), int64(42)).Return(nil, nil)
				// No ListTrainingsBetween call — there's no boundary to query with.

				sender.EXPECT().
					Send(gomock.Any(), chatID, "🎉 Поздравляю с пурпурным поясом!").
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "a repeat promotion to an already-recorded belt sends no congratulation",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().ChangeBelt(gomock.Any(), int64(42), model.BeltPurple, gomock.Any(), true).Return(false, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)
				// recorded=false — no ListBeltPromotions/ListTrainingsBetween/Send at all.
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "a skip-step promotion (white -> black) is recorded but not congratulated",
			u: func() *model.User {
				u := baseUser()
				u.Belt = model.BeltWhite

				return u
			}(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:black",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().ChangeBelt(gomock.Any(), int64(42), model.BeltBlack, gomock.Any(), true).Return(true, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)
				// 4-step jump, recorded=true, but no congratulation — no further calls.
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.BeltBlack, u.Belt)
			},
		},

		{
			name: "a downgrade never attempts to record a promotion, no congratulation",
			u:    baseUser(), // blue
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:white",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().ChangeBelt(gomock.Any(), int64(42), model.BeltWhite, gomock.Any(), false).Return(false, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.BeltWhite, u.Belt)
			},
		},

		{
			name: "coming back to the same belt never attempts to record a promotion",
			u:    baseUser(), // blue
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:blue",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().ChangeBelt(gomock.Any(), int64(42), model.BeltBlue, gomock.Any(), false).Return(false, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "a brown -> black promotion congratulates with the correct belt wording",
			u: func() *model.User {
				u := baseUser()
				u.Belt = model.BeltBrown

				return u
			}(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:black",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().ChangeBelt(gomock.Any(), int64(42), model.BeltBlack, gomock.Any(), true).Return(true, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)
				user.EXPECT().ListBeltPromotions(gomock.Any(), int64(42)).Return(nil, nil)

				sender.EXPECT().
					Send(gomock.Any(), chatID, "🎉 Поздравляю с чёрным поясом!").
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "a congratulation send failure does not roll back the belt change",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().ChangeBelt(gomock.Any(), int64(42), model.BeltPurple, gomock.Any(), true).Return(true, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)
				user.EXPECT().ListBeltPromotions(gomock.Any(), int64(42)).Return(nil, nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(errors.New("telegram is down"))
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.BeltPurple, u.Belt)
			},
		},

		{
			name: "failing to list belt promotions for the congratulation is logged, not propagated",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().ChangeBelt(gomock.Any(), int64(42), model.BeltPurple, gomock.Any(), true).Return(true, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)
				user.EXPECT().ListBeltPromotions(gomock.Any(), int64(42)).Return(nil, errors.New("fail"))
				// No Send at all — the text could never be built.
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failing to list the previous belt's trainings for the congratulation is logged, not propagated",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().ChangeBelt(gomock.Any(), int64(42), model.BeltPurple, gomock.Any(), true).Return(true, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)

				blue := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				user.EXPECT().
					ListBeltPromotions(gomock.Any(), int64(42)).
					Return([]*model.BeltPromotion{{Belt: model.BeltBlue, PromotedAt: blue}}, nil)
				trainings.EXPECT().
					ListTrainingsBetween(gomock.Any(), int64(42), blue, gomock.Any()).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed belt token is just acknowledged",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "profile:belt:set:bogus",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to change belt",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().
					ChangeBelt(gomock.Any(), int64(42), model.BeltPurple, gomock.Any(), true).
					Return(false, errors.New("fail"))
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "profile:tz:edit opens the zone picker",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:tz:edit",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						assert.Equal(t, "profile:tz:set:msk", kb[0][0].Data)
						assert.Equal(t, "profile:edit", kb[3][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a zone sets timezone without touching reminders_enabled",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:tz:set:sam",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, u *model.User) error {
						require.NotNil(t, u.Timezone)
						assert.Equal(t, "Europe/Samara", *u.Timezone)

						return nil
					})

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "Часовой пояс: Самара, Баку, Ереван +4")

						return nil
					})
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.False(t, u.RemindersEnabled)
			},
		},

		{
			name: "an unrecognized zone code is rejected",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "profile:tz:set:utc",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "re-tapping an already-picked zone is a no-op repeat, not an error",
			u: func() *model.User {
				u := baseUser()
				tz := "Europe/Samara"
				u.Timezone = &tz

				return u
			}(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:tz:set:sam",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "profile:age:edit sets the draft and asks for the age",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "profile:age:edit",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				drafts.EXPECT().
					SetEditDraft(gomock.Any(), int64(42)).
					Return(nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ string, kb dto.Keyboard) error {
						assert.Equal(t, "profile:age:cancel", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "profile:age:cancel clears the draft and shows the edit menu",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:age:cancel",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				drafts.EXPECT().DeleteEditDraft(gomock.Any(), int64(42)).Return(nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "✏️ Что изменить?", gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to delete draft on age cancel",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "profile:age:cancel",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				drafts.EXPECT().DeleteEditDraft(gomock.Any(), int64(42)).Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "failed to set age draft",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "profile:age:edit",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				drafts.EXPECT().
					SetEditDraft(gomock.Any(), int64(42)).
					Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockSender := NewMocksender(ctrl)
			mockUser := NewMockuser(ctrl)
			mockDrafts := NewMockprofileEditDraft(ctrl)
			mockTrainings := NewMocktrainingRepo(ctrl)

			tc.prepare(mockSender, mockUser, mockDrafts, mockTrainings)

			uc := profile.New(mockSender, mockUser, mockDrafts, mockTrainings, discardLogger())

			err := uc.Handle(context.Background(), tc.u, tc.in)

			tc.expected(t, tc.u, err)
		})
	}
}

func TestUseCase_Continue(t *testing.T) {
	t.Parallel()

	const chatID int64 = 777

	name := "test"
	age := int16(28)

	baseUser := func() *model.User {
		return &model.User{ID: 42, Name: &name, Age: &age, Belt: model.BeltBlue}
	}

	draft := &model.ProfileEditDraft{UserID: 42}

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo)
		expected func(t assert.TestingT, u *model.User, err error)
	}{
		{
			name:    "no message — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unparsable age is re-asked, draft kept",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "twenty five"},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				sender.EXPECT().
					Send(gomock.Any(), chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int16(28), *u.Age)
			},
		},

		{
			name: "out-of-range age is re-asked, draft kept",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "999"},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				sender.EXPECT().
					Send(gomock.Any(), chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int16(28), *u.Age)
			},
		},

		{
			name: "valid age is stored and the draft cleared",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "31"},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				drafts.EXPECT().
					DeleteEditDraft(gomock.Any(), int64(42)).
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "Возраст: 31")

						return nil
					})
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int16(31), *u.Age)
			},
		},

		{
			name: "failed to persist age",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "31"},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "failed to delete draft",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "31"},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft, trainings *MocktrainingRepo) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				drafts.EXPECT().
					DeleteEditDraft(gomock.Any(), int64(42)).
					Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockSender := NewMocksender(ctrl)
			mockUser := NewMockuser(ctrl)
			mockDrafts := NewMockprofileEditDraft(ctrl)
			mockTrainings := NewMocktrainingRepo(ctrl)

			tc.prepare(mockSender, mockUser, mockDrafts, mockTrainings)

			uc := profile.New(mockSender, mockUser, mockDrafts, mockTrainings, discardLogger())

			u := baseUser()
			err := uc.Continue(context.Background(), u, draft, tc.in)

			tc.expected(t, u, err)
		})
	}
}
