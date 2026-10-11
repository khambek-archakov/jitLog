package parse_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/usecase/competition/import/ajp/parse"
)

func loadTestdata(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)

	return string(data)
}

func TestParseEvents(t *testing.T) {
	t.Parallel()

	t.Run("saved AJP page", func(t *testing.T) {
		t.Parallel()

		events, err := parse.ParseEvents(loadTestdata(t, "ajp_events.html"))
		require.NoError(t, err)
		require.Len(t, events, 54)

		first := events[0]
		assert.Equal(t, int64(1558), first.ID)
		assert.Equal(t, "AJP TOUR SWITZERLAND NATIONAL JIU-JITSU CHAMPIONSHIP 2026 - GI & NO-GI", first.Name)
		assert.Equal(t, "October 10", first.DateRaw)
		assert.Equal(t, "https://ajptour.com/en/event/1558", first.URL)
		assert.Equal(t, "CH", first.CountryCode)
		require.NotNil(t, first.City)
		assert.Equal(t, "Bern", *first.City)

		// Card 5 — a multi-day event, and p.muted has a stray space before
		// its comma ("Committee , Ulaanbaatar, Mongolia") that still needs
		// to parse cleanly.
		multiDay := events[4]
		assert.Equal(t, int64(1491), multiDay.ID)
		assert.Equal(t, "October 10 - 11", multiDay.DateRaw)
		require.NotNil(t, multiDay.City)
		assert.Equal(t, "Ulaanbaatar", *multiDay.City)

		last := events[len(events)-1]
		assert.Equal(t, int64(1578), last.ID)
		assert.Equal(t, "2027 September 11", last.DateRaw)
		assert.Equal(t, "AE", last.CountryCode)
	})

	t.Run("no eventItem cards at all", func(t *testing.T) {
		t.Parallel()

		_, err := parse.ParseEvents(`<html><body><section id="upcoming-events"></section></body></html>`)
		assert.ErrorIs(t, err, parse.ErrNoEvents)
	})

	t.Run("no upcoming-events section at all", func(t *testing.T) {
		t.Parallel()

		_, err := parse.ParseEvents(`<html><body><p>nothing here</p></body></html>`)
		assert.ErrorIs(t, err, parse.ErrNoEvents)
	})

	t.Run("a card missing data-id fails the whole parse", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><section id="upcoming-events">
			<div class="event-bg cover-bg eventItem" data-name="X" data-date="October 10"
			     data-url="https://ajptour.com/en/event/1" data-country="RU"></div>
		</section></body></html>`

		_, err := parse.ParseEvents(html)
		assert.ErrorIs(t, err, parse.ErrMissingAttributes)
	})

	t.Run("a card missing data-url fails the whole parse", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><section id="upcoming-events">
			<div class="event-bg cover-bg eventItem" data-id="1" data-name="X" data-date="October 10"
			     data-country="RU"></div>
		</section></body></html>`

		_, err := parse.ParseEvents(html)
		assert.ErrorIs(t, err, parse.ErrMissingAttributes)
	})

	t.Run("a card missing data-date fails the whole parse", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><section id="upcoming-events">
			<div class="event-bg cover-bg eventItem" data-id="1" data-name="X"
			     data-url="https://ajptour.com/en/event/1" data-country="RU"></div>
		</section></body></html>`

		_, err := parse.ParseEvents(html)
		assert.ErrorIs(t, err, parse.ErrMissingAttributes)
	})

	t.Run("p.muted with only city and country — city is the first part", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><section id="upcoming-events">
			<div class="event-bg cover-bg eventItem" data-id="1" data-name="X" data-date="October 10"
			     data-url="https://ajptour.com/en/event/1" data-country="RU">
				<p class="muted">Moscow, Russia</p>
			</div>
		</section></body></html>`

		events, err := parse.ParseEvents(html)
		require.NoError(t, err)
		require.Len(t, events, 1)
		require.NotNil(t, events[0].City)
		assert.Equal(t, "Moscow", *events[0].City)
	})

	t.Run("p.muted with a single part — city is nil, not a parse failure", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><section id="upcoming-events">
			<div class="event-bg cover-bg eventItem" data-id="1" data-name="X" data-date="October 10"
			     data-url="https://ajptour.com/en/event/1" data-country="RU">
				<p class="muted">Somewhere</p>
			</div>
		</section></body></html>`

		events, err := parse.ParseEvents(html)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Nil(t, events[0].City)
	})

	t.Run("no p.muted at all — city is nil", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><section id="upcoming-events">
			<div class="event-bg cover-bg eventItem" data-id="1" data-name="X" data-date="October 10"
			     data-url="https://ajptour.com/en/event/1" data-country="RU"></div>
		</section></body></html>`

		events, err := parse.ParseEvents(html)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Nil(t, events[0].City)
	})

	t.Run("data-name is unescaped back to plain text", func(t *testing.T) {
		t.Parallel()

		html := `<html><body><section id="upcoming-events">
			<div class="event-bg cover-bg eventItem" data-id="1" data-name="GI &amp; NO-GI" data-date="October 10"
			     data-url="https://ajptour.com/en/event/1" data-country="RU"></div>
		</section></body></html>`

		events, err := parse.ParseEvents(html)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, "GI & NO-GI", events[0].Name)
	})
}
