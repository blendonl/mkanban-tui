package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type GoogleCalendarClient struct {
	service     *calendar.Service
	config      *oauth2.Config
	tokenPath   string
	calendarID  string
}

type CalendarEvent struct {
	ID          string
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
	Location    string
	Attendees   []string
	MeetingLink string
	IsAllDay    bool
	Recurring   bool
	RecurringID string
}

func NewGoogleCalendarClient(credentialsPath, tokenPath, calendarID string) (*GoogleCalendarClient, error) {
	credentials, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	config, err := google.ConfigFromJSON(credentials, calendar.CalendarReadonlyScope, calendar.CalendarEventsScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	if calendarID == "" {
		calendarID = "primary"
	}

	return &GoogleCalendarClient{
		config:     config,
		tokenPath:  tokenPath,
		calendarID: calendarID,
	}, nil
}

func (c *GoogleCalendarClient) IsAuthenticated() bool {
	_, err := c.loadToken()
	return err == nil
}

func (c *GoogleCalendarClient) GetAuthURL() string {
	return c.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

func (c *GoogleCalendarClient) ExchangeToken(ctx context.Context, code string) error {
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}

	return c.saveToken(token)
}

func (c *GoogleCalendarClient) Connect(ctx context.Context) error {
	token, err := c.loadToken()
	if err != nil {
		return fmt.Errorf("not authenticated: %w", err)
	}

	client := c.config.Client(ctx, token)
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create calendar service: %w", err)
	}

	c.service = service
	return nil
}

func (c *GoogleCalendarClient) GetEvents(ctx context.Context, start, end time.Time) ([]CalendarEvent, error) {
	if c.service == nil {
		if err := c.Connect(ctx); err != nil {
			return nil, err
		}
	}

	events, err := c.service.Events.List(c.calendarID).
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	result := make([]CalendarEvent, 0, len(events.Items))
	for _, item := range events.Items {
		event := c.parseEvent(item)
		result = append(result, event)
	}

	return result, nil
}

func (c *GoogleCalendarClient) GetTodayEvents(ctx context.Context) ([]CalendarEvent, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, 1)
	return c.GetEvents(ctx, start, end)
}

func (c *GoogleCalendarClient) GetWeekEvents(ctx context.Context) ([]CalendarEvent, error) {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, 7)
	return c.GetEvents(ctx, start, end)
}

func (c *GoogleCalendarClient) CreateEvent(ctx context.Context, event CalendarEvent) (*CalendarEvent, error) {
	if c.service == nil {
		if err := c.Connect(ctx); err != nil {
			return nil, err
		}
	}

	gEvent := &calendar.Event{
		Summary:     event.Title,
		Description: event.Description,
		Location:    event.Location,
	}

	if event.IsAllDay {
		gEvent.Start = &calendar.EventDateTime{Date: event.StartTime.Format("2006-01-02")}
		gEvent.End = &calendar.EventDateTime{Date: event.EndTime.Format("2006-01-02")}
	} else {
		gEvent.Start = &calendar.EventDateTime{DateTime: event.StartTime.Format(time.RFC3339)}
		gEvent.End = &calendar.EventDateTime{DateTime: event.EndTime.Format(time.RFC3339)}
	}

	if len(event.Attendees) > 0 {
		attendees := make([]*calendar.EventAttendee, len(event.Attendees))
		for i, email := range event.Attendees {
			attendees[i] = &calendar.EventAttendee{Email: email}
		}
		gEvent.Attendees = attendees
	}

	created, err := c.service.Events.Insert(c.calendarID, gEvent).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	result := c.parseEvent(created)
	return &result, nil
}

func (c *GoogleCalendarClient) UpdateEvent(ctx context.Context, event CalendarEvent) error {
	if c.service == nil {
		if err := c.Connect(ctx); err != nil {
			return err
		}
	}

	gEvent := &calendar.Event{
		Summary:     event.Title,
		Description: event.Description,
		Location:    event.Location,
	}

	if event.IsAllDay {
		gEvent.Start = &calendar.EventDateTime{Date: event.StartTime.Format("2006-01-02")}
		gEvent.End = &calendar.EventDateTime{Date: event.EndTime.Format("2006-01-02")}
	} else {
		gEvent.Start = &calendar.EventDateTime{DateTime: event.StartTime.Format(time.RFC3339)}
		gEvent.End = &calendar.EventDateTime{DateTime: event.EndTime.Format(time.RFC3339)}
	}

	_, err := c.service.Events.Update(c.calendarID, event.ID, gEvent).Do()
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

func (c *GoogleCalendarClient) DeleteEvent(ctx context.Context, eventID string) error {
	if c.service == nil {
		if err := c.Connect(ctx); err != nil {
			return err
		}
	}

	if err := c.service.Events.Delete(c.calendarID, eventID).Do(); err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

func (c *GoogleCalendarClient) parseEvent(item *calendar.Event) CalendarEvent {
	event := CalendarEvent{
		ID:          item.Id,
		Title:       item.Summary,
		Description: item.Description,
		Location:    item.Location,
		Recurring:   item.RecurringEventId != "",
		RecurringID: item.RecurringEventId,
	}

	if item.HangoutLink != "" {
		event.MeetingLink = item.HangoutLink
	}

	if item.ConferenceData != nil && len(item.ConferenceData.EntryPoints) > 0 {
		for _, ep := range item.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" {
				event.MeetingLink = ep.Uri
				break
			}
		}
	}

	if item.Attendees != nil {
		event.Attendees = make([]string, len(item.Attendees))
		for i, a := range item.Attendees {
			event.Attendees[i] = a.Email
		}
	}

	if item.Start.Date != "" {
		event.IsAllDay = true
		event.StartTime, _ = time.Parse("2006-01-02", item.Start.Date)
		event.EndTime, _ = time.Parse("2006-01-02", item.End.Date)
	} else {
		event.StartTime, _ = time.Parse(time.RFC3339, item.Start.DateTime)
		event.EndTime, _ = time.Parse(time.RFC3339, item.End.DateTime)
	}

	return event
}

func (c *GoogleCalendarClient) loadToken() (*oauth2.Token, error) {
	f, err := os.Open(c.tokenPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	token := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, err
	}

	return token, nil
}

func (c *GoogleCalendarClient) saveToken(token *oauth2.Token) error {
	dir := filepath.Dir(c.tokenPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	f, err := os.OpenFile(c.tokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}

type OAuthCallbackServer struct {
	server   *http.Server
	codeChan chan string
	errChan  chan error
}

func NewOAuthCallbackServer(port int) *OAuthCallbackServer {
	s := &OAuthCallbackServer{
		codeChan: make(chan string, 1),
		errChan:  make(chan error, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return s
}

func (s *OAuthCallbackServer) Start() error {
	go func() {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			s.errChan <- err
		}
	}()
	return nil
}

func (s *OAuthCallbackServer) WaitForCode(timeout time.Duration) (string, error) {
	select {
	case code := <-s.codeChan:
		return code, nil
	case err := <-s.errChan:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout waiting for OAuth callback")
	}
}

func (s *OAuthCallbackServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *OAuthCallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		s.errChan <- fmt.Errorf("missing code parameter")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authentication Complete</title></head>
<body style="font-family: system-ui; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0;">
<div style="text-align: center;">
<h1>✓ Authentication Successful</h1>
<p>You can close this window and return to the terminal.</p>
</div>
</body>
</html>`)

	s.codeChan <- code
}
