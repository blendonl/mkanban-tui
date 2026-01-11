package time

import (
	"context"
	"sort"
	"time"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
)

type TimeReportUseCase struct {
	timeLogRepo repository.TimeLogRepository
	projectRepo repository.ProjectRepository
}

func NewTimeReportUseCase(
	timeLogRepo repository.TimeLogRepository,
	projectRepo repository.ProjectRepository,
) *TimeReportUseCase {
	return &TimeReportUseCase{
		timeLogRepo: timeLogRepo,
		projectRepo: projectRepo,
	}
}

type TimeReport struct {
	Period        ReportPeriod
	TotalDuration time.Duration
	ByProject     []ProjectTimeEntry
	ByDay         []DayTimeEntry
	ByTask        []TaskTimeEntry
	BySource      map[entity.TimeLogSource]time.Duration
}

type ReportPeriod struct {
	Start time.Time
	End   time.Time
	Type  string
}

type ProjectTimeEntry struct {
	ProjectID   string
	ProjectName string
	Duration    time.Duration
	Percentage  float64
	TaskCount   int
}

type DayTimeEntry struct {
	Date     time.Time
	Duration time.Duration
	LogCount int
}

type TaskTimeEntry struct {
	TaskID      string
	TaskTitle   string
	ProjectID   string
	Duration    time.Duration
	LogCount    int
}

func (u *TimeReportUseCase) GenerateDailyReport(ctx context.Context, date time.Time) (*TimeReport, error) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := start.AddDate(0, 0, 1)

	return u.generateReport(ctx, start, end, "daily")
}

func (u *TimeReportUseCase) GenerateWeeklyReport(ctx context.Context, date time.Time) (*TimeReport, error) {
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := time.Date(date.Year(), date.Month(), date.Day()-weekday+1, 0, 0, 0, 0, date.Location())
	end := start.AddDate(0, 0, 7)

	return u.generateReport(ctx, start, end, "weekly")
}

func (u *TimeReportUseCase) GenerateMonthlyReport(ctx context.Context, year int, month time.Month) (*TimeReport, error) {
	start := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)

	return u.generateReport(ctx, start, end, "monthly")
}

func (u *TimeReportUseCase) GenerateCustomReport(ctx context.Context, start, end time.Time) (*TimeReport, error) {
	return u.generateReport(ctx, start, end, "custom")
}

func (u *TimeReportUseCase) generateReport(ctx context.Context, start, end time.Time, periodType string) (*TimeReport, error) {
	projects, err := u.projectRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	projectMap := make(map[string]*entity.Project)
	for _, p := range projects {
		projectMap[p.ID()] = p
	}

	report := &TimeReport{
		Period: ReportPeriod{
			Start: start,
			End:   end,
			Type:  periodType,
		},
		ByProject: make([]ProjectTimeEntry, 0),
		ByDay:     make([]DayTimeEntry, 0),
		ByTask:    make([]TaskTimeEntry, 0),
		BySource:  make(map[entity.TimeLogSource]time.Duration),
	}

	projectDurations := make(map[string]time.Duration)
	projectTaskCounts := make(map[string]int)
	dayDurations := make(map[string]time.Duration)
	dayLogCounts := make(map[string]int)
	taskDurations := make(map[string]time.Duration)
	taskLogCounts := make(map[string]int)
	taskTitles := make(map[string]string)
	taskProjects := make(map[string]string)

	for _, project := range projects {
		logs, err := u.timeLogRepo.FindByDateRange(ctx, project.ID(), start, end)
		if err != nil {
			continue
		}

		for _, log := range logs {
			duration := log.Duration()
			report.TotalDuration += duration

			projectDurations[log.ProjectID()] += duration
			projectTaskCounts[log.ProjectID()]++

			dayKey := log.StartTime().Format("2006-01-02")
			dayDurations[dayKey] += duration
			dayLogCounts[dayKey]++

			report.BySource[log.Source()] += duration

			if log.TaskID() != nil {
				taskID := log.TaskID().String()
				taskDurations[taskID] += duration
				taskLogCounts[taskID]++
				taskProjects[taskID] = log.ProjectID()
			}
		}
	}

	for projectID, duration := range projectDurations {
		var projectName string
		if p, ok := projectMap[projectID]; ok {
			projectName = p.Name()
		} else {
			projectName = projectID
		}

		percentage := 0.0
		if report.TotalDuration > 0 {
			percentage = float64(duration) / float64(report.TotalDuration) * 100
		}

		report.ByProject = append(report.ByProject, ProjectTimeEntry{
			ProjectID:   projectID,
			ProjectName: projectName,
			Duration:    duration,
			Percentage:  percentage,
			TaskCount:   projectTaskCounts[projectID],
		})
	}

	sort.Slice(report.ByProject, func(i, j int) bool {
		return report.ByProject[i].Duration > report.ByProject[j].Duration
	})

	current := start
	for !current.After(end) {
		dayKey := current.Format("2006-01-02")
		report.ByDay = append(report.ByDay, DayTimeEntry{
			Date:     current,
			Duration: dayDurations[dayKey],
			LogCount: dayLogCounts[dayKey],
		})
		current = current.AddDate(0, 0, 1)
	}

	for taskID, duration := range taskDurations {
		report.ByTask = append(report.ByTask, TaskTimeEntry{
			TaskID:    taskID,
			TaskTitle: taskTitles[taskID],
			ProjectID: taskProjects[taskID],
			Duration:  duration,
			LogCount:  taskLogCounts[taskID],
		})
	}

	sort.Slice(report.ByTask, func(i, j int) bool {
		return report.ByTask[i].Duration > report.ByTask[j].Duration
	})

	return report, nil
}

func (u *TimeReportUseCase) GetProjectSummary(ctx context.Context, projectID string, start, end time.Time) (*ProjectTimeSummary, error) {
	logs, err := u.timeLogRepo.FindByDateRange(ctx, projectID, start, end)
	if err != nil {
		return nil, err
	}

	summary := &ProjectTimeSummary{
		ProjectID:   projectID,
		Period:      ReportPeriod{Start: start, End: end},
		BySource:    make(map[entity.TimeLogSource]time.Duration),
		DailyAverage: 0,
	}

	daySet := make(map[string]bool)

	for _, log := range logs {
		duration := log.Duration()
		summary.TotalDuration += duration
		summary.LogCount++
		summary.BySource[log.Source()] += duration

		dayKey := log.StartTime().Format("2006-01-02")
		daySet[dayKey] = true
	}

	if len(daySet) > 0 {
		summary.DailyAverage = summary.TotalDuration / time.Duration(len(daySet))
	}

	return summary, nil
}

type ProjectTimeSummary struct {
	ProjectID     string
	Period        ReportPeriod
	TotalDuration time.Duration
	LogCount      int
	DailyAverage  time.Duration
	BySource      map[entity.TimeLogSource]time.Duration
}

func (u *TimeReportUseCase) CompareProjects(ctx context.Context, projectIDs []string, start, end time.Time) ([]ProjectTimeEntry, error) {
	var result []ProjectTimeEntry

	for _, projectID := range projectIDs {
		logs, err := u.timeLogRepo.FindByDateRange(ctx, projectID, start, end)
		if err != nil {
			continue
		}

		var totalDuration time.Duration
		taskSet := make(map[string]bool)

		for _, log := range logs {
			totalDuration += log.Duration()
			if log.TaskID() != nil {
				taskSet[log.TaskID().String()] = true
			}
		}

		project, _ := u.projectRepo.FindByID(ctx, projectID)
		projectName := projectID
		if project != nil {
			projectName = project.Name()
		}

		result = append(result, ProjectTimeEntry{
			ProjectID:   projectID,
			ProjectName: projectName,
			Duration:    totalDuration,
			TaskCount:   len(taskSet),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Duration > result[j].Duration
	})

	return result, nil
}
