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
	body := c.Title + "\n\n📅 " + dateRange(c)

	if c.City != nil && *c.City != "" {
		body += "\n🏙 " + *c.City
	}

	body += "\n" + Status(c, today)

	if !today.Before(c.Date) && c.Result != nil && *c.Result != "" {
		body += "\n\n🏅 " + *c.Result
	}

	return body
}

// Keyboard is the card's own layout: a URL button to the tournament page
// (if a link is set), quick toggles for city/link/end date that read
// "➕ ..." when the field is still empty and "✏️ ..." once it's filled, a
// result shortcut (once the tournament has started), then
// Изменить/Удалить/Назад. This is the one keyboard every screen that shows
// a tournament card uses — the just-saved confirmation, the card view, and
// the screen after any edit — so a field filled in from any of them
// updates its own toggle everywhere else too.
func Keyboard(c *model.UserCompetition, today time.Time) dto.Keyboard {
	var kb dto.Keyboard

	if c.URL != nil && *c.URL != "" {
		kb = append(kb, dto.Row(dto.Button{Label: "🔗 Страница турнира", URL: *c.URL}))
	}

	kb = append(kb, dto.Row(
		fieldToggle(c.ID, "city", "Город", c.City != nil && *c.City != ""),
		fieldToggle(c.ID, "url", "Ссылка", c.URL != nil && *c.URL != ""),
	))

	kb = append(kb, dto.Row(fieldToggle(c.ID, "end_date", "Дата окончания", c.EndDate != nil)))

	if !today.Before(c.Date) {
		kb = append(kb, dto.Row(dto.Button{Label: "🏅 Результат", Data: fmt.Sprintf("%s%d:result", callbackEditPrefix, c.ID)}))
	}

	kb = append(kb,
		dto.Row(dto.Button{Label: "✏️ Изменить", Data: fmt.Sprintf("%s%d", callbackEditPrefix, c.ID)}),
		dto.Row(dto.Button{Label: "🗑️ Удалить", Data: fmt.Sprintf("%s%d", callbackDeletePrefix, c.ID)}),
		dto.Row(dto.Button{Label: "← Назад", Data: callbackCompetitionList}),
	)

	return kb
}

func fieldToggle(id int64, field, label string, filled bool) dto.Button {
	icon := "➕"
	if filled {
		icon = "✏️"
	}

	return dto.Button{Label: icon + " " + label, Data: fmt.Sprintf("%s%d:%s", callbackEditPrefix, id, field)}
}

// Status computes a tournament's status purely from today vs. its dates —
// nothing is stored in the database for this.
func Status(c *model.UserCompetition, today time.Time) string {
	end := c.Date
	if c.EndDate != nil {
		end = *c.EndDate
	}

	switch {
	case today.Before(c.Date):
		days := daysBetween(today, c.Date)
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

func dateRange(c *model.UserCompetition) string {
	if c.EndDate == nil {
		return FormatDate(c.Date)
	}

	return FormatDate(c.Date) + " – " + FormatDate(*c.EndDate)
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
