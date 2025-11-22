package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"mkanban/internal/application/dto"
	"mkanban/internal/di"
	"mkanban/internal/infrastructure/config"
)

// Server represents the daemon server
type Server struct {
	container      *di.Container
	config         *config.Config
	listener       net.Listener
	sessionManager *SessionManager
	actionManager  *ActionManager
	mu             sync.RWMutex
	subscribers    map[string]map[net.Conn]chan *Notification // boardID -> conn -> channel
	subMu          sync.RWMutex
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
		return s.handleGetActiveBoard(ctx)
	case RequestPing:
		return &Response{Success: true, Data: "pong"}
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
func (s *Server) handleGetActiveBoard(ctx context.Context) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if we have the GetActiveSessionBoardUseCase
	if s.container.GetActiveSessionBoardUseCase == nil {
		return &Response{Success: false, Error: "session tracking not available"}
	}

	boardID, err := s.container.GetActiveSessionBoardUseCase.Execute(ctx)
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

	// Close the listener
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
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
