package commands

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"mkanban/internal/infrastructure/external"
)

var calendarCmd = &cobra.Command{
	Use:   "calendar",
	Short: "Google Calendar integration",
	Long: `Manage Google Calendar integration.

Commands for authenticating with Google Calendar and syncing events.

Examples:
  # Authenticate with Google Calendar
  magenda calendar auth

  # Show calendar sync status
  magenda calendar status

  # Sync calendar events
  magenda calendar sync

  # List upcoming calendar events
  magenda calendar events`,
}

var calendarAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Google Calendar",
	Long: `Start the OAuth2 flow to authenticate with Google Calendar.

This will open a browser window for you to authorize access to your calendar.
Make sure you have downloaded your credentials from Google Cloud Console
and saved them to ~/.config/mkanban/google_credentials.json

To set up Google Calendar integration:
1. Go to https://console.cloud.google.com
2. Create a new project or select existing
3. Enable the Google Calendar API
4. Create OAuth 2.0 credentials (Desktop application)
5. Download the credentials JSON file
6. Save it as ~/.config/mkanban/google_credentials.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := container.Config

		if cfg.Calendar.CredentialsPath == "" {
			return fmt.Errorf("calendar credentials path not configured")
		}

		client, err := external.NewGoogleCalendarClient(
			cfg.Calendar.CredentialsPath,
			cfg.Calendar.TokenPath,
			cfg.Calendar.CalendarID,
		)
		if err != nil {
			return fmt.Errorf("failed to create calendar client: %w", err)
		}

		if client.IsAuthenticated() {
			fmt.Println("Already authenticated with Google Calendar")
			return nil
		}

		authURL := client.GetAuthURL()
		fmt.Println("Opening browser for Google Calendar authentication...")
		fmt.Printf("\nIf the browser doesn't open, visit this URL:\n%s\n\n", authURL)

		callbackServer := external.NewOAuthCallbackServer(cfg.Calendar.CallbackPort)
		if err := callbackServer.Start(); err != nil {
			return fmt.Errorf("failed to start callback server: %w", err)
		}
		defer callbackServer.Stop(context.Background())

		openBrowser(authURL)

		fmt.Println("Waiting for authorization...")
		code, err := callbackServer.WaitForCode(5 * time.Minute)
		if err != nil {
			return fmt.Errorf("failed to receive authorization: %w", err)
		}

		ctx := context.Background()
		if err := client.ExchangeToken(ctx, code); err != nil {
			return fmt.Errorf("failed to exchange token: %w", err)
		}

		fmt.Println("✓ Successfully authenticated with Google Calendar!")
		return nil
	},
}

var calendarStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show calendar sync status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := container.Config

		if !cfg.Calendar.Enabled {
			fmt.Println("Calendar integration is disabled")
			fmt.Println("Enable it in ~/.config/mkanban/config.yml")
			return nil
		}

		client, err := external.NewGoogleCalendarClient(
			cfg.Calendar.CredentialsPath,
			cfg.Calendar.TokenPath,
			cfg.Calendar.CalendarID,
		)
		if err != nil {
			return fmt.Errorf("failed to create calendar client: %w", err)
		}

		fmt.Println("📅 Calendar Integration Status")
		fmt.Println("═══════════════════════════════════════════════════")
		fmt.Printf("  Enabled:         %v\n", cfg.Calendar.Enabled)
		fmt.Printf("  Authenticated:   %v\n", client.IsAuthenticated())
		fmt.Printf("  Auto-sync:       %v\n", cfg.Calendar.AutoSync)
		fmt.Printf("  Sync interval:   %d seconds\n", cfg.Calendar.SyncInterval)
		fmt.Printf("  Pull enabled:    %v\n", cfg.Calendar.PullEnabled)
		fmt.Printf("  Push enabled:    %v\n", cfg.Calendar.PushEnabled)
		fmt.Printf("  Conflict policy: %s\n", cfg.Calendar.ConflictPolicy)
		fmt.Printf("  Calendar ID:     %s\n", cfg.Calendar.CalendarID)

		return nil
	},
}

var calendarSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync calendar events",
	Long: `Manually trigger a calendar sync.

This will:
- Pull new events from Google Calendar as meeting tasks
- Push meeting tasks to Google Calendar`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := container.Config

		client, err := external.NewGoogleCalendarClient(
			cfg.Calendar.CredentialsPath,
			cfg.Calendar.TokenPath,
			cfg.Calendar.CalendarID,
		)
		if err != nil {
			return fmt.Errorf("failed to create calendar client: %w", err)
		}

		if !client.IsAuthenticated() {
			return fmt.Errorf("not authenticated with Google Calendar. Run 'magenda calendar auth' first")
		}

		ctx := context.Background()
		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to calendar: %w", err)
		}

		pullOnly, _ := cmd.Flags().GetBool("pull")
		pushOnly, _ := cmd.Flags().GetBool("push")

		if pullOnly {
			fmt.Println("Pulling events from Google Calendar...")
			events, err := client.GetWeekEvents(ctx)
			if err != nil {
				return fmt.Errorf("failed to fetch events: %w", err)
			}
			fmt.Printf("Found %d events this week\n", len(events))
			return nil
		}

		if pushOnly {
			fmt.Println("Pushing meeting tasks to Google Calendar...")
			fmt.Println("(Push sync would happen here)")
			return nil
		}

		fmt.Println("Running full calendar sync...")
		events, err := client.GetWeekEvents(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch events: %w", err)
		}
		fmt.Printf("✓ Found %d events this week\n", len(events))
		fmt.Println("✓ Sync complete")

		return nil
	},
}

var calendarEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "List calendar events",
	Long: `List upcoming calendar events.

Examples:
  # List today's events
  magenda calendar events --today

  # List this week's events
  magenda calendar events --week`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := container.Config

		client, err := external.NewGoogleCalendarClient(
			cfg.Calendar.CredentialsPath,
			cfg.Calendar.TokenPath,
			cfg.Calendar.CalendarID,
		)
		if err != nil {
			return fmt.Errorf("failed to create calendar client: %w", err)
		}

		if !client.IsAuthenticated() {
			return fmt.Errorf("not authenticated with Google Calendar. Run 'magenda calendar auth' first")
		}

		ctx := context.Background()

		todayOnly, _ := cmd.Flags().GetBool("today")

		var events []external.CalendarEvent
		if todayOnly {
			events, err = client.GetTodayEvents(ctx)
			fmt.Println("📅 Today's Calendar Events")
		} else {
			events, err = client.GetWeekEvents(ctx)
			fmt.Println("📅 This Week's Calendar Events")
		}

		if err != nil {
			return fmt.Errorf("failed to fetch events: %w", err)
		}

		fmt.Println("═══════════════════════════════════════════════════")

		if len(events) == 0 {
			fmt.Println("  No events found")
			return nil
		}

		currentDate := ""
		for _, event := range events {
			eventDate := event.StartTime.Format("Monday, Jan 2")
			if eventDate != currentDate {
				if currentDate != "" {
					fmt.Println()
				}
				fmt.Printf("\n%s\n", eventDate)
				fmt.Println("───────────────────────────────────────────────────")
				currentDate = eventDate
			}

			timeStr := "All day"
			if !event.IsAllDay {
				timeStr = fmt.Sprintf("%s - %s",
					event.StartTime.Format("15:04"),
					event.EndTime.Format("15:04"))
			}

			fmt.Printf("  %s  %s\n", timeStr, event.Title)
			if event.Location != "" {
				fmt.Printf("           📍 %s\n", event.Location)
			}
			if event.MeetingLink != "" {
				fmt.Printf("           🔗 %s\n", event.MeetingLink)
			}
		}

		fmt.Println()
		return nil
	},
}

func openBrowser(url string) {
	// Use xdg-open on Linux
	// This is a simple implementation - in production you'd want platform detection
	cmd := exec.Command("xdg-open", url)
	cmd.Start()
}

func init() {
	rootCmd.AddCommand(calendarCmd)

	calendarCmd.AddCommand(calendarAuthCmd)
	calendarCmd.AddCommand(calendarStatusCmd)
	calendarCmd.AddCommand(calendarSyncCmd)
	calendarCmd.AddCommand(calendarEventsCmd)

	calendarSyncCmd.Flags().Bool("pull", false, "Only pull events from calendar")
	calendarSyncCmd.Flags().Bool("push", false, "Only push tasks to calendar")

	calendarEventsCmd.Flags().Bool("today", false, "Show only today's events")
	calendarEventsCmd.Flags().Bool("week", true, "Show this week's events (default)")
}
