package moderate_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/moderate"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	adminTelegramID int64 = 1001
	chatID          int64 = 777
	messageID       int64 = 555
)

func adminUser() *model.User {
	return &model.User{ID: 1, TelegramID: adminTelegramID}
}

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		u        *model.User
		in       dto.Input
		prepare  func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no callback — no-op",
			u:       adminUser(),
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "non-admin gets no access",
			u:    &model.User{ID: 2, TelegramID: 2222},
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:moderate:55:approve",
			},
			prepare: func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo) {
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Нет доступа.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed id is just acknowledged",
			u:    adminUser(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:moderate:bogus"},
			prepare: func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "approve notifies every owner and marks the message done",
			u:    adminUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "competition:moderate:55:approve",
			},
			prepare: func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo) {
				repo.EXPECT().SetCompetitionStatus(gomock.Any(), int64(55), model.CompetitionStatusPublished).Return(true, nil)
				repo.EXPECT().ListOwnersByCompetitionID(gomock.Any(), int64(55)).Return([]int64{10, 11}, nil)

				users.EXPECT().GetByID(gomock.Any(), int64(10)).Return(&model.User{ID: 10, TelegramID: 100}, nil)
				users.EXPECT().GetByID(gomock.Any(), int64(11)).Return(&model.User{ID: 11, TelegramID: 110}, nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), int64(100), "✅ Турнир опубликован в каталоге!", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ string, kb dto.Keyboard) error {
						assert.Equal(t, "competition:list", kb[0][0].Data)

						return nil
					})
				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), int64(110), "✅ Турнир опубликован в каталоге!", gomock.Any()).
					Return(nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "✅ Одобрено.", gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "approving an already-moderated entry is a silent no-op",
			u:    adminUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:moderate:55:approve",
			},
			prepare: func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo) {
				repo.EXPECT().SetCompetitionStatus(gomock.Any(), int64(55), model.CompetitionStatusPublished).Return(false, nil)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Уже обработано.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "reject notifies owners too",
			u:    adminUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "competition:moderate:55:reject",
			},
			prepare: func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo) {
				repo.EXPECT().SetCompetitionStatus(gomock.Any(), int64(55), model.CompetitionStatusRejected).Return(true, nil)
				repo.EXPECT().ListOwnersByCompetitionID(gomock.Any(), int64(55)).Return([]int64{10}, nil)
				users.EXPECT().GetByID(gomock.Any(), int64(10)).Return(&model.User{ID: 10, TelegramID: 100}, nil)
				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), int64(100), "❌ Турнир отклонён модератором.", gomock.Any()).
					Return(nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "❌ Отклонено.", gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "dup sets the merge draft and prompts for a search term",
			u:    adminUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:moderate:55:dup",
			},
			prepare: func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo) {
				repo.EXPECT().SetMergeDraft(gomock.Any(), int64(1), int64(55)).Return(nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "merge repoints owners and deletes the draft",
			u:    adminUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "competition:moderate:55:merge:60",
			},
			prepare: func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo) {
				repo.EXPECT().ListOwnersByCompetitionID(gomock.Any(), int64(55)).Return([]int64{10}, nil)
				repo.EXPECT().MergeCompetition(gomock.Any(), int64(55), int64(60)).Return(true, nil)
				repo.EXPECT().DeleteMergeDraft(gomock.Any(), int64(1)).Return(nil)
				users.EXPECT().GetByID(gomock.Any(), int64(10)).Return(&model.User{ID: 10, TelegramID: 100}, nil)
				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), int64(100), "Этот турнир уже есть в каталоге, я добавил его к существующему.", gomock.Any()).
					Return(nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "🔗 Объединено с существующим турниром.", gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "merging an already-merged pending entry is a silent no-op",
			u:    adminUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:moderate:55:merge:60",
			},
			prepare: func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo) {
				repo.EXPECT().ListOwnersByCompetitionID(gomock.Any(), int64(55)).Return(nil, nil)
				repo.EXPECT().MergeCompetition(gomock.Any(), int64(55), int64(60)).Return(false, nil)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Уже обработано.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed merge target is just acknowledged",
			u:    adminUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:moderate:55:merge:bogus",
			},
			prepare: func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to set status",
			u:    adminUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:moderate:55:approve",
			},
			prepare: func(sender *Mocksender, repo *Mockrepo, users *MockuserRepo) {
				repo.EXPECT().SetCompetitionStatus(gomock.Any(), int64(55), model.CompetitionStatusPublished).
					Return(false, errors.New("fail"))
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
			mockRepo := NewMockrepo(ctrl)
			mockUsers := NewMockuserRepo(ctrl)

			tc.prepare(mockSender, mockRepo, mockUsers)

			uc := moderate.New(mockSender, mockRepo, mockUsers, adminTelegramID)

			err := uc.Handle(context.Background(), tc.u, tc.in)

			tc.expected(t, err)
		})
	}
}

func TestUseCase_Continue(t *testing.T) {
	t.Parallel()

	u := adminUser()
	draft := &model.CompetitionMergeDraft{UserID: 1, PendingCompetitionID: 55}

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *Mockrepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no message — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *Mockrepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "blank term is re-asked",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "   "},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "no results — draft stays active, admin can retry",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "nonexistent"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				repo.EXPECT().SearchCompetitionsByTitle(gomock.Any(), "nonexistent", int64(55), 10).Return(nil, nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "results shown as a picker",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "Moscow"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				results := []*model.Competition{
					{ID: 60, Title: "Moscow Open", Date: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)},
				}
				repo.EXPECT().SearchCompetitionsByTitle(gomock.Any(), "Moscow", int64(55), 10).Return(results, nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, "Это один из них?", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 1)
						assert.Equal(t, "competition:moderate:55:merge:60", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to search",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "Moscow"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				repo.EXPECT().SearchCompetitionsByTitle(gomock.Any(), "Moscow", int64(55), 10).Return(nil, errors.New("fail"))
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
			mockRepo := NewMockrepo(ctrl)
			mockUsers := NewMockuserRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := moderate.New(mockSender, mockRepo, mockUsers, adminTelegramID)

			err := uc.Continue(context.Background(), u, draft, tc.in)

			tc.expected(t, err)
		})
	}
}
