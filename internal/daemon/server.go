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
	mu             sync.RWMutex
}

// NewServer creates a new daemon server
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize dependency injection container
	container, err := di.InitializeContainer()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize container: %w", err)
	}

	return &Server{
		container: container,
		config:    cfg,
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
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var req Request
	if err := decoder.Decode(&req); err != nil {
		s.sendError(encoder, fmt.Sprintf("failed to decode request: %v", err))
		return
	}

	resp := s.handleRequest(&req)
	if err := encoder.Encode(resp); err != nil {
		fmt.Printf("Failed to encode response: %v\n", err)
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
