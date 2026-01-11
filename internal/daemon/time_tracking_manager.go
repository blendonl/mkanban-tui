package daemon

import (
	"context"
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/service"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/config"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TimeTrackingManager struct {
	config         *config.Config
	projectRepo    repository.ProjectRepository
	timeLogRepo    repository.TimeLogRepository
	sessionTracker service.SessionTracker
	vcsProvider    service.VCSProvider

	activeTimers   map[string]*entity.TimeLog
	autoTimers     map[string]*entity.TimeLog
	currentProject *entity.Project
	currentTaskID  *valueobject.TaskID

	mu       sync.RWMutex
	stopChan chan struct{}
	stopped  bool
}

func NewTimeTrackingManager(
	config *config.Config,
	projectRepo repository.ProjectRepository,
	timeLogRepo repository.TimeLogRepository,
	sessionTracker service.SessionTracker,
	vcsProvider service.VCSProvider,
) *TimeTrackingManager {
	return &TimeTrackingManager{
		config:         config,
		projectRepo:    projectRepo,
		timeLogRepo:    timeLogRepo,
		sessionTracker: sessionTracker,
		vcsProvider:    vcsProvider,
		activeTimers:   make(map[string]*entity.TimeLog),
		autoTimers:     make(map[string]*entity.TimeLog),
		stopChan:       make(chan struct{}),
		stopped:        false,
	}
}

func (tm *TimeTrackingManager) Start(ctx context.Context) error {
	if !tm.config.TimeTracking.Enabled {
		fmt.Println("[TimeTrackingManager] Time tracking is disabled in config")
		return nil
	}

	if tm.sessionTracker == nil || !tm.sessionTracker.IsAvailable() {
		fmt.Println("[TimeTrackingManager] Session tracker not available")
		return nil
	}

	fmt.Println("[TimeTrackingManager] Starting time tracking")
	go tm.pollLoop(ctx)

	return nil
}

func (tm *TimeTrackingManager) Stop() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.stopped {
		return nil
	}

	tm.stopped = true
	close(tm.stopChan)

	ctx := context.Background()
	for _, timer := range tm.autoTimers {
		if timer.IsRunning() {
			_ = timer.Stop(time.Now())
			_ = tm.timeLogRepo.Save(ctx, timer)
		}
	}

	return nil
}

func (tm *TimeTrackingManager) StartTimer(ctx context.Context, projectID string, taskID *valueobject.TaskID, description string) (*entity.TimeLog, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	key := projectID
	if taskID != nil {
		key = taskID.String()
	}

	if existing, ok := tm.activeTimers[key]; ok && existing.IsRunning() {
		return existing, nil
	}

	id := uuid.New().String()
	log, err := entity.NewTimeLog(id, projectID, entity.TimeLogSourceTimer, time.Now())
	if err != nil {
		return nil, err
	}

	if taskID != nil {
		log.SetTaskID(taskID)
	}
	if description != "" {
		log.SetDescription(description)
	}

	if err := tm.timeLogRepo.Save(ctx, log); err != nil {
		return nil, err
	}

	tm.activeTimers[key] = log
	fmt.Printf("[TimeTrackingManager] Started timer for %s\n", key)

	return log, nil
}

func (tm *TimeTrackingManager) StopTimer(ctx context.Context, projectID string, taskID *valueobject.TaskID) (*entity.TimeLog, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	key := projectID
	if taskID != nil {
		key = taskID.String()
	}

	timer, ok := tm.activeTimers[key]
	if !ok || !timer.IsRunning() {
		return nil, entity.ErrTimeLogNotFound
	}

	if err := timer.Stop(time.Now()); err != nil {
		return nil, err
	}

	if err := tm.timeLogRepo.Save(ctx, timer); err != nil {
		return nil, err
	}

	delete(tm.activeTimers, key)
	fmt.Printf("[TimeTrackingManager] Stopped timer for %s (duration: %s)\n", key, timer.Duration())

	return timer, nil
}

func (tm *TimeTrackingManager) GetActiveTimers() []*entity.TimeLog {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	timers := make([]*entity.TimeLog, 0, len(tm.activeTimers)+len(tm.autoTimers))
	for _, t := range tm.activeTimers {
		if t.IsRunning() {
			timers = append(timers, t)
		}
	}
	for _, t := range tm.autoTimers {
		if t.IsRunning() {
			timers = append(timers, t)
		}
	}
	return timers
}

func (tm *TimeTrackingManager) AddManualEntry(ctx context.Context, projectID string, taskID *valueobject.TaskID, startTime time.Time, duration time.Duration, description string) (*entity.TimeLog, error) {
	id := uuid.New().String()
	log, err := entity.NewTimeLog(id, projectID, entity.TimeLogSourceManual, startTime)
	if err != nil {
		return nil, err
	}

	if taskID != nil {
		log.SetTaskID(taskID)
	}
	if description != "" {
		log.SetDescription(description)
	}

	if err := log.SetDuration(duration); err != nil {
		return nil, err
	}

	if err := tm.timeLogRepo.Save(ctx, log); err != nil {
		return nil, err
	}

	return log, nil
}

func (tm *TimeTrackingManager) pollLoop(ctx context.Context) {
	tm.syncAutoTracking(ctx)

	pollInterval := time.Duration(tm.config.SessionTracking.PollInterval) * time.Second
	if pollInterval == 0 {
		pollInterval = 5 * time.Second
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.syncAutoTracking(ctx)
		case <-tm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (tm *TimeTrackingManager) syncAutoTracking(ctx context.Context) {
	if !tm.config.TimeTracking.AutoTrack {
		return
	}

	activeSession, err := tm.sessionTracker.GetActiveSession()
	if err != nil || activeSession == nil {
		tm.pauseAutoTimers(ctx)
		return
	}

	project, taskID := tm.detectProjectAndTask(ctx, activeSession)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	projectChanged := (tm.currentProject == nil && project != nil) ||
		(tm.currentProject != nil && project == nil) ||
		(tm.currentProject != nil && project != nil && tm.currentProject.ID() != project.ID())

	taskChanged := (tm.currentTaskID == nil && taskID != nil) ||
		(tm.currentTaskID != nil && taskID == nil) ||
		(tm.currentTaskID != nil && taskID != nil && !tm.currentTaskID.Equal(taskID))

	if projectChanged || taskChanged {
		tm.pauseAutoTimersLocked(ctx)

		tm.currentProject = project
		tm.currentTaskID = taskID

		if project != nil {
			tm.startAutoTimerLocked(ctx, project, taskID)
		}
	}
}

func (tm *TimeTrackingManager) detectProjectAndTask(ctx context.Context, session *entity.Session) (*entity.Project, *valueobject.TaskID) {
	workingDir := session.WorkingDir()
	if workingDir == "" {
		return nil, nil
	}

	projects, err := tm.projectRepo.FindAll(ctx)
	if err != nil {
		return nil, nil
	}

	var matchedProject *entity.Project
	for _, p := range projects {
		if p.WorkingDir() == workingDir {
			matchedProject = p
			break
		}
	}

	if matchedProject == nil {
		return nil, nil
	}

	var taskID *valueobject.TaskID
	if tm.vcsProvider != nil {
		branch, err := tm.vcsProvider.GetCurrentBranch(workingDir)
		if err == nil && branch != "" {
			taskID = tm.extractTaskIDFromBranch(branch)
		}
	}

	return matchedProject, taskID
}

func (tm *TimeTrackingManager) extractTaskIDFromBranch(branch string) *valueobject.TaskID {
	patterns := []string{
		`^(?:feature|bugfix|fix|hotfix|chore|refactor)/([A-Z]{3}-\d+-[a-z0-9-]+)`,
		`([A-Z]{3}-\d+-[a-z0-9-]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(branch)
		if len(matches) > 1 {
			taskID, err := valueobject.ParseTaskID(matches[1])
			if err == nil {
				return taskID
			}
		}
	}

	return nil
}

func (tm *TimeTrackingManager) pauseAutoTimers(ctx context.Context) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.pauseAutoTimersLocked(ctx)
}

func (tm *TimeTrackingManager) pauseAutoTimersLocked(ctx context.Context) {
	for key, timer := range tm.autoTimers {
		if timer.IsRunning() {
			_ = timer.Stop(time.Now())
			_ = tm.timeLogRepo.Save(ctx, timer)
			fmt.Printf("[TimeTrackingManager] Auto-paused timer for %s\n", key)
		}
	}
	tm.autoTimers = make(map[string]*entity.TimeLog)
	tm.currentProject = nil
	tm.currentTaskID = nil
}

func (tm *TimeTrackingManager) startAutoTimerLocked(ctx context.Context, project *entity.Project, taskID *valueobject.TaskID) {
	key := project.ID()
	if taskID != nil {
		key = taskID.String()
	}

	if _, hasManual := tm.activeTimers[key]; hasManual {
		return
	}

	id := uuid.New().String()
	log, err := entity.NewTimeLog(id, project.ID(), entity.TimeLogSourceTmux, time.Now())
	if err != nil {
		return
	}

	if taskID != nil {
		log.SetTaskID(taskID)
	}
	log.SetMetadata("session_type", "tmux")
	log.SetMetadata("auto_tracked", "true")

	if err := tm.timeLogRepo.Save(ctx, log); err != nil {
		return
	}

	tm.autoTimers[key] = log

	if taskID != nil {
		fmt.Printf("[TimeTrackingManager] Auto-started timer for project %s, task %s\n", project.Name(), taskID.String())
	} else {
		fmt.Printf("[TimeTrackingManager] Auto-started timer for project %s\n", project.Name())
	}
}
