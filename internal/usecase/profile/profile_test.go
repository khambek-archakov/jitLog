package profile_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/profile"
)

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
		prepare  func(sender *Mocksender, user *Mockuser)
		expected func(t assert.TestingT, u *model.User, err error)
	}{
		{
			name:    "no callback — no-op",
			u:       baseUser(),
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, user *Mockuser) {},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			u:    baseUser(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, user *Mockuser) {
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
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
			prepare: func(sender *Mocksender, user *Mockuser) {
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Имя: test")
						assert.Contains(t, text, "Возраст: 28")
						assert.Contains(t, text, "🔵 Синий")

						require.Len(t, kb, 2)
						assert.Equal(t, "profile:belt:edit", kb[0][0].Data)
						assert.Equal(t, "menu:back", kb[1][0].Data)

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
			prepare: func(sender *Mocksender, user *Mockuser) {
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 5) // 4 belts (blue excluded) + back row

						var labels []string
						for _, row := range kb[:4] {
							labels = append(labels, row[0].Label)
						}

						assert.NotContains(t, labels, "🔵 Синий")
						assert.Contains(t, labels, "⚪ Белый")

						last := kb[len(kb)-1]
						assert.Equal(t, "profile:show", last[0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a belt updates the user and records the promotion",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				user.EXPECT().
					AddBeltPromotion(gomock.Any(), int64(42), model.BeltPurple, gomock.Any()).
					Return(nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "🟣 Пурпурный")

						return nil
					})
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.BeltPurple, u.Belt)
			},
		},

		{
			name: "belt already recorded is not an error",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				user.EXPECT().
					AddBeltPromotion(gomock.Any(), int64(42), model.BeltPurple, gomock.Any()).
					Return(model.ErrDuplicateBeltPromotion)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), gomock.Any(), gomock.Any()).
					Return(nil)
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
			prepare: func(sender *Mocksender, user *Mockuser) {
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to persist belt",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "failed to add belt promotion",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				user.EXPECT().
					AddBeltPromotion(gomock.Any(), int64(42), model.BeltPurple, gomock.Any()).
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

			tc.prepare(mockSender, mockUser)

			uc := profile.New(mockSender, mockUser)

			err := uc.Handle(context.Background(), tc.u, tc.in)

			tc.expected(t, tc.u, err)
		})
	}
}
