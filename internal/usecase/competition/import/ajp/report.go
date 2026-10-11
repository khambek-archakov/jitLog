package ajp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
)

// FormatReport renders a Report as plain text — used for --dry-run's own
// stdout printout and, by cmd/cron/competition_import/ajp, for every real
// run's own log line too (the one artifact and no DB write, but a
// genuinely empty AJP_COUNTRIES, plus a plain informational run, share the
// exact same shape of "what happened").
func FormatReport(r *Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Карточек на странице: %d\n", r.TotalCards)
	b.WriteString("По странам:\n")

	for _, code := range sortedCountryCodes(r.CountryCounts) {
		fmt.Fprintf(&b, "  %s: %d\n", code, r.CountryCounts[code])
	}

	if len(r.Created) > 0 {
		fmt.Fprintf(&b, "\nНовые соревнования (%d):\n", len(r.Created))

		for _, c := range r.Created {
			fmt.Fprintf(&b, "  • %s — %s\n", c.Title, dateLabel(c))
		}
	}

	if len(r.Diffs) > 0 {
		fmt.Fprintf(&b, "\nРасхождения (%d):\n", len(r.Diffs))

		for _, d := range r.Diffs {
			oldCity := "—"
			if d.Competition.City != nil && *d.Competition.City != "" {
				oldCity = *d.Competition.City
			}

			newDate := info.FormatDateRangeCompact(d.ProposedDate, endOrDate(d.ProposedDate, d.ProposedEndDate))

			fmt.Fprintf(
				&b, "  • %s — было %s, %s; стало %s, %s\n",
				d.Competition.Title, dateLabel(d.Competition), oldCity, newDate, d.ProposedCity,
			)
		}
	}

	if len(r.Skipped) > 0 {
		fmt.Fprintf(&b, "\nПропущено (%d):\n", len(r.Skipped))

		for _, s := range r.Skipped {
			fmt.Fprintf(&b, "  • %s — %s\n", s.Name, s.Reason)
		}
	}

	if len(r.UntranslatedCities) > 0 {
		b.WriteString("\nГород без перевода:\n")

		for _, c := range r.UntranslatedCities {
			fmt.Fprintf(&b, "  • %s\n", c)
		}
	}

	return b.String()
}

func sortedCountryCodes(counts map[string]int) []string {
	codes := make([]string, 0, len(counts))
	for code := range counts {
		codes = append(codes, code)
	}

	sort.Strings(codes)

	return codes
}
