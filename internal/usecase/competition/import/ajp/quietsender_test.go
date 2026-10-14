package ajp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/usecase/competition/import/ajp"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestQuietFailureSender(t *testing.T) {
	t.Run("Send prints to stderr instead of calling the wrapped bot", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		bot := NewMocksender(ctrl)
		// No bot.EXPECT().Send(...) at all: the wrapped bot's own Send
		// must never be reached.

		quiet := ajp.QuietFailureSender(bot)

		stderr := captureStderr(t, func() {
			err := quiet.Send(context.Background(), 1001, "Импорт AJP не работает: у страницы статус 403")
			require.NoError(t, err)
		})

		assert.Contains(t, stderr, "Импорт AJP не работает")
	})

	t.Run("SendWithKeyboard still passes straight through to the wrapped bot", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		bot := NewMocksender(ctrl)

		bot.EXPECT().SendWithKeyboard(gomock.Any(), int64(1001), "AJP: 1 новых соревнований на проверке", gomock.Any()).
			Return(nil)

		quiet := ajp.QuietFailureSender(bot)

		err := quiet.SendWithKeyboard(context.Background(), 1001, "AJP: 1 новых соревнований на проверке", dto.Keyboard{})
		require.NoError(t, err)
	})
}
