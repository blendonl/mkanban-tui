package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"mkanban/internal/application/dto"
	"mkanban/internal/infrastructure/config"
)

// Client represents a daemon client
type Client struct {
	config         *config.Config
	mu             sync.Mutex
	conn           net.Conn
	subConn        net.Conn
	subMu          sync.Mutex
	notifChan      chan *Notification
	stopChan       chan struct{}
	isSubscribed   bool
	socketPath string
}

// NewClient creates a new daemon client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		config:    cfg,
		notifChan: make(chan *Notification, 10),
		stopChan:  make(chan struct{}),
	}
}

// Connect establishes a connection to the daemon
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil // Already connected
	}

	socketPath := GetSocketPath(c.config)

	// Try to connect to existing daemon
	conn, err := net.Dial("unix", socketPath)
	if err == nil {
		c.conn = conn
		return nil
	}

	// Daemon not running, try to start it
	if err := c.startDaemon(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait for daemon to be ready and retry connection
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		time.Sleep(200 * time.Millisecond)
		conn, err = net.Dial("unix", socketPath)
		if err == nil {
			c.conn = conn
			return nil
		}
	}

	return fmt.Errorf("failed to connect to daemon after starting it: %w", err)
}

// startDaemon starts the daemon process
func (c *Client) startDaemon() error {
	// Find mkanbad binary
	mkanbadPath, err := exec.LookPath("mkanbad")
	if err != nil {
		// Try in the same directory as the current executable
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to find mkanbad binary: %w", err)
		}
		exeDir := filepath.Dir(exePath)
		mkanbadPath = filepath.Join(exeDir, "mkanbad")

		if _, err := os.Stat(mkanbadPath); err != nil {
			return fmt.Errorf("mkanbad binary not found in PATH or %s", exeDir)
		}
	}

	// Start daemon as background process
	cmd := exec.Command(mkanbadPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}

	// Detach from parent process
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("failed to release daemon process: %w", err)
	}

	return nil
}

// Close closes the connection to the daemon
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// sendRequest sends a request to the daemon and returns the response
func (c *Client) sendRequest(req *Request) (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("not connected to daemon")
	}

	// Set write deadline
	if err := c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send request
	encoder := json.NewEncoder(c.conn)
	if err := encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read response
	var resp Response
	decoder := json.NewDecoder(c.conn)
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("daemon error: %s", resp.Error)
	}

	return &resp, nil
}

// GetBoard retrieves a board from the daemon
func (c *Client) GetBoard(ctx context.Context, boardID string) (*dto.BoardDTO, error) {
	req := &Request{
		Type: RequestGetBoard,
		Payload: GetBoardPayload{
			BoardID: boardID,
		},
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Decode board from response data
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal board data: %w", err)
	}

	var board dto.BoardDTO
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}

	return &board, nil
}

// ListBoards retrieves all boards from the daemon
func (c *Client) ListBoards(ctx context.Context) ([]dto.BoardDTO, error) {
	req := &Request{
		Type: RequestListBoards,
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Decode boards from response data
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal boards data: %w", err)
	}

	var boards []dto.BoardDTO
	if err := json.Unmarshal(data, &boards); err != nil {
		return nil, fmt.Errorf("failed to unmarshal boards: %w", err)
	}

	return boards, nil
}

// GetActiveBoard retrieves the active board ID from the daemon
func (c *Client) GetActiveBoard(ctx context.Context) (string, error) {
	req := &Request{
		Type: RequestGetActiveBoard,
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return "", err
	}

	// Decode board ID from response data
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response data: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result["board_id"], nil
}

// CreateBoard creates a new board
func (c *Client) CreateBoard(ctx context.Context, name, description string) (*dto.BoardDTO, error) {
	req := &Request{
		Type: RequestCreateBoard,
		Payload: CreateBoardPayload{
			Name:        name,
			Description: description,
		},
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Decode board from response data
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal board data: %w", err)
	}

	var board dto.BoardDTO
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}

	return &board, nil
}

// CreateTask creates a new task
func (c *Client) CreateTask(ctx context.Context, boardID string, taskReq dto.CreateTaskRequest) (*dto.TaskDTO, error) {
	req := &Request{
		Type: RequestAddTask,
		Payload: AddTaskPayload{
			BoardID:     boardID,
			TaskRequest: taskReq,
		},
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Decode task from response data
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task data: %w", err)
	}

	var task dto.TaskDTO
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// MoveTask moves a task to a different column
func (c *Client) MoveTask(ctx context.Context, boardID, taskID, targetColumn string) (*dto.BoardDTO, error) {
	req := &Request{
		Type: RequestMoveTask,
		Payload: MoveTaskPayload{
			BoardID:          boardID,
			TaskID:           taskID,
			TargetColumnName: targetColumn,
		},
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Decode board from response data
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal board data: %w", err)
	}

	var board dto.BoardDTO
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}

	return &board, nil
}

// UpdateTask updates an existing task
func (c *Client) UpdateTask(ctx context.Context, boardID, taskID string, taskReq dto.UpdateTaskRequest) (*dto.TaskDTO, error) {
	req := &Request{
		Type: RequestUpdateTask,
		Payload: UpdateTaskPayload{
			BoardID:     boardID,
			TaskID:      taskID,
			TaskRequest: taskReq,
		},
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Decode task from response data
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task data: %w", err)
	}

	var task dto.TaskDTO
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// DeleteTask deletes a task
func (c *Client) DeleteTask(ctx context.Context, boardID, taskID string) error {
	req := &Request{
		Type: RequestDeleteTask,
		Payload: DeleteTaskPayload{
			BoardID: boardID,
			TaskID:  taskID,
		},
	}

	_, err := c.sendRequest(req)
	return err
}

// CreateColumn creates a new column
func (c *Client) CreateColumn(ctx context.Context, boardID string, columnReq dto.CreateColumnRequest) (*dto.BoardDTO, error) {
	req := &Request{
		Type: RequestAddColumn,
		Payload: AddColumnPayload{
			BoardID:       boardID,
			ColumnRequest: columnReq,
		},
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Decode board from response data
	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal board data: %w", err)
	}

	var board dto.BoardDTO
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}

	return &board, nil
}

// DeleteColumn deletes a column
func (c *Client) DeleteColumn(ctx context.Context, boardID, columnName string) error {
	req := &Request{
		Type: RequestDeleteColumn,
		Payload: DeleteColumnPayload{
			BoardID:    boardID,
			ColumnName: columnName,
		},
	}

	_, err := c.sendRequest(req)
	return err
}

// IsHealthy checks if the daemon is healthy
func (c *Client) IsHealthy() bool {
	socketPath := GetSocketPath(c.config)

	// Check if socket exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return false
	}

	// Try to connect
	conn, err := net.DialTimeout("unix", socketPath, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()

	return true
}

// Subscribe subscribes to real-time updates for a board
func (c *Client) Subscribe(boardID string) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	if c.isSubscribed {
		return fmt.Errorf("already subscribed")
	}

	socketPath := GetSocketPath(c.config)

	// Create a separate connection for subscription
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		// Try to start daemon if not running
		if err := c.startDaemon(); err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}

		// Retry connection
		time.Sleep(500 * time.Millisecond)
		conn, err = net.Dial("unix", socketPath)
		if err != nil {
			return fmt.Errorf("failed to connect to daemon: %w", err)
		}
	}

	c.subConn = conn

	// Send subscribe request
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	req := &Request{
		Type: RequestSubscribe,
		Payload: SubscribePayload{
			BoardID: boardID,
		},
	}

	if err := encoder.Encode(req); err != nil {
		conn.Close()
		c.subConn = nil
		return fmt.Errorf("failed to send subscribe request: %w", err)
	}

	// Read subscription response
	var resp Response
	if err := decoder.Decode(&resp); err != nil {
		conn.Close()
		c.subConn = nil
		return fmt.Errorf("failed to read subscribe response: %w", err)
	}

	if !resp.Success {
		conn.Close()
		c.subConn = nil
		return fmt.Errorf("subscription failed: %s", resp.Error)
	}

	c.isSubscribed = true

	// Start listening for notifications
	go c.listenForNotifications(decoder)

	return nil
}

// listenForNotifications listens for notifications from the daemon
func (c *Client) listenForNotifications(decoder *json.Decoder) {
	for {
		select {
		case <-c.stopChan:
			return
		default:
			var notif Notification
			if err := decoder.Decode(&notif); err != nil {
				// Connection error, stop listening
				c.subMu.Lock()
				c.isSubscribed = false
				c.subMu.Unlock()
				return
			}

			// Send notification to channel
			select {
			case c.notifChan <- &notif:
			default:
				// Channel full, skip
			}
		}
	}
}

// Unsubscribe unsubscribes from real-time updates
func (c *Client) Unsubscribe() error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	if !c.isSubscribed {
		return nil
	}

	// Signal to stop listening
	close(c.stopChan)
	c.stopChan = make(chan struct{})

	// Close subscription connection
	if c.subConn != nil {
		c.subConn.Close()
		c.subConn = nil
	}

	c.isSubscribed = false
	return nil
}

// Notifications returns the notification channel
func (c *Client) Notifications() <-chan *Notification {
	return c.notifChan
}
