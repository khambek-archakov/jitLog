package start_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

func TestUseCase_Start(t *testing.T) {
	t.Parallel()

	const telegramID, chatID int64 = 123, 777

	name := "test"

	tests := []struct {
		name    string
		in      dto.Input
		prepare func(
			user *Mockuser,
			sender *Mocksender,
		)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "failed to fetch user",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "no user, not a /start command — ignored",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, HasMessage: true, Text: "hi"},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(nil, model.ErrNotFound)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to create user",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(nil, model.ErrNotFound)

				user.EXPECT().
					Create(gomock.Any(), telegramID).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "new user is greeted on /start",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(nil, model.ErrNotFound)

				user.EXPECT().
					Create(gomock.Any(), telegramID).
					Return(&model.User{
						ID:             1,
						TelegramID:     telegramID,
						OnboardingStep: model.OnboardingStepAwaitingName,
					}, nil)

				sender.EXPECT().
					Send(chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "user mid-onboarding is re-asked their pending question on /start",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{
						ID:             2,
						TelegramID:     telegramID,
						Name:           &name,
						OnboardingStep: model.OnboardingStepAwaitingAge,
					}, nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown onboarding step — no handler, no-op",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{
						ID:             3,
						TelegramID:     telegramID,
						OnboardingStep: model.OnboardingStep("bogus"),
					}, nil)
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

			mockUser := NewMockuser(ctrl)
			mockSender := NewMocksender(ctrl)

			tc.prepare(mockUser, mockSender)

			uc := start.New(mockSender, mockUser)

			err := uc.Start(context.Background(), tc.in)

			tc.expected(t, err)
		})
	}
}
