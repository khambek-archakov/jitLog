package router_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/router"
)

func TestRouter_Route(t *testing.T) {
	t.Parallel()

	const telegramID, userID, chatID int64 = 123, 42, 777

	tests := []struct {
		name    string
		in      dto.Input
		prepare func(
			onboarding *Mockonboarding,
			training *Mocktraining,
			user *Mockuser,
			drafts *MocktrainingDraft,
		)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "failed to fetch user",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(onboarding *Mockonboarding, training *Mocktraining, user *Mockuser, drafts *MocktrainingDraft) {
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
			prepare: func(onboarding *Mockonboarding, training *Mocktraining, user *Mockuser, drafts *MocktrainingDraft) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(nil, model.ErrNotFound)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "onboarding not complete goes to onboarding",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(onboarding *Mockonboarding, training *Mocktraining, user *Mockuser, drafts *MocktrainingDraft) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepAwaitingAge}, nil)

				onboarding.EXPECT().
					Handle(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "active training draft goes to training",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, HasMessage: true, Text: "60"},
			prepare: func(onboarding *Mockonboarding, training *Mocktraining, user *Mockuser, drafts *MocktrainingDraft) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				drafts.EXPECT().
					GetDraftByUserID(gomock.Any(), userID).
					Return(&model.TrainingDraft{ID: 1, UserID: userID}, nil)

				training.EXPECT().
					Continue(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch training draft",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, HasMessage: true, Text: "hi"},
			prepare: func(onboarding *Mockonboarding, training *Mocktraining, user *Mockuser, drafts *MocktrainingDraft) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				drafts.EXPECT().
					GetDraftByUserID(gomock.Any(), userID).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "add-training button with no draft begins training",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "menu:add_training"},
			prepare: func(onboarding *Mockonboarding, training *Mocktraining, user *Mockuser, drafts *MocktrainingDraft) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				drafts.EXPECT().
					GetDraftByUserID(gomock.Any(), userID).
					Return(nil, model.ErrNotFound)

				training.EXPECT().
					Begin(gomock.Any(), userID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "no draft, no add-training trigger falls through to onboarding's menu handling",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, HasMessage: true, Text: "hi"},
			prepare: func(onboarding *Mockonboarding, training *Mocktraining, user *Mockuser, drafts *MocktrainingDraft) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				drafts.EXPECT().
					GetDraftByUserID(gomock.Any(), userID).
					Return(nil, model.ErrNotFound)

				onboarding.EXPECT().
					Handle(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
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

			mockOnboarding := NewMockonboarding(ctrl)
			mockTraining := NewMocktraining(ctrl)
			mockUser := NewMockuser(ctrl)
			mockDrafts := NewMocktrainingDraft(ctrl)

			tc.prepare(mockOnboarding, mockTraining, mockUser, mockDrafts)

			r := router.New(mockOnboarding, mockTraining, mockUser, mockDrafts)

			err := r.Route(context.Background(), tc.in)

			tc.expected(t, err)
		})
	}
}
