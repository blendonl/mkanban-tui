package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"

	"mkanban/internal/infrastructure/config"
)

// Client represents a daemon client
type Client struct {
	socketPath string
}

// NewClient creates a new daemon client
func NewClient(cfg *config.Config) *Client {
	socketPath := filepath.Join(cfg.Daemon.SocketDir, cfg.Daemon.SocketName)
	return &Client{
		socketPath: socketPath,
	}
}

// sendRequest sends a request to the daemon and returns the response
func (c *Client) sendRequest(req *Request) (*Response, error) {
	// Connect to daemon socket
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	// Send request
	if err := encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	var resp Response
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resp, nil
}

// GetActiveBoard gets the board ID for the active session
func (c *Client) GetActiveBoard() (string, error) {
	req := &Request{
		Type: RequestGetActiveBoard,
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", fmt.Errorf("daemon error: %s", resp.Error)
	}

	// Parse response data
	if resp.Data == nil {
		return "", nil
	}

	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	boardID, ok := dataMap["board_id"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected board_id format")
	}

	return boardID, nil
}

// Ping checks if the daemon is running and responding
func (c *Client) Ping() error {
	// Try to connect to the socket
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
