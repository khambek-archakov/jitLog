package stats

import (
	"strconv"
	"strings"
	"time"
)

// callbackStatsPeriodPrefix is followed by a period token ("week", "month",
// "year" or "all") — this is both the entry point (the main menu's own
// "📊 Статистика" button already points at stats:period:week) and every
// period-tab button's callback.
const callbackStatsPeriodPrefix = "stats:period:"

type period string

const (
	periodWeek  period = "week"
	periodMonth period = "month"
	periodYear  period = "year"
	periodAll   period = "all"
)

func parsePeriod(data string) (period, bool) {
	if !strings.HasPrefix(data, callbackStatsPeriodPrefix) {
		return "", false
	}

	switch p := period(strings.TrimPrefix(data, callbackStatsPeriodPrefix)); p {
	case periodWeek, periodMonth, periodYear, periodAll:
		return p, true
	default:
		return "", false
	}
}

func periodLabel(p period) string {
	switch p {
	case periodWeek:
		return "Неделя"
	case periodMonth:
		return "Месяц"
	case periodYear:
		return "Год"
	default:
		return "Всё время"
	}
}

// periodStart is the inclusive lower bound of p, anchored at now. periodAll
// has no lower bound.
func periodStart(p period, now time.Time) time.Time {
	switch p {
	case periodWeek:
		return mondayOf(now)
	case periodMonth:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case periodYear:
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	default:
		return time.Time{}
	}
}

// mondayOf returns the Monday (00:00) of d's calendar week.
func mondayOf(d time.Time) time.Time {
	weekday := int(d.Weekday())
	if weekday == 0 {
		weekday = 7 // time.Sunday == 0 — shift so Monday=1..Sunday=7.
	}

	day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())

	return day.AddDate(0, 0, -(weekday - 1))
}

var monthsGenitive = [...]string{
	"января", "февраля", "марта", "апреля", "мая", "июня",
	"июля", "августа", "сентября", "октября", "ноября", "декабря",
}

var monthsNominative = [...]string{
	"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
	"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
}

// periodRangeText describes p's current window — empty for periodAll,
// which has no bounded range to show.
func periodRangeText(p period, now time.Time) string {
	switch p {
	case periodWeek:
		monday := mondayOf(now)
		sunday := monday.AddDate(0, 0, 6)

		return weekRangeText(monday, sunday)
	case periodMonth:
		return monthsNominative[now.Month()-1] + " " + strconv.Itoa(now.Year())
	case periodYear:
		return strconv.Itoa(now.Year())
	default:
		return ""
	}
}

func weekRangeText(monday, sunday time.Time) string {
	if monday.Month() == sunday.Month() {
		return strconv.Itoa(monday.Day()) + "–" + strconv.Itoa(sunday.Day()) + " " + monthsGenitive[monday.Month()-1]
	}

	return strconv.Itoa(monday.Day()) + " " + monthsGenitive[monday.Month()-1] +
		" – " + strconv.Itoa(sunday.Day()) + " " + monthsGenitive[sunday.Month()-1]
}
