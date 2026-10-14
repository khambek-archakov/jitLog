package ajp

import (
	"context"
	"fmt"
	"os"
)

// QuietFailureSender wraps bot so the one monitoring-failure message (see
// Run/monitoringFailure — the only place in this package that calls the
// bare Send) prints to stderr instead of reaching Telegram, while every
// other message (new competitions, diffs) still goes out normally through
// bot's own SendWithKeyboard. cmd/cron/competition_import/ajp uses this
// for --from-file: a human is already watching the terminal and would
// otherwise see the exact same failure twice.
func QuietFailureSender(bot sender) sender {
	return quietFailureSender{bot}
}

type quietFailureSender struct {
	sender
}

func (quietFailureSender) Send(_ context.Context, _ int64, text string) error {
	fmt.Fprintln(os.Stderr, text)

	return nil
}
