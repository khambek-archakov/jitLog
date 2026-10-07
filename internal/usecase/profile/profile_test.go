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
		prepare  func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft)
		expected func(t assert.TestingT, u *model.User, err error)
	}{
		{
			name:    "no callback — no-op",
			u:       baseUser(),
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			u:    baseUser(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 3)
						assert.Equal(t, "profile:belt:edit", kb[0][0].Data)
						assert.Equal(t, "profile:age:edit", kb[1][0].Data)
						assert.Equal(t, "profile:show", kb[2][0].Data)

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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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
			name: "picking a belt updates the user and records the promotion",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "profile:belt:set:purple",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				user.EXPECT().
					AddBeltPromotion(gomock.Any(), int64(42), model.BeltPurple, gomock.Any()).
					Return(nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				user.EXPECT().
					AddBeltPromotion(gomock.Any(), int64(42), model.BeltPurple, gomock.Any()).
					Return(model.ErrDuplicateBeltPromotion)

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
			name: "malformed belt token is just acknowledged",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "profile:belt:set:bogus",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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

		{
			name: "profile:age:edit sets the draft and asks for the age",
			u:    baseUser(),
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "profile:age:edit",
			},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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

			tc.prepare(mockSender, mockUser, mockDrafts)

			uc := profile.New(mockSender, mockUser, mockDrafts)

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
		prepare  func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft)
		expected func(t assert.TestingT, u *model.User, err error)
	}{
		{
			name:    "no message — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unparsable age is re-asked, draft kept",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "twenty five"},
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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
			prepare: func(sender *Mocksender, user *Mockuser, drafts *MockprofileEditDraft) {
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

			tc.prepare(mockSender, mockUser, mockDrafts)

			uc := profile.New(mockSender, mockUser, mockDrafts)

			u := baseUser()
			err := uc.Continue(context.Background(), u, draft, tc.in)

			tc.expected(t, u, err)
		})
	}
}
