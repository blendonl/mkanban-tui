package entity

import (
	"fmt"
	"time"
)

type Weekday int

const (
	WeekdaySunday Weekday = iota
	WeekdayMonday
	WeekdayTuesday
	WeekdayWednesday
	WeekdayThursday
	WeekdayFriday
	WeekdaySaturday
)

func (w Weekday) String() string {
	return [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}[w]
}

type WorkSchedule struct {
	id          string
	name        string
	isDefault   bool
	workDays    map[Weekday]*DaySchedule
	exceptions  []ScheduleException
	timezone    string
	createdAt   time.Time
	modifiedAt  time.Time
}

type DaySchedule struct {
	enabled    bool
	startTime  TimeOfDay
	endTime    TimeOfDay
	breakStart *TimeOfDay
	breakEnd   *TimeOfDay
}

type TimeOfDay struct {
	Hour   int
	Minute int
}

func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

func (t TimeOfDay) ToMinutes() int {
	return t.Hour*60 + t.Minute
}

func (t TimeOfDay) After(other TimeOfDay) bool {
	return t.ToMinutes() > other.ToMinutes()
}

func (t TimeOfDay) Before(other TimeOfDay) bool {
	return t.ToMinutes() < other.ToMinutes()
}

type ScheduleException struct {
	Date        time.Time
	Type        ExceptionType
	Description string
	StartTime   *TimeOfDay
	EndTime     *TimeOfDay
}

type ExceptionType string

const (
	ExceptionTypeHoliday  ExceptionType = "holiday"
	ExceptionTypeTimeOff  ExceptionType = "time_off"
	ExceptionTypeOverride ExceptionType = "override"
)

func NewWorkSchedule(id, name string) (*WorkSchedule, error) {
	if id == "" {
		return nil, fmt.Errorf("work schedule id cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("work schedule name cannot be empty")
	}

	now := time.Now()
	return &WorkSchedule{
		id:         id,
		name:       name,
		workDays:   make(map[Weekday]*DaySchedule),
		exceptions: make([]ScheduleException, 0),
		timezone:   time.Local.String(),
		createdAt:  now,
		modifiedAt: now,
	}, nil
}

func NewDefaultWorkSchedule(id string) *WorkSchedule {
	schedule, _ := NewWorkSchedule(id, "Default Schedule")
	schedule.isDefault = true

	defaultStart := TimeOfDay{Hour: 9, Minute: 0}
	defaultEnd := TimeOfDay{Hour: 17, Minute: 0}
	lunchStart := TimeOfDay{Hour: 12, Minute: 0}
	lunchEnd := TimeOfDay{Hour: 13, Minute: 0}

	for day := WeekdayMonday; day <= WeekdayFriday; day++ {
		schedule.workDays[day] = &DaySchedule{
			enabled:    true,
			startTime:  defaultStart,
			endTime:    defaultEnd,
			breakStart: &lunchStart,
			breakEnd:   &lunchEnd,
		}
	}

	schedule.workDays[WeekdaySunday] = &DaySchedule{enabled: false}
	schedule.workDays[WeekdaySaturday] = &DaySchedule{enabled: false}

	return schedule
}

func (w *WorkSchedule) ID() string {
	return w.id
}

func (w *WorkSchedule) Name() string {
	return w.name
}

func (w *WorkSchedule) IsDefault() bool {
	return w.isDefault
}

func (w *WorkSchedule) Timezone() string {
	return w.timezone
}

func (w *WorkSchedule) CreatedAt() time.Time {
	return w.createdAt
}

func (w *WorkSchedule) ModifiedAt() time.Time {
	return w.modifiedAt
}

func (w *WorkSchedule) SetName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	w.name = name
	w.modifiedAt = time.Now()
	return nil
}

func (w *WorkSchedule) SetTimezone(tz string) {
	w.timezone = tz
	w.modifiedAt = time.Now()
}

func (w *WorkSchedule) SetDefault(isDefault bool) {
	w.isDefault = isDefault
	w.modifiedAt = time.Now()
}

func (w *WorkSchedule) SetDaySchedule(day Weekday, schedule *DaySchedule) {
	w.workDays[day] = schedule
	w.modifiedAt = time.Now()
}

func (w *WorkSchedule) GetDaySchedule(day Weekday) *DaySchedule {
	if schedule, ok := w.workDays[day]; ok {
		return schedule
	}
	return &DaySchedule{enabled: false}
}

func (w *WorkSchedule) AddException(exception ScheduleException) {
	w.exceptions = append(w.exceptions, exception)
	w.modifiedAt = time.Now()
}

func (w *WorkSchedule) RemoveException(date time.Time) {
	dateStr := date.Format("2006-01-02")
	filtered := make([]ScheduleException, 0, len(w.exceptions))
	for _, e := range w.exceptions {
		if e.Date.Format("2006-01-02") != dateStr {
			filtered = append(filtered, e)
		}
	}
	w.exceptions = filtered
	w.modifiedAt = time.Now()
}

func (w *WorkSchedule) GetExceptions() []ScheduleException {
	result := make([]ScheduleException, len(w.exceptions))
	copy(result, w.exceptions)
	return result
}

func (w *WorkSchedule) IsWorkingDay(date time.Time) bool {
	for _, e := range w.exceptions {
		if e.Date.Format("2006-01-02") == date.Format("2006-01-02") {
			if e.Type == ExceptionTypeHoliday || e.Type == ExceptionTypeTimeOff {
				return false
			}
			if e.Type == ExceptionTypeOverride && e.StartTime != nil {
				return true
			}
		}
	}

	weekday := Weekday(date.Weekday())
	if schedule, ok := w.workDays[weekday]; ok {
		return schedule.enabled
	}
	return false
}

func (w *WorkSchedule) GetWorkingHours(date time.Time) (*TimeOfDay, *TimeOfDay) {
	for _, e := range w.exceptions {
		if e.Date.Format("2006-01-02") == date.Format("2006-01-02") {
			if e.Type == ExceptionTypeOverride && e.StartTime != nil && e.EndTime != nil {
				return e.StartTime, e.EndTime
			}
			if e.Type == ExceptionTypeHoliday || e.Type == ExceptionTypeTimeOff {
				return nil, nil
			}
		}
	}

	weekday := Weekday(date.Weekday())
	if schedule, ok := w.workDays[weekday]; ok && schedule.enabled {
		return &schedule.startTime, &schedule.endTime
	}
	return nil, nil
}

func (w *WorkSchedule) IsWithinWorkingHours(t time.Time) bool {
	if !w.IsWorkingDay(t) {
		return false
	}

	start, end := w.GetWorkingHours(t)
	if start == nil || end == nil {
		return false
	}

	currentTime := TimeOfDay{Hour: t.Hour(), Minute: t.Minute()}
	return !currentTime.Before(*start) && currentTime.Before(*end)
}

func (w *WorkSchedule) GetAvailableMinutes(date time.Time) int {
	if !w.IsWorkingDay(date) {
		return 0
	}

	start, end := w.GetWorkingHours(date)
	if start == nil || end == nil {
		return 0
	}

	totalMinutes := end.ToMinutes() - start.ToMinutes()

	weekday := Weekday(date.Weekday())
	if schedule, ok := w.workDays[weekday]; ok && schedule.breakStart != nil && schedule.breakEnd != nil {
		breakMinutes := schedule.breakEnd.ToMinutes() - schedule.breakStart.ToMinutes()
		totalMinutes -= breakMinutes
	}

	return totalMinutes
}

func (w *WorkSchedule) GetNextWorkingDay(from time.Time) time.Time {
	current := from.AddDate(0, 0, 1)
	for i := 0; i < 14; i++ {
		if w.IsWorkingDay(current) {
			return current
		}
		current = current.AddDate(0, 0, 1)
	}
	return current
}

func (w *WorkSchedule) GetWorkingDaysInRange(start, end time.Time) []time.Time {
	var days []time.Time
	current := start
	for !current.After(end) {
		if w.IsWorkingDay(current) {
			days = append(days, current)
		}
		current = current.AddDate(0, 0, 1)
	}
	return days
}

func (d *DaySchedule) Enabled() bool {
	return d.enabled
}

func (d *DaySchedule) StartTime() TimeOfDay {
	return d.startTime
}

func (d *DaySchedule) EndTime() TimeOfDay {
	return d.endTime
}

func (d *DaySchedule) BreakStart() *TimeOfDay {
	return d.breakStart
}

func (d *DaySchedule) BreakEnd() *TimeOfDay {
	return d.breakEnd
}

func (d *DaySchedule) WorkingMinutes() int {
	if !d.enabled {
		return 0
	}

	total := d.endTime.ToMinutes() - d.startTime.ToMinutes()
	if d.breakStart != nil && d.breakEnd != nil {
		total -= d.breakEnd.ToMinutes() - d.breakStart.ToMinutes()
	}
	return total
}
