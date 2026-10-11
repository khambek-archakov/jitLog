// Package parse owns turning the AJP federation events page's HTML into a
// list of structured events. ParseEvents is a pure function — no network,
// no database, no clock — so it's fully testable against a saved copy of
// the page (see testdata/ajp_events.html).
package parse

import (
	"errors"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Event is one div.eventItem's data-* attributes, plus a City derived
// from its sibling p.muted markup. The venue (p.muted's first part, when
// there are three) is deliberately not kept here at all — the import
// never saves it.
type Event struct {
	ID          int64
	Name        string
	DateRaw     string
	URL         string
	CountryCode string
	// City is nil when p.muted couldn't be split into at least two
	// comma-separated parts — the caller reports this, it never guesses
	// a city out of a single unstructured string.
	City *string
}

// ErrNoEvents means the page loaded but not a single div.eventItem turned
// up inside section#upcoming-events — AJP's own calendar is never
// actually empty, so this means the page's structure changed, not "zero
// events today".
var ErrNoEvents = errors.New("no eventItem cards found in section#upcoming-events")

// ErrMissingAttributes means at least one eventItem card was missing (or
// had an empty/unparseable) data-id, data-date or data-url — the one
// shape this importer refuses to guess around, since dedup and
// scheduling both depend on them.
var ErrMissingAttributes = errors.New("eventItem card missing a required data attribute")

// ParseEvents extracts every div.eventItem inside section#upcoming-events.
func ParseEvents(pageHTML string) ([]Event, error) {
	doc, err := html.Parse(strings.NewReader(pageHTML))
	if err != nil {
		return nil, err
	}

	section := findFirst(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "section" && attrEquals(n, "id", "upcoming-events")
	})
	if section == nil {
		return nil, ErrNoEvents
	}

	cards := findAll(section, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "eventItem")
	})
	if len(cards) == 0 {
		return nil, ErrNoEvents
	}

	events := make([]Event, 0, len(cards))

	for _, card := range cards {
		idRaw, hasID := attr(card, "data-id")
		dateRaw, hasDate := attr(card, "data-date")
		url, hasURL := attr(card, "data-url")

		if !hasID || idRaw == "" || !hasDate || dateRaw == "" || !hasURL || url == "" {
			return nil, ErrMissingAttributes
		}

		id, err := strconv.ParseInt(idRaw, 10, 64)
		if err != nil {
			return nil, ErrMissingAttributes
		}

		name, _ := attr(card, "data-name")
		country, _ := attr(card, "data-country")

		events = append(events, Event{
			ID:          id,
			Name:        name,
			DateRaw:     dateRaw,
			URL:         url,
			CountryCode: country,
			City:        cityFromMuted(card),
		})
	}

	return events, nil
}

// cityFromMuted splits p.muted's text by comma and trims whitespace off
// each part (the site itself sometimes leaves a stray space before the
// comma, e.g. "Committee ,"). The last part is always the country (read
// from data-country instead — p.muted's own country text has had typos
// on the live site), the one before it is the city; "Venue" is simply
// absent when there are only two parts. One part alone leaves City nil.
func cityFromMuted(card *html.Node) *string {
	p := findFirst(card, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "p" && hasClass(n, "muted")
	})
	if p == nil {
		return nil
	}

	parts := strings.Split(textContent(p), ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(parts) < 2 {
		return nil
	}

	city := parts[len(parts)-2]
	if city == "" {
		return nil
	}

	return &city
}

func findFirst(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirst(c, match); found != nil {
			return found
		}
	}

	return nil
}

func findAll(n *html.Node, match func(*html.Node) bool) []*html.Node {
	var found []*html.Node

	if match(n) {
		found = append(found, n)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		found = append(found, findAll(c, match)...)
	}

	return found
}

func attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}

	return "", false
}

func attrEquals(n *html.Node, key, val string) bool {
	v, ok := attr(n, key)
	return ok && v == val
}

func hasClass(n *html.Node, class string) bool {
	v, ok := attr(n, "class")
	if !ok {
		return false
	}

	for _, c := range strings.Fields(v) {
		if c == class {
			return true
		}
	}

	return false
}

func textContent(n *html.Node) string {
	var b strings.Builder

	var walk func(*html.Node)

	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)

	return b.String()
}
