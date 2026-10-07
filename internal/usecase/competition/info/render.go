package info

import (
	"fmt"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	callbackEditPrefix   = "competition:edit:"
	callbackDeletePrefix = "competition:delete:"
	// callbackSubmitPrefix mirrors competition/submit's own private
	// constant — "📤 Предложить в каталог" hands off there.
	callbackSubmitPrefix = "competition:submit:"
	// callbackCompetitionList mirrors internal/usecase/competition/list's
	// own private constant — the card's "← Назад" button goes back there.
	callbackCompetitionList = "competition:list"
)

// Today returns "today" as a UTC-midnight time.Time (matching how a date
// column round-trips through pgx) in u's own timezone, falling back to UTC
// when Timezone is unset or isn't a name time.LoadLocation recognizes —
// nothing sets it yet, so that fallback is the common case today.
func Today(u *model.User) time.Time {
	loc := time.UTC

	if u.Timezone != nil {
		if l, err := time.LoadLocation(*u.Timezone); err == nil {
			loc = l
		}
	}

	now := time.Now().In(loc)

	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// Card is the full single-tournament view — just Body, since the title
// already is the headline here (unlike training/schedule's card, which
// prefixes a generic "Тренировка"/"Слот расписания" label).
func Card(c *model.UserCompetition, today time.Time) string {
	return Body(c, today)
}

// Body formats a tournament's title/date(-range)/city/status, plus a
// result line once the result exists and the tournament has started —
// shared by the card view and create's "saved" confirmation screen.
func Body(c *model.UserCompetition, today time.Time) string {
	body := c.Title + "\n\n📅 " + DateRange(c.Date, c.EndDate)

	if c.City != nil && *c.City != "" {
		body += "\n🏙 " + *c.City
	}

	body += "\n" + Status(c.Date, c.EndDate, today)

	if !today.Before(c.Date) && c.Result != nil && *c.Result != "" {
		body += "\n\n🏅 " + *c.Result
	}

	return body
}

// Keyboard is the card's own layout: a URL button to the tournament page
// (if a link is set), "➕ ..." quick-add shortcuts for city/link/end
// date/result that appear only while the field is still empty — once a
// field is filled, its only way to change is "✏️ Изменить" (which lists
// every field, marking the still-empty ones the same "➕" way), there's no
// standalone button for an already-filled field anymore. This is the one
// keyboard every screen that shows a tournament card uses — the
// just-saved confirmation, the card view, and the screen after any edit —
// so filling a field from any of them removes its own quick-add shortcut
// everywhere else too.
func Keyboard(c *model.UserCompetition, today time.Time) dto.Keyboard {
	var kb dto.Keyboard

	if c.URL != nil && *c.URL != "" {
		kb = append(kb, dto.Row(dto.Button{Label: "🔗 Страница соревнования", URL: *c.URL}))
	}

	var quickAdds []dto.Button
	if c.City == nil || *c.City == "" {
		quickAdds = append(quickAdds, fieldQuickAdd(c.ID, "city", "Город"))
	}
	if c.URL == nil || *c.URL == "" {
		quickAdds = append(quickAdds, fieldQuickAdd(c.ID, "url", "Ссылка"))
	}
	if len(quickAdds) > 0 {
		kb = append(kb, quickAdds)
	}

	if c.EndDate == nil {
		kb = append(kb, dto.Row(fieldQuickAdd(c.ID, "end_date", "Дата окончания")))
	}

	if !today.Before(c.Date) && (c.Result == nil || *c.Result == "") {
		kb = append(kb, dto.Row(fieldQuickAdd(c.ID, "result", "Результат")))
	}

	// CompetitionID is only ever set once this record is already linked to
	// a catalog entry (submitted, auto-attached, or copied in via
	// catalog/add) — offering to submit it again makes no sense at that
	// point.
	if c.URL != nil && *c.URL != "" && c.CompetitionID == nil {
		kb = append(kb, dto.Row(dto.Button{Label: "📤 Предложить в каталог", Data: fmt.Sprintf("%s%d", callbackSubmitPrefix, c.ID)}))
	}

	kb = append(kb,
		dto.Row(dto.Button{Label: "✏️ Изменить", Data: fmt.Sprintf("%s%d", callbackEditPrefix, c.ID)}),
		dto.Row(dto.Button{Label: "🗑️ Удалить", Data: fmt.Sprintf("%s%d", callbackDeletePrefix, c.ID)}),
		dto.Row(dto.Button{Label: "← Назад", Data: callbackCompetitionList}),
	)

	return kb
}

func fieldQuickAdd(id int64, field, label string) dto.Button {
	return dto.Button{Label: "➕ " + label, Data: fmt.Sprintf("%s%d:%s", callbackEditPrefix, id, field)}
}

// Status computes a tournament's status purely from today vs. its dates —
// nothing is stored in the database for this. Takes plain date/endDate
// rather than *model.UserCompetition so catalog/info (which renders the
// same status for a *model.Competition — identical shape, different type)
// can reuse it without converting between the two.
func Status(date time.Time, endDate *time.Time, today time.Time) string {
	end := date
	if endDate != nil {
		end = *endDate
	}

	switch {
	case today.Before(date):
		days := daysBetween(today, date)
		if days == 1 {
			return "Завтра"
		}

		return "Через " + FormatDays(days)

	case !today.After(end):
		return "Идёт сейчас"

	default:
		return "Прошёл " + FormatDays(daysBetween(end, today)) + " назад"
	}
}

// DateRange formats a single date, or a "date – date" range when endDate is
// set — shared with catalog/info for the same reason as Status.
func DateRange(date time.Time, endDate *time.Time) string {
	if endDate == nil {
		return FormatDate(date)
	}

	return FormatDate(date) + " – " + FormatDate(*endDate)
}

func daysBetween(from, to time.Time) int {
	return int(to.Sub(from).Hours() / 24)
}

var russianMonthsGenitive = [...]string{
	"января", "февраля", "марта", "апреля", "мая", "июня",
	"июля", "августа", "сентября", "октября", "ноября", "декабря",
}

func FormatDate(d time.Time) string {
	return fmt.Sprintf("%d %s %d", d.Day(), russianMonthsGenitive[d.Month()-1], d.Year())
}

var russianMonthsShort = [...]string{
	"янв", "фев", "мар", "апр", "мая", "июн",
	"июл", "авг", "сен", "окт", "ноя", "дек",
}

// FormatDateRow is the abbreviated form a list row uses, e.g. "15 ноя".
func FormatDateRow(d time.Time) string {
	return fmt.Sprintf("%d %s", d.Day(), russianMonthsShort[d.Month()-1])
}

func FormatDays(n int) string {
	return fmt.Sprintf("%d %s", n, daysWord(n))
}

// daysWord picks the right Russian plural form of "день" for n.
func daysWord(n int) string {
	if n%100 >= 11 && n%100 <= 14 {
		return "дней"
	}

	switch n % 10 {
	case 1:
		return "день"
	case 2, 3, 4:
		return "дня"
	default:
		return "дней"
	}
}
