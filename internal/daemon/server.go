package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"mkanban/internal/application/dto"
	"mkanban/internal/di"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/config"
	"mkanban/pkg/slug"
)

// Server represents the daemon server
type Server struct {
	container           *di.Container
	config              *config.Config
	listener            net.Listener
	sessionManager      *SessionManager
	actionManager       *ActionManager
	timeTrackingManager *TimeTrackingManager
	mu                  sync.RWMutex
	subscribers         map[string]map[net.Conn]chan *Notification // boardID -> conn -> channel
	subMu               sync.RWMutex
}

// NewServer creates a new daemon server
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize dependency injection container
	container, err := di.InitializeContainer()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize container: %w", err)
	}

	return &Server{
		container:   container,
		config:      cfg,
		subscribers: make(map[string]map[net.Conn]chan *Notification),
	}, nil
}

// Start starts the daemon server
func (s *Server) Start() error {
	if err := s.acquireLock(); err != nil {
		return err
	}

	ctx := context.Background()

	// Initialize session manager if session tracking use cases are available
	if s.container.TrackSessionsUseCase != nil &&
		s.container.SessionTracker != nil &&
		s.container.ChangeWatcher != nil &&
		s.container.BoardSyncStrategies != nil {

		s.sessionManager = NewSessionManager(
			s.container.Config,
			s.container.TrackSessionsUseCase,
			s.container.SessionTracker,
			s.container.ChangeWatcher,
			s.container.BoardSyncStrategies,
		)

		if err := s.sessionManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start session manager: %w", err)
		}
		fmt.Println("Session tracking started")
	}

	// Initialize action manager if action use cases are available
	if s.container.EvaluateActionsUseCase != nil &&
		s.container.ExecuteActionUseCase != nil &&
		s.container.ProcessEventUseCase != nil &&
		s.container.ActionRepo != nil &&
		s.container.EventBus != nil {

		s.actionManager = NewActionManager(
			s.container.Config,
			s.container.EvaluateActionsUseCase,
			s.container.ExecuteActionUseCase,
			s.container.ProcessEventUseCase,
			s.container.ActionRepo,
			s.container.EventBus,
		)

		if err := s.actionManager.Start(); err != nil {
			return fmt.Errorf("failed to start action manager: %w", err)
		}
		fmt.Println("Action manager started")
	}

	socketDir := s.config.Daemon.SocketDir
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	socketPath := filepath.Join(socketDir, s.config.Daemon.SocketName)

	// Remove existing socket if it exists
	if err := os.RemoveAll(socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}

	s.listener = listener
	fmt.Printf("Daemon listening on %s\n", socketPath)

	return s.acceptConnections()
}

// acceptConnections handles incoming connections
func (s *Server) acceptConnections() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return fmt.Errorf("failed to accept connection: %w", err)
		}

		go s.handleConnection(conn)
	}
}

// handleConnection handles a single client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		s.cleanupSubscriber(conn)
		conn.Close()
	}()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	// Handle requests in a loop for persistent connections
	for {
		var req Request
		if err := decoder.Decode(&req); err != nil {
			// Connection closed or error
			return
		}

		// Handle subscribe request specially - it keeps the connection open
		if req.Type == RequestSubscribe {
			s.handleSubscribe(conn, encoder, &req)
			return // Connection will be kept open for notifications
		}

		// Handle regular request-response
		resp := s.handleRequest(&req)
		if err := encoder.Encode(resp); err != nil {
			fmt.Printf("Failed to encode response: %v\n", err)
			return
		}

		// For regular requests, close after response
		// (except for subscribe which is handled above)
		if req.Type != RequestUnsubscribe && req.Type != RequestPing {
			return
		}
	}
}

// handleRequest processes a request and returns a response
func (s *Server) handleRequest(req *Request) *Response {
	ctx := context.Background()

	switch req.Type {
	case RequestGetBoard:
		return s.handleGetBoard(ctx, req)
	case RequestListBoards:
		return s.handleListBoards(ctx)
	case RequestCreateBoard:
		return s.handleCreateBoard(ctx, req)
	case RequestAddTask:
		return s.handleAddTask(ctx, req)
	case RequestMoveTask:
		return s.handleMoveTask(ctx, req)
	case RequestUpdateTask:
		return s.handleUpdateTask(ctx, req)
	case RequestDeleteTask:
		return s.handleDeleteTask(ctx, req)
	case RequestAddColumn:
		return s.handleAddColumn(ctx, req)
	case RequestDeleteColumn:
		return s.handleDeleteColumn(ctx, req)
	case RequestGetActiveBoard:
		return s.handleGetActiveBoard(ctx, req)
	case RequestPing:
		return &Response{Success: true, Data: "pong"}

	case RequestStartTimer:
		return s.handleStartTimer(ctx, req)
	case RequestStopTimer:
		return s.handleStopTimer(ctx, req)
	case RequestGetActiveTimers:
		return s.handleGetActiveTimers(ctx)
	case RequestListTimeLogs:
		return s.handleListTimeLogs(ctx, req)
	case RequestAddTimeEntry:
		return s.handleAddTimeEntry(ctx, req)

	case RequestCreateProject:
		return s.handleCreateProject(ctx, req)
	case RequestGetProject:
		return s.handleGetProject(ctx, req)
	case RequestListProjects:
		return s.handleListProjects(ctx)
	case RequestUpdateProject:
		return s.handleUpdateProject(ctx, req)
	case RequestDeleteProject:
		return s.handleDeleteProject(ctx, req)

	case RequestScheduleTask:
		return s.handleScheduleTask(ctx, req)
	case RequestCreateMeeting:
		return s.handleCreateMeeting(ctx, req)

	default:
		return &Response{
			Success: false,
			Error:   fmt.Sprintf("unknown request type: %s", req.Type),
		}
	}
}

// handleGetBoard returns a specific board
func (s *Server) handleGetBoard(ctx context.Context, req *Request) *Response {
	var payload GetBoardPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	boardDTO, err := s.container.GetBoardUseCase.Execute(ctx, payload.BoardID)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	return &Response{Success: true, Data: boardDTO}
}

// handleListBoards returns all boards
func (s *Server) handleListBoards(ctx context.Context) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	boards, err := s.container.ListBoardsUseCase.Execute(ctx)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	return &Response{Success: true, Data: boards}
}

// handleCreateBoard creates a new board
func (s *Server) handleCreateBoard(ctx context.Context, req *Request) *Response {
	var payload CreateBoardPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	createReq := dto.CreateBoardRequest{
		Name:        payload.Name,
		Description: payload.Description,
	}

	boardDTO, err := s.container.CreateBoardUseCase.Execute(ctx, createReq)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	return &Response{Success: true, Data: boardDTO}
}

// handleAddTask adds a new task to a column
func (s *Server) handleAddTask(ctx context.Context, req *Request) *Response {
	var payload AddTaskPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	taskDTO, err := s.container.CreateTaskUseCase.Execute(ctx, payload.BoardID, payload.TaskRequest)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	// Notify subscribers
	s.notifySubscribers(payload.BoardID, &Notification{
		Type:    NotificationTaskCreated,
		BoardID: payload.BoardID,
		Data:    taskDTO,
	})

	return &Response{Success: true, Data: taskDTO}
}

// handleMoveTask moves a task between columns
func (s *Server) handleMoveTask(ctx context.Context, req *Request) *Response {
	var payload MoveTaskPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	moveReq := dto.MoveTaskRequest{
		TaskID:           payload.TaskID,
		TargetColumnName: payload.TargetColumnName,
	}

	boardDTO, err := s.container.MoveTaskUseCase.Execute(ctx, payload.BoardID, moveReq)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	// Notify subscribers
	s.notifySubscribers(payload.BoardID, &Notification{
		Type:    NotificationTaskMoved,
		BoardID: payload.BoardID,
		Data:    boardDTO,
	})

	return &Response{Success: true, Data: boardDTO}
}

// handleUpdateTask updates an existing task
func (s *Server) handleUpdateTask(ctx context.Context, req *Request) *Response {
	var payload UpdateTaskPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	taskDTO, err := s.container.UpdateTaskUseCase.Execute(ctx, payload.BoardID, payload.TaskID, payload.TaskRequest)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	// Notify subscribers
	s.notifySubscribers(payload.BoardID, &Notification{
		Type:    NotificationTaskUpdated,
		BoardID: payload.BoardID,
		Data:    taskDTO,
	})

	return &Response{Success: true, Data: taskDTO}
}

// handleDeleteTask deletes a task
func (s *Server) handleDeleteTask(ctx context.Context, req *Request) *Response {
	var payload DeleteTaskPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// We'll need to add DeleteTaskUseCase
	// For now, return not implemented
	return &Response{Success: false, Error: "delete task not yet implemented"}
}

// handleAddColumn adds a new column
func (s *Server) handleAddColumn(ctx context.Context, req *Request) *Response {
	var payload AddColumnPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	boardDTO, err := s.container.CreateColumnUseCase.Execute(ctx, payload.BoardID, payload.ColumnRequest)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	return &Response{Success: true, Data: boardDTO}
}

// handleDeleteColumn deletes a column
func (s *Server) handleDeleteColumn(ctx context.Context, req *Request) *Response {
	var payload DeleteColumnPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// We'll need to add DeleteColumnUseCase
	// For now, return not implemented
	return &Response{Success: false, Error: "delete column not yet implemented"}
}

// handleGetActiveBoard returns the board ID for the active session
func (s *Server) handleGetActiveBoard(ctx context.Context, req *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if we have the GetActiveSessionBoardUseCase
	if s.container.GetActiveSessionBoardUseCase == nil {
		return &Response{Success: false, Error: "session tracking not available"}
	}

	var sessionName string
	if req != nil && req.Payload != nil {
		var payload GetActiveBoardPayload
		if err := s.decodePayload(req.Payload, &payload); err == nil {
			sessionName = payload.SessionName
		}
	}

	boardID, err := s.container.GetActiveSessionBoardUseCase.Execute(ctx, sessionName)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	// Return the board ID (may be empty if no active session)
	return &Response{Success: true, Data: map[string]string{"board_id": boardID}}
}

// decodePayload decodes request payload into target struct
func (s *Server) decodePayload(payload interface{}, target interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return nil
}

// sendError sends an error response
func (s *Server) sendError(encoder *json.Encoder, message string) {
	resp := &Response{
		Success: false,
		Error:   message,
	}
	encoder.Encode(resp)
}

// Stop stops the daemon server
func (s *Server) Stop() error {
	// Stop time tracking manager if it exists
	if s.timeTrackingManager != nil {
		if err := s.timeTrackingManager.Stop(); err != nil {
			fmt.Printf("Error stopping time tracking manager: %v\n", err)
		}
	}

	// Stop action manager if it exists
	if s.actionManager != nil {
		if err := s.actionManager.Stop(); err != nil {
			fmt.Printf("Error stopping action manager: %v\n", err)
		}
	}

	// Stop session manager if it exists
	if s.sessionManager != nil {
		if err := s.sessionManager.Stop(); err != nil {
			fmt.Printf("Error stopping session manager: %v\n", err)
		}
	}

	s.releaseLock()

	// Close the listener
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) lockFilePath() string {
	return filepath.Join(s.config.Daemon.SocketDir, "mkanban.pid")
}

func (s *Server) acquireLock() error {
	lockPath := s.lockFilePath()

	if err := os.MkdirAll(s.config.Daemon.SocketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	data, err := os.ReadFile(lockPath)
	if err == nil {
		var pid int
		if _, err := fmt.Sscanf(string(data), "%d", &pid); err == nil {
			if s.isProcessRunning(pid) {
				return fmt.Errorf("daemon already running (pid %d)", pid)
			}
		}
		os.Remove(lockPath)
	}

	pid := os.Getpid()
	if err := os.WriteFile(lockPath, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

func (s *Server) releaseLock() {
	os.Remove(s.lockFilePath())
}

func (s *Server) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// GetSocketPath returns the socket path from config
func GetSocketPath(cfg *config.Config) string {
	return filepath.Join(cfg.Daemon.SocketDir, cfg.Daemon.SocketName)
}

// handleSubscribe handles a subscription request
func (s *Server) handleSubscribe(conn net.Conn, encoder *json.Encoder, req *Request) {
	var payload SubscribePayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		s.sendError(encoder, err.Error())
		return
	}

	// Create notification channel for this connection
	notifChan := make(chan *Notification, 10)

	// Register subscriber
	s.subMu.Lock()
	if _, exists := s.subscribers[payload.BoardID]; !exists {
		s.subscribers[payload.BoardID] = make(map[net.Conn]chan *Notification)
	}
	s.subscribers[payload.BoardID][conn] = notifChan
	s.subMu.Unlock()

	// Send success response
	resp := &Response{Success: true, Data: "subscribed"}
	if err := encoder.Encode(resp); err != nil {
		s.cleanupSubscriber(conn)
		return
	}

	// Start sending notifications
	for notification := range notifChan {
		if err := encoder.Encode(notification); err != nil {
			// Connection error, cleanup and exit
			s.cleanupSubscriber(conn)
			return
		}
	}
}

// notifySubscribers sends a notification to all subscribers of a board
func (s *Server) notifySubscribers(boardID string, notification *Notification) {
	s.subMu.RLock()
	defer s.subMu.RUnlock()

	subscribers, exists := s.subscribers[boardID]
	if !exists {
		return
	}

	// Send notification to all subscribers
	for _, ch := range subscribers {
		select {
		case ch <- notification:
			// Notification sent
		default:
			// Channel full, skip this subscriber
		}
	}
}

// cleanupSubscriber removes a connection from all subscriptions
func (s *Server) cleanupSubscriber(conn net.Conn) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	// Find and remove this connection from all boards
	for boardID, subscribers := range s.subscribers {
		if ch, exists := subscribers[conn]; exists {
			close(ch)
			delete(subscribers, conn)

			// Clean up empty board subscriptions
			if len(subscribers) == 0 {
				delete(s.subscribers, boardID)
			}
		}
	}
}

func (s *Server) handleStartTimer(ctx context.Context, req *Request) *Response {
	if s.timeTrackingManager == nil {
		return &Response{Success: false, Error: "time tracking not available"}
	}

	var payload StartTimerPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	log, err := s.timeTrackingManager.StartTimer(ctx, payload.ProjectID, nil, payload.Description)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	return &Response{Success: true, Data: map[string]interface{}{
		"id":         log.ID(),
		"project_id": log.ProjectID(),
		"start_time": log.StartTime(),
		"running":    log.IsRunning(),
	}}
}

func (s *Server) handleStopTimer(ctx context.Context, req *Request) *Response {
	if s.timeTrackingManager == nil {
		return &Response{Success: false, Error: "time tracking not available"}
	}

	var payload StopTimerPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	log, err := s.timeTrackingManager.StopTimer(ctx, payload.ProjectID, nil)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	return &Response{Success: true, Data: map[string]interface{}{
		"id":         log.ID(),
		"project_id": log.ProjectID(),
		"start_time": log.StartTime(),
		"end_time":   log.EndTime(),
		"duration":   log.Duration().Seconds(),
	}}
}

func (s *Server) handleGetActiveTimers(ctx context.Context) *Response {
	if s.timeTrackingManager == nil {
		return &Response{Success: false, Error: "time tracking not available"}
	}

	timers := s.timeTrackingManager.GetActiveTimers()
	result := make([]map[string]interface{}, 0, len(timers))

	for _, t := range timers {
		entry := map[string]interface{}{
			"id":         t.ID(),
			"project_id": t.ProjectID(),
			"source":     t.Source().String(),
			"start_time": t.StartTime(),
			"duration":   t.Duration().Seconds(),
		}
		if t.TaskID() != nil {
			entry["task_id"] = t.TaskID().String()
		}
		result = append(result, entry)
	}

	return &Response{Success: true, Data: result}
}

func (s *Server) handleListTimeLogs(ctx context.Context, req *Request) *Response {
	var payload ListTimeLogsPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	var startDate, endDate *time.Time
	if payload.StartDate != nil {
		t, err := time.Parse("2006-01-02", *payload.StartDate)
		if err != nil {
			return &Response{Success: false, Error: "invalid start_date format, use YYYY-MM-DD"}
		}
		startDate = &t
	}
	if payload.EndDate != nil {
		t, err := time.Parse("2006-01-02", *payload.EndDate)
		if err != nil {
			return &Response{Success: false, Error: "invalid end_date format, use YYYY-MM-DD"}
		}
		endDate = &t
	}

	var logs []*entity.TimeLog
	var err error

	if payload.TaskID != nil {
		var taskID *valueobject.TaskID
		taskID, err = valueobject.ParseTaskID(*payload.TaskID)
		if err != nil {
			return &Response{Success: false, Error: "invalid task_id format"}
		}
		logs, err = s.container.TimeLogRepo.FindByTask(ctx, taskID)
	} else if payload.ProjectID != "" && startDate != nil && endDate != nil {
		logs, err = s.container.TimeLogRepo.FindByDateRange(ctx, payload.ProjectID, *startDate, *endDate)
	} else if payload.ProjectID != "" {
		logs, err = s.container.TimeLogRepo.FindByProject(ctx, payload.ProjectID)
	} else {
		logs, err = s.container.TimeLogRepo.FindRunning(ctx)
	}

	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	result := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		entry := map[string]interface{}{
			"id":          log.ID(),
			"project_id":  log.ProjectID(),
			"source":      log.Source().String(),
			"start_time":  log.StartTime(),
			"description": log.Description(),
			"running":     log.IsRunning(),
		}
		if log.TaskID() != nil {
			entry["task_id"] = log.TaskID().String()
		}
		if log.EndTime() != nil {
			entry["end_time"] = log.EndTime()
			entry["duration"] = log.Duration().Seconds()
		}
		result = append(result, entry)
	}

	return &Response{Success: true, Data: result}
}

func (s *Server) handleAddTimeEntry(ctx context.Context, req *Request) *Response {
	var payload AddTimeEntryPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	startTime, err := time.Parse(time.RFC3339, payload.StartTime)
	if err != nil {
		return &Response{Success: false, Error: "invalid start_time format, use RFC3339"}
	}

	id := uuid.New().String()
	duration := time.Duration(payload.Duration) * time.Second
	endTime := startTime.Add(duration)

	log := entity.NewTimeLogWithDuration(
		id,
		payload.ProjectID,
		payload.TaskID,
		entity.TimeLogSourceManual,
		startTime,
		endTime,
		payload.Description,
	)

	if err := s.container.TimeLogRepo.Save(ctx, log); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	return &Response{Success: true, Data: map[string]interface{}{
		"id":          log.ID(),
		"project_id":  log.ProjectID(),
		"start_time":  log.StartTime(),
		"end_time":    log.EndTime(),
		"duration":    log.Duration().Seconds(),
		"description": log.Description(),
	}}
}

func (s *Server) handleCreateProject(ctx context.Context, req *Request) *Response {
	var payload CreateProjectPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	id := uuid.New().String()
	project, err := entity.NewProject(id, payload.Name, payload.Description)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	if payload.WorkingDir != "" {
		project.SetWorkingDir(payload.WorkingDir)
	}

	if err := s.container.ProjectRepo.Save(ctx, project); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	return &Response{Success: true, Data: map[string]interface{}{
		"id":   project.ID(),
		"name": project.Name(),
		"slug": project.Slug(),
	}}
}

func (s *Server) handleGetProject(ctx context.Context, req *Request) *Response {
	var payload GetProjectPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	project, err := s.container.ProjectRepo.FindByID(ctx, payload.ProjectID)
	if err != nil {
		project, err = s.container.ProjectRepo.FindBySlug(ctx, payload.ProjectID)
		if err != nil {
			return &Response{Success: false, Error: "project not found"}
		}
	}

	return &Response{Success: true, Data: map[string]interface{}{
		"id":          project.ID(),
		"name":        project.Name(),
		"slug":        project.Slug(),
		"description": project.Description(),
		"working_dir": project.WorkingDir(),
		"archived":    project.Archived(),
		"created_at":  project.CreatedAt(),
		"modified_at": project.ModifiedAt(),
	}}
}

func (s *Server) handleListProjects(ctx context.Context) *Response {
	projects, err := s.container.ProjectRepo.FindAll(ctx)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	result := make([]map[string]interface{}, 0, len(projects))
	for _, p := range projects {
		result = append(result, map[string]interface{}{
			"id":          p.ID(),
			"name":        p.Name(),
			"slug":        p.Slug(),
			"description": p.Description(),
			"archived":    p.Archived(),
			"board_count": p.BoardCount(),
			"task_count":  p.TotalTaskCount(),
		})
	}

	return &Response{Success: true, Data: result}
}

func (s *Server) handleUpdateProject(ctx context.Context, req *Request) *Response {
	var payload UpdateProjectPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	project, err := s.container.ProjectRepo.FindByID(ctx, payload.ProjectID)
	if err != nil {
		return &Response{Success: false, Error: "project not found"}
	}

	if payload.Name != nil {
		if err := project.UpdateName(*payload.Name); err != nil {
			return &Response{Success: false, Error: err.Error()}
		}
	}
	if payload.Description != nil {
		project.UpdateDescription(*payload.Description)
	}
	if payload.WorkingDir != nil {
		project.SetWorkingDir(*payload.WorkingDir)
	}
	if payload.Archived != nil {
		if *payload.Archived {
			project.Archive()
		} else {
			project.Unarchive()
		}
	}

	if err := s.container.ProjectRepo.Save(ctx, project); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	return &Response{Success: true, Data: map[string]interface{}{
		"id":   project.ID(),
		"name": project.Name(),
		"slug": project.Slug(),
	}}
}

func (s *Server) handleDeleteProject(ctx context.Context, req *Request) *Response {
	var payload DeleteProjectPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	if err := s.container.ProjectRepo.Delete(ctx, payload.ProjectID); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	return &Response{Success: true, Data: "project deleted"}
}

func (s *Server) handleScheduleTask(ctx context.Context, req *Request) *Response {
	fmt.Println("[Schedule] Decoding payload...")
	var payload ScheduleTaskPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}
	fmt.Printf("[Schedule] Task ID: %s, Date: %s\n", payload.TaskID, payload.Date)

	scheduledDate, err := time.Parse("2006-01-02", payload.Date)
	if err != nil {
		return &Response{Success: false, Error: "invalid date format, use YYYY-MM-DD"}
	}

	fmt.Println("[Schedule] Finding task...")
	var board *entity.Board
	var task *entity.Task
	var columnName string

	taskID, err := valueobject.ParseTaskID(payload.TaskID)
	if err == nil {
		fmt.Println("[Schedule] Using full ID...")
		board, task, columnName, err = s.findTaskAcrossBoards(ctx, taskID)
	} else {
		fmt.Println("[Schedule] Using short ID...")
		board, task, columnName, err = s.findTaskByShortID(ctx, payload.TaskID)
	}
	fmt.Printf("[Schedule] Find result: err=%v\n", err)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	fmt.Println("[Schedule] Setting scheduled date...")
	task.SetScheduledDate(scheduledDate)

	if payload.Time != nil {
		fmt.Println("[Schedule] Setting scheduled time...")
		scheduledTime, err := time.Parse("15:04", *payload.Time)
		if err != nil {
			return &Response{Success: false, Error: "invalid time format, use HH:MM"}
		}
		fullTime := time.Date(
			scheduledDate.Year(), scheduledDate.Month(), scheduledDate.Day(),
			scheduledTime.Hour(), scheduledTime.Minute(), 0, 0,
			scheduledDate.Location(),
		)
		task.SetScheduledTime(fullTime)
	}

	if payload.Duration != nil {
		fmt.Println("[Schedule] Setting duration...")
		duration, err := time.ParseDuration(*payload.Duration)
		if err != nil {
			return &Response{Success: false, Error: "invalid duration format, use e.g., 2h, 30m"}
		}
		task.SetTimeBlock(duration)
	}

	fmt.Printf("[Schedule] Board ID: %s, Column: %s\n", board.ID(), columnName)
	fmt.Println("[Schedule] Saving task...")
	if err := s.container.BoardRepo.SaveTask(ctx, board.ID(), columnName, task); err != nil {
		fmt.Printf("[Schedule] Save error: %v\n", err)
		return &Response{Success: false, Error: err.Error()}
	}
	fmt.Println("[Schedule] Task saved!")

	fmt.Println("[Schedule] Notifying subscribers...")
	s.notifySubscribers(board.ID(), &Notification{
		Type:    NotificationTaskUpdated,
		BoardID: board.ID(),
		Data:    task,
	})
	fmt.Println("[Schedule] Returning response...")

	return &Response{Success: true, Data: map[string]interface{}{
		"id":             task.ID().String(),
		"title":          task.Title(),
		"scheduled_date": task.ScheduledDate(),
		"scheduled_time": task.ScheduledTime(),
		"time_block":     task.TimeBlock(),
	}}
}

func (s *Server) handleCreateMeeting(ctx context.Context, req *Request) *Response {
	var payload CreateMeetingPayload
	if err := s.decodePayload(req.Payload, &payload); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	scheduledDate, err := time.Parse("2006-01-02", payload.Date)
	if err != nil {
		return &Response{Success: false, Error: "invalid date format, use YYYY-MM-DD"}
	}

	board, err := s.container.BoardRepo.FindByID(ctx, payload.BoardID)
	if err != nil {
		return &Response{Success: false, Error: "board not found"}
	}

	taskSlug := slug.Generate(payload.Title)
	taskID, err := board.GenerateNextTaskID(taskSlug)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	task, err := entity.NewTask(
		taskID,
		payload.Title,
		"",
		valueobject.PriorityMedium,
		valueobject.StatusTodo,
	)
	if err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	task.SetTaskType(entity.TaskTypeMeeting)
	task.SetScheduledDate(scheduledDate)

	if payload.Time != nil {
		scheduledTime, err := time.Parse("15:04", *payload.Time)
		if err != nil {
			return &Response{Success: false, Error: "invalid time format, use HH:MM"}
		}
		fullTime := time.Date(
			scheduledDate.Year(), scheduledDate.Month(), scheduledDate.Day(),
			scheduledTime.Hour(), scheduledTime.Minute(), 0, 0,
			scheduledDate.Location(),
		)
		task.SetScheduledTime(fullTime)
	}

	if payload.Duration != nil {
		duration, err := time.ParseDuration(*payload.Duration)
		if err != nil {
			return &Response{Success: false, Error: "invalid duration format, use e.g., 2h, 30m"}
		}
		task.SetTimeBlock(duration)
	}

	meetingData := &entity.MeetingData{}
	if len(payload.Attendees) > 0 {
		meetingData.Attendees = payload.Attendees
	}
	if payload.Location != nil {
		meetingData.Location = *payload.Location
	}
	task.SetMeetingData(meetingData)

	columns := board.Columns()
	if len(columns) == 0 {
		return &Response{Success: false, Error: "board has no columns"}
	}

	targetColumn := columns[0]
	for _, col := range columns {
		name := col.Name()
		if name == "Meetings" || name == "Calendar" {
			targetColumn = col
			break
		}
	}

	if err := targetColumn.AddTask(task); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	if err := s.container.BoardRepo.Save(ctx, board); err != nil {
		return &Response{Success: false, Error: err.Error()}
	}

	s.notifySubscribers(board.ID(), &Notification{
		Type:    NotificationTaskCreated,
		BoardID: board.ID(),
		Data:    task,
	})

	return &Response{Success: true, Data: map[string]interface{}{
		"id":             task.ID().String(),
		"title":          task.Title(),
		"task_type":      task.TaskType(),
		"scheduled_date": task.ScheduledDate(),
		"scheduled_time": task.ScheduledTime(),
		"time_block":     task.TimeBlock(),
	}}
}

func (s *Server) findTaskAcrossBoards(ctx context.Context, taskID *valueobject.TaskID) (*entity.Board, *entity.Task, string, error) {
	boards, err := s.container.ListBoardsUseCase.Execute(ctx)
	if err != nil {
		return nil, nil, "", err
	}

	for _, boardInfo := range boards {
		board, err := s.container.BoardRepo.FindByID(ctx, boardInfo.ID)
		if err != nil {
			continue
		}
		task, column, err := board.FindTask(taskID)
		if err == nil {
			return board, task, column.Name(), nil
		}
	}

	return nil, nil, "", fmt.Errorf("task not found: %s", taskID.String())
}

func (s *Server) findTaskByShortID(ctx context.Context, shortID string) (*entity.Board, *entity.Task, string, error) {
	fmt.Printf("[findTaskByShortID] Looking for: %s\n", shortID)
	shortIDRegex := regexp.MustCompile(`^([A-Z]{3})-(\d+)$`)
	matches := shortIDRegex.FindStringSubmatch(strings.TrimSpace(shortID))
	if matches == nil {
		return nil, nil, "", fmt.Errorf("invalid short ID format: %s", shortID)
	}

	prefix := matches[1]
	number, _ := strconv.Atoi(matches[2])
	fmt.Printf("[findTaskByShortID] Prefix: %s, Number: %d\n", prefix, number)

	fmt.Println("[findTaskByShortID] Listing boards...")
	boards, err := s.container.ListBoardsUseCase.Execute(ctx)
	if err != nil {
		return nil, nil, "", err
	}
	fmt.Printf("[findTaskByShortID] Found %d boards\n", len(boards))

	activeSession := s.sessionManager.GetActiveSession()
	var activeBoardID string
	if activeSession != nil {
		activeBoardID = slug.Generate(activeSession.Name())
		fmt.Printf("[findTaskByShortID] Active session board: %s\n", activeBoardID)
	}

	searchBoard := func(boardID string) (*entity.Board, *entity.Task, string, error) {
		board, err := s.container.BoardRepo.FindByID(ctx, boardID)
		if err != nil {
			return nil, nil, "", err
		}
		for _, col := range board.Columns() {
			for _, task := range col.Tasks() {
				if task.ID().Prefix() == prefix && task.ID().Number() == number {
					return board, task, col.Name(), nil
				}
			}
		}
		return nil, nil, "", nil
	}

	if activeBoardID != "" {
		fmt.Printf("[findTaskByShortID] Searching active board first: %s\n", activeBoardID)
		board, task, colName, err := searchBoard(activeBoardID)
		if err == nil && task != nil {
			fmt.Println("[findTaskByShortID] Found task in active board!")
			return board, task, colName, nil
		}
	}

	for _, boardInfo := range boards {
		if boardInfo.ID == activeBoardID {
			continue
		}
		fmt.Printf("[findTaskByShortID] Checking board: %s\n", boardInfo.ID)
		board, task, colName, err := searchBoard(boardInfo.ID)
		if err == nil && task != nil {
			fmt.Println("[findTaskByShortID] Found task!")
			return board, task, colName, nil
		}
	}

	return nil, nil, "", fmt.Errorf("task not found: %s", shortID)
}
