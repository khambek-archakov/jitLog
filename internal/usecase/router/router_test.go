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

type mocks struct {
	onboarding *Mockonboarding
	create     *MocktrainingCreate
	info       *MocktrainingInfo
	history    *MocktrainingHistory
	update     *MocktrainingUpdate
	delete     *MocktrainingDelete
	user       *Mockuser
	drafts     *MocktrainingDraft
	edits      *MocktrainingEditDraft
}

// noActiveDrafts stubs both the training draft and edit draft lookups to
// "none in progress" — the shared setup every case reaching the
// callback-prefix switch needs.
func noActiveDrafts(m mocks, userID int64) {
	m.drafts.EXPECT().
		GetDraftByUserID(gomock.Any(), userID).
		Return(nil, model.ErrNotFound)

	m.edits.EXPECT().
		GetEditDraftByUserID(gomock.Any(), userID).
		Return(nil, model.ErrNotFound)
}

func TestRouter_Route(t *testing.T) {
	t.Parallel()

	const telegramID, userID, chatID int64 = 123, 42, 777

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(m mocks)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "failed to fetch user",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(m mocks) {
				m.user.EXPECT().
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
			prepare: func(m mocks) {
				m.user.EXPECT().
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
			prepare: func(m mocks) {
				m.user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepAwaitingAge}, nil)

				m.onboarding.EXPECT().
					Handle(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "active training draft goes to create",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, HasMessage: true, Text: "60"},
			prepare: func(m mocks) {
				m.user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				m.drafts.EXPECT().
					GetDraftByUserID(gomock.Any(), userID).
					Return(&model.TrainingDraft{ID: 1, UserID: userID}, nil)

				m.create.EXPECT().
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
			prepare: func(m mocks) {
				m.user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				m.drafts.EXPECT().
					GetDraftByUserID(gomock.Any(), userID).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "active edit draft goes to update",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, HasMessage: true, Text: "45"},
			prepare: func(m mocks) {
				m.user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				m.drafts.EXPECT().
					GetDraftByUserID(gomock.Any(), userID).
					Return(nil, model.ErrNotFound)

				m.edits.EXPECT().
					GetEditDraftByUserID(gomock.Any(), userID).
					Return(&model.TrainingEditDraft{UserID: userID, TrainingID: 7}, nil)

				m.update.EXPECT().
					Continue(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch training edit draft",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, HasMessage: true, Text: "hi"},
			prepare: func(m mocks) {
				m.user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				m.drafts.EXPECT().
					GetDraftByUserID(gomock.Any(), userID).
					Return(nil, model.ErrNotFound)

				m.edits.EXPECT().
					GetEditDraftByUserID(gomock.Any(), userID).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "add-training button with no draft begins create",
			in: dto.Input{
				TelegramID: telegramID, ChatID: chatID, HasCallback: true, CallbackID: "cb-1",
				CallbackData: "menu:add_training",
			},
			prepare: func(m mocks) {
				m.user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				noActiveDrafts(m, userID)

				m.create.EXPECT().
					Begin(gomock.Any(), userID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "view callback goes to info",
			in: dto.Input{
				TelegramID: telegramID, ChatID: chatID, HasCallback: true, CallbackID: "cb-1",
				CallbackData: "training:view:7",
			},
			prepare: func(m mocks) {
				m.user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				noActiveDrafts(m, userID)

				m.info.EXPECT().
					Handle(gomock.Any(), userID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "history page callback goes to history",
			in: dto.Input{
				TelegramID: telegramID, ChatID: chatID, HasCallback: true, CallbackID: "cb-1",
				CallbackData: "training:history:page:0",
			},
			prepare: func(m mocks) {
				m.user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				noActiveDrafts(m, userID)

				m.history.EXPECT().
					Handle(gomock.Any(), userID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "edit callback goes to update",
			in: dto.Input{
				TelegramID: telegramID, ChatID: chatID, HasCallback: true, CallbackID: "cb-1",
				CallbackData: "training:edit:7",
			},
			prepare: func(m mocks) {
				m.user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				noActiveDrafts(m, userID)

				m.update.EXPECT().
					Handle(gomock.Any(), userID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "delete callback goes to delete",
			in: dto.Input{
				TelegramID: telegramID, ChatID: chatID, HasCallback: true, CallbackID: "cb-1",
				CallbackData: "training:delete:7",
			},
			prepare: func(m mocks) {
				m.user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				noActiveDrafts(m, userID)

				m.delete.EXPECT().
					Handle(gomock.Any(), userID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "no draft, no known trigger falls through to onboarding's menu handling",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, HasMessage: true, Text: "hi"},
			prepare: func(m mocks) {
				m.user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}, nil)

				noActiveDrafts(m, userID)

				m.onboarding.EXPECT().
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

			m := mocks{
				onboarding: NewMockonboarding(ctrl),
				create:     NewMocktrainingCreate(ctrl),
				info:       NewMocktrainingInfo(ctrl),
				history:    NewMocktrainingHistory(ctrl),
				update:     NewMocktrainingUpdate(ctrl),
				delete:     NewMocktrainingDelete(ctrl),
				user:       NewMockuser(ctrl),
				drafts:     NewMocktrainingDraft(ctrl),
				edits:      NewMocktrainingEditDraft(ctrl),
			}

			tc.prepare(m)

			r := router.New(
				m.onboarding, m.create, m.info, m.history, m.update, m.delete, m.user, m.drafts, m.edits,
			)

			err := r.Route(context.Background(), tc.in)

			tc.expected(t, err)
		})
	}
}
