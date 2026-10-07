package create_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/create"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	callbackCompetitionAdd            = "competition:add"
	callbackCompetitionAddFromCatalog = "competition:add:from_catalog"
)

func TestUseCase_Begin(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	u := &model.User{ID: userID}

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockdraftRepo, catalogList *MockcatalogList)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "failed to create draft",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackCompetitionAdd},
			prepare: func(sender *Mocksender, repo *MockdraftRepo, catalogList *MockcatalogList) {
				repo.EXPECT().CreateDraft(gomock.Any(), userID, false).Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "creates the draft and shows the title question",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackCompetitionAdd},
			prepare: func(sender *Mocksender, repo *MockdraftRepo, catalogList *MockcatalogList) {
				repo.EXPECT().
					CreateDraft(gomock.Any(), userID, false).
					Return(&model.UserCompetitionDraft{ID: 1, UserID: userID, Step: model.UserCompetitionDraftStepAwaitingTitle}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "started from the catalog screen marks the draft accordingly",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackCompetitionAddFromCatalog,
			},
			prepare: func(sender *Mocksender, repo *MockdraftRepo, catalogList *MockcatalogList) {
				repo.EXPECT().
					CreateDraft(gomock.Any(), userID, true).
					Return(&model.UserCompetitionDraft{ID: 1, UserID: userID, Step: model.UserCompetitionDraftStepAwaitingTitle, FromCatalog: true}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
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
			mockRepo := NewMockdraftRepo(ctrl)
			mockCatalogList := NewMockcatalogList(ctrl)

			tc.prepare(mockSender, mockRepo, mockCatalogList)

			uc := create.New(mockSender, mockRepo, mockCatalogList)

			err := uc.Begin(context.Background(), u, tc.in)

			tc.expected(t, err)
		})
	}
}

func TestUseCase_Continue(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	u := &model.User{ID: userID}
	title := "Moscow Open"

	tests := []struct {
		name     string
		d        *model.UserCompetitionDraft
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockdraftRepo, catalogList *MockcatalogList)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "dispatches to the step matching the draft's current step",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Step: model.UserCompetitionDraftStepAwaitingTitle},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "Moscow Open"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo, catalogList *MockcatalogList) {
				repo.EXPECT().UpdateDraft(gomock.Any(), gomock.Any()).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "date step finishes the wizard end to end",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Title: &title, Step: model.UserCompetitionDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "20.11.2026"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo, catalogList *MockcatalogList) {
				want := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

				repo.EXPECT().
					CreateUserCompetition(gomock.Any(), userID, "Moscow Open", want).
					Return(&model.UserCompetition{ID: 7, Title: "Moscow Open", Date: want}, nil)

				repo.EXPECT().DeleteDraft(gomock.Any(), userID).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:    "unknown step is a no-op",
			d:       &model.UserCompetitionDraft{ID: 1, Step: model.UserCompetitionDraftStep("bogus")},
			in:      dto.Input{ChatID: chatID, HasMessage: true, Text: "x"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo, catalogList *MockcatalogList) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cancel deletes the draft and shows the main menu, regardless of step",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Step: model.UserCompetitionDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:draft:cancel"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo, catalogList *MockcatalogList) {
				repo.EXPECT().DeleteDraft(gomock.Any(), userID).Return(nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, "Вот что я умею:", gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cancel on a draft started from the catalog returns there instead",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Step: model.UserCompetitionDraftStepAwaitingDate, FromCatalog: true},
			in:   dto.Input{ChatID: chatID, MessageID: 555, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:draft:cancel"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo, catalogList *MockcatalogList) {
				repo.EXPECT().DeleteDraft(gomock.Any(), userID).Return(nil)

				catalogList.EXPECT().
					Handle(gomock.Any(), u, dto.Input{
						ChatID: chatID, MessageID: 555, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:list",
					}).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to delete draft on cancel",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Step: model.UserCompetitionDraftStepAwaitingTitle},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:draft:cancel"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo, catalogList *MockcatalogList) {
				repo.EXPECT().DeleteDraft(gomock.Any(), userID).Return(errors.New("fail"))
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
			mockRepo := NewMockdraftRepo(ctrl)
			mockCatalogList := NewMockcatalogList(ctrl)

			tc.prepare(mockSender, mockRepo, mockCatalogList)

			uc := create.New(mockSender, mockRepo, mockCatalogList)

			err := uc.Continue(context.Background(), u, tc.d, tc.in)

			tc.expected(t, err)
		})
	}
}
