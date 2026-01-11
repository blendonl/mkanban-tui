package valueobject

import (
	"fmt"
	"time"
)

type RecurrenceFrequency string

const (
	FrequencyDaily   RecurrenceFrequency = "daily"
	FrequencyWeekly  RecurrenceFrequency = "weekly"
	FrequencyMonthly RecurrenceFrequency = "monthly"
	FrequencyYearly  RecurrenceFrequency = "yearly"
)

func (f RecurrenceFrequency) IsValid() bool {
	switch f {
	case FrequencyDaily, FrequencyWeekly, FrequencyMonthly, FrequencyYearly:
		return true
	}
	return false
}

type RecurrenceRule struct {
	frequency  RecurrenceFrequency
	interval   int
	daysOfWeek []time.Weekday
	dayOfMonth int
	endDate    *time.Time
	count      int
}

func NewRecurrenceRule(frequency RecurrenceFrequency, interval int) (*RecurrenceRule, error) {
	if !frequency.IsValid() {
		return nil, fmt.Errorf("invalid recurrence frequency: %s", frequency)
	}
	if interval < 1 {
		interval = 1
	}

	return &RecurrenceRule{
		frequency:  frequency,
		interval:   interval,
		daysOfWeek: make([]time.Weekday, 0),
	}, nil
}

func (r *RecurrenceRule) Frequency() RecurrenceFrequency {
	return r.frequency
}

func (r *RecurrenceRule) Interval() int {
	return r.interval
}

func (r *RecurrenceRule) DaysOfWeek() []time.Weekday {
	daysCopy := make([]time.Weekday, len(r.daysOfWeek))
	copy(daysCopy, r.daysOfWeek)
	return daysCopy
}

func (r *RecurrenceRule) SetDaysOfWeek(days []time.Weekday) {
	r.daysOfWeek = make([]time.Weekday, len(days))
	copy(r.daysOfWeek, days)
}

func (r *RecurrenceRule) DayOfMonth() int {
	return r.dayOfMonth
}

func (r *RecurrenceRule) SetDayOfMonth(day int) {
	r.dayOfMonth = day
}

func (r *RecurrenceRule) EndDate() *time.Time {
	if r.endDate == nil {
		return nil
	}
	copy := *r.endDate
	return &copy
}

func (r *RecurrenceRule) SetEndDate(date time.Time) {
	r.endDate = &date
}

func (r *RecurrenceRule) Count() int {
	return r.count
}

func (r *RecurrenceRule) SetCount(count int) {
	r.count = count
}

func (r *RecurrenceRule) NextOccurrence(after time.Time) time.Time {
	switch r.frequency {
	case FrequencyDaily:
		return after.AddDate(0, 0, r.interval)
	case FrequencyWeekly:
		return after.AddDate(0, 0, 7*r.interval)
	case FrequencyMonthly:
		return after.AddDate(0, r.interval, 0)
	case FrequencyYearly:
		return after.AddDate(r.interval, 0, 0)
	}
	return after
}

func (r *RecurrenceRule) String() string {
	switch r.frequency {
	case FrequencyDaily:
		if r.interval == 1 {
			return "daily"
		}
		return fmt.Sprintf("every %d days", r.interval)
	case FrequencyWeekly:
		if r.interval == 1 {
			return "weekly"
		}
		return fmt.Sprintf("every %d weeks", r.interval)
	case FrequencyMonthly:
		if r.interval == 1 {
			return "monthly"
		}
		return fmt.Sprintf("every %d months", r.interval)
	case FrequencyYearly:
		if r.interval == 1 {
			return "yearly"
		}
		return fmt.Sprintf("every %d years", r.interval)
	}
	return ""
}
