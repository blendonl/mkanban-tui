package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/external"
)

type CalendarSyncService struct {
	calendarClient *external.GoogleCalendarClient
	boardRepo      repository.BoardRepository
	projectRepo    repository.ProjectRepository
	syncState      *SyncState
	mu             sync.RWMutex
}

type SyncState struct {
	LastSyncTime   time.Time
	SyncedEvents   map[string]string // calendarEventID -> taskID
	SyncedTasks    map[string]string // taskID -> calendarEventID
	ConflictPolicy ConflictPolicy
}

type ConflictPolicy string

const (
	ConflictPolicyCalendarWins ConflictPolicy = "calendar_wins"
	ConflictPolicyTaskWins     ConflictPolicy = "task_wins"
	ConflictPolicyNewerWins    ConflictPolicy = "newer_wins"
	ConflictPolicyAskUser      ConflictPolicy = "ask_user"
)

type SyncResult struct {
	EventsCreated   int
	EventsUpdated   int
	EventsDeleted   int
	TasksCreated    int
	TasksUpdated    int
	Conflicts       []SyncConflict
	Errors          []error
	LastSyncTime    time.Time
}

type SyncConflict struct {
	EventID     string
	TaskID      string
	EventTitle  string
	TaskTitle   string
	Description string
}

func NewCalendarSyncService(
	calendarClient *external.GoogleCalendarClient,
	boardRepo repository.BoardRepository,
	projectRepo repository.ProjectRepository,
) *CalendarSyncService {
	return &CalendarSyncService{
		calendarClient: calendarClient,
		boardRepo:      boardRepo,
		projectRepo:    projectRepo,
		syncState: &SyncState{
			SyncedEvents:   make(map[string]string),
			SyncedTasks:    make(map[string]string),
			ConflictPolicy: ConflictPolicyNewerWins,
		},
	}
}

func (s *CalendarSyncService) SetConflictPolicy(policy ConflictPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncState.ConflictPolicy = policy
}

func (s *CalendarSyncService) SyncAll(ctx context.Context) (*SyncResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := &SyncResult{
		LastSyncTime: time.Now(),
	}

	pullResult, err := s.pullFromCalendar(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("pull sync failed: %w", err))
	} else {
		result.TasksCreated = pullResult.TasksCreated
		result.TasksUpdated = pullResult.TasksUpdated
		result.Conflicts = append(result.Conflicts, pullResult.Conflicts...)
	}

	pushResult, err := s.pushToCalendar(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("push sync failed: %w", err))
	} else {
		result.EventsCreated = pushResult.EventsCreated
		result.EventsUpdated = pushResult.EventsUpdated
		result.EventsDeleted = pushResult.EventsDeleted
		result.Conflicts = append(result.Conflicts, pushResult.Conflicts...)
	}

	s.syncState.LastSyncTime = result.LastSyncTime
	return result, nil
}

func (s *CalendarSyncService) PullFromCalendar(ctx context.Context) (*SyncResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pullFromCalendar(ctx)
}

func (s *CalendarSyncService) pullFromCalendar(ctx context.Context) (*SyncResult, error) {
	result := &SyncResult{
		LastSyncTime: time.Now(),
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0)

	events, err := s.calendarClient.GetEvents(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calendar events: %w", err)
	}

	boards, err := s.boardRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch boards: %w", err)
	}

	var meetingBoard *entity.Board
	for _, b := range boards {
		if b.Name() == "Meetings" || b.Name() == "Calendar" {
			meetingBoard = b
			break
		}
	}

	if meetingBoard == nil && len(boards) > 0 {
		meetingBoard = boards[0]
	}

	if meetingBoard == nil {
		return result, nil
	}

	for _, event := range events {
		if taskID, exists := s.syncState.SyncedEvents[event.ID]; exists {
			task := s.findTaskByID(meetingBoard, taskID)
			if task != nil {
				if s.eventNewerThanTask(event, task) {
					s.updateTaskFromEvent(task, event)
					result.TasksUpdated++
				}
			}
			continue
		}

		existingTask := s.findTaskByCalendarID(meetingBoard, event.ID)
		if existingTask != nil {
			continue
		}

		task := s.createTaskFromEvent(meetingBoard, event)
		if task != nil {
			s.syncState.SyncedEvents[event.ID] = task.ID().String()
			s.syncState.SyncedTasks[task.ID().String()] = event.ID
			result.TasksCreated++
		}
	}

	return result, nil
}

func (s *CalendarSyncService) PushToCalendar(ctx context.Context) (*SyncResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pushToCalendar(ctx)
}

func (s *CalendarSyncService) pushToCalendar(ctx context.Context) (*SyncResult, error) {
	result := &SyncResult{
		LastSyncTime: time.Now(),
	}

	boards, err := s.boardRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch boards: %w", err)
	}

	for _, board := range boards {
		for _, col := range board.Columns() {
			for _, task := range col.Tasks() {
				if task.TaskType() != entity.TaskTypeMeeting {
					continue
				}

				if task.ScheduledDate() == nil {
					continue
				}

				eventID, exists := s.syncState.SyncedTasks[task.ID().String()]
				if exists {
					event := s.createEventFromTask(task)
					event.ID = eventID
					if err := s.calendarClient.UpdateEvent(ctx, event); err != nil {
						result.Errors = append(result.Errors, err)
					} else {
						result.EventsUpdated++
					}
				} else {
					event := s.createEventFromTask(task)
					created, err := s.calendarClient.CreateEvent(ctx, event)
					if err != nil {
						result.Errors = append(result.Errors, err)
					} else {
						s.syncState.SyncedEvents[created.ID] = task.ID().String()
						s.syncState.SyncedTasks[task.ID().String()] = created.ID
						result.EventsCreated++
					}
				}
			}
		}
	}

	return result, nil
}

func (s *CalendarSyncService) findTaskByID(board *entity.Board, taskID string) *entity.Task {
	for _, col := range board.Columns() {
		for _, task := range col.Tasks() {
			if task.ID().String() == taskID {
				return task
			}
		}
	}
	return nil
}

func (s *CalendarSyncService) findTaskByCalendarID(board *entity.Board, calendarID string) *entity.Task {
	for _, col := range board.Columns() {
		for _, task := range col.Tasks() {
			if id, ok := task.GetMetadata("calendar_event_id"); ok && id == calendarID {
				return task
			}
		}
	}
	return nil
}

func (s *CalendarSyncService) eventNewerThanTask(event external.CalendarEvent, task *entity.Task) bool {
	return event.StartTime.After(task.ModifiedAt())
}

func (s *CalendarSyncService) updateTaskFromEvent(task *entity.Task, event external.CalendarEvent) {
	task.UpdateTitle(event.Title)
	task.UpdateDescription(event.Description)
	task.SetScheduledDate(event.StartTime)
	if !event.IsAllDay {
		task.SetScheduledTime(event.StartTime)
		duration := event.EndTime.Sub(event.StartTime)
		task.SetTimeBlock(duration)
	}
	if event.Location != "" {
		task.SetMetadata("location", event.Location)
	}
	if event.MeetingLink != "" {
		task.SetMetadata("meeting_link", event.MeetingLink)
	}
}

func (s *CalendarSyncService) createTaskFromEvent(board *entity.Board, event external.CalendarEvent) *entity.Task {
	var todoColumn *entity.Column
	todoColumn, _ = board.GetColumn("To Do")
	if todoColumn == nil {
		todoColumn, _ = board.GetColumn("todo")
	}
	if todoColumn == nil {
		columns := board.Columns()
		if len(columns) > 0 {
			todoColumn = columns[0]
		}
	}

	if todoColumn == nil {
		return nil
	}

	taskID, err := board.GenerateNextTaskID(valueobject.GenerateSlug(event.Title))
	if err != nil {
		return nil
	}

	task, err := entity.NewTask(taskID, event.Title, event.Description, valueobject.PriorityMedium, valueobject.StatusTodo)
	if err != nil {
		return nil
	}

	if err := todoColumn.AddTask(task); err != nil {
		return nil
	}

	task.SetTaskType(entity.TaskTypeMeeting)
	task.SetScheduledDate(event.StartTime)
	if !event.IsAllDay {
		task.SetScheduledTime(event.StartTime)
		duration := event.EndTime.Sub(event.StartTime)
		task.SetTimeBlock(duration)
	}
	task.SetMetadata("calendar_event_id", event.ID)
	if event.Location != "" {
		task.SetMetadata("location", event.Location)
	}
	if event.MeetingLink != "" {
		task.SetMetadata("meeting_link", event.MeetingLink)
	}
	if len(event.Attendees) > 0 {
		meetingData := &entity.MeetingData{
			Attendees: event.Attendees,
			Location:  event.Location,
		}
		task.SetMeetingData(meetingData)
	}

	return task
}

func (s *CalendarSyncService) createEventFromTask(task *entity.Task) external.CalendarEvent {
	event := external.CalendarEvent{
		Title:       task.Title(),
		Description: task.Description(),
	}

	if task.ScheduledDate() != nil {
		event.StartTime = *task.ScheduledDate()
		if task.ScheduledTime() != nil {
			event.StartTime = *task.ScheduledTime()
		}
		if task.TimeBlock() != nil {
			event.EndTime = event.StartTime.Add(*task.TimeBlock())
		} else {
			event.EndTime = event.StartTime.Add(time.Hour)
		}
	}

	if task.MeetingData() != nil {
		event.Attendees = task.MeetingData().Attendees
		event.Location = task.MeetingData().Location
	}

	if loc, ok := task.GetMetadata("location"); ok {
		event.Location = loc
	}

	return event
}

func (s *CalendarSyncService) GetLastSyncTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.syncState.LastSyncTime
}

func (s *CalendarSyncService) GetSyncStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]interface{}{
		"last_sync":      s.syncState.LastSyncTime,
		"synced_events":  len(s.syncState.SyncedEvents),
		"synced_tasks":   len(s.syncState.SyncedTasks),
		"conflict_policy": s.syncState.ConflictPolicy,
	}
}
