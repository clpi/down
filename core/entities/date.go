package entities

import (
	"time"
)

type (
	Month   = int
	Weekday = int
)

const (
	DateSecond   = iota
	DateMinute   = iota
	DateHour     = iota
	DateDay      = iota
	DateWeek     = iota
	DateMonth    = iota
	DateYear     = iota
	DateTime     = iota
	DateDatetime = iota
	DateDate     = iota
)
const (
	Monday    Weekday = 0
	Tuesday   Weekday = 1
	Wednesday Weekday = 2
	Thursday  Weekday = 3
	Friday    Weekday = 4
	Saturday  Weekday = 5
	Sunday    Weekday = 6
)
const (
	Jan int = 1
	Feb int = 2
	Mar int = 3
	Apr int = 4
	May     = 5
	Jun     = 6
	Jul     = 7
	Aug     = 8
	Sep     = 9
	Oct     = 10
	Nov     = 11
	Dec     = 12
)

type (
	Time struct {
		Hour, Minute, Second int
	}
	Date struct {
		Weekday Weekday
		Week    int
		Month   Month
		Year    int
	}
	Datetime struct {
		Date   Date
		Time   Time
		Notify bool
	}
	Unit struct {
		// "week" = 1, "month" = 2, "year" = 3, "day" = 4,
		// hour = -1 minute = -2 second = -3
		Unit string
		// "2" = 2, every "2" hoursr
		// "every other" = 2, "every third" = 3, "every" = 1
		Offset int
		// how many times to repeat
		Count int
		// start and end times
		Start time.Time
		End   time.Time
	}
	Repeat struct {
		Date Unit
		Time Unit
	}
)
