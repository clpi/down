package date

import "time"

type (
	Date struct {
		Month   time.Month
		Weekday time.Weekday
		Time    time.Time
	}
	NaturalDate struct {
		Date Date
		Map  map[string]int
	}
)

func ParseNaturalDate() *NaturalDate {
	return nil
}
