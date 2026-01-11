package commands

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"mkanban/internal/application/dto"
	"mkanban/internal/daemon"
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's agenda",
	Long: `Display all scheduled tasks and meetings for today.

Shows:
- Scheduled tasks with time blocks
- Meetings
- All-day tasks
- Overdue tasks

Examples:
  # Show today's agenda
  magenda today

  # Show for specific project
  magenda today --project backend`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		projectID, _ := cmd.Flags().GetString("project")

		boardList, err := container.ListBoardsUseCase.Execute(ctx)
		if err != nil {
			return err
		}

		today := time.Now()
		todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
		todayEnd := todayStart.AddDate(0, 0, 1)

		fmt.Printf("📅 Agenda for %s\n", today.Format("Monday, January 2, 2006"))
		fmt.Println("═══════════════════════════════════════════════════")

		var scheduledTasks []dto.TaskDTO
		var meetings []dto.TaskDTO
		var allDayTasks []dto.TaskDTO

		for _, boardInfo := range boardList {
			board, err := container.GetBoardUseCase.Execute(ctx, boardInfo.ID)
			if err != nil {
				continue
			}

			if projectID != "" && board.ProjectID != projectID {
				continue
			}

			for _, column := range board.Columns {
				for _, task := range column.Tasks {
					if task.ScheduledDate != nil {
						scheduledDate := *task.ScheduledDate
						if !scheduledDate.Before(todayStart) && scheduledDate.Before(todayEnd) {
							if task.TaskType == "meeting" {
								meetings = append(meetings, task)
							} else if task.ScheduledTime != nil {
								scheduledTasks = append(scheduledTasks, task)
							} else {
								allDayTasks = append(allDayTasks, task)
							}
						}
					}
				}
			}
		}

		sort.Slice(meetings, func(i, j int) bool {
			if meetings[i].ScheduledTime == nil || meetings[j].ScheduledTime == nil {
				return false
			}
			return meetings[i].ScheduledTime.Before(*meetings[j].ScheduledTime)
		})

		sort.Slice(scheduledTasks, func(i, j int) bool {
			if scheduledTasks[i].ScheduledTime == nil || scheduledTasks[j].ScheduledTime == nil {
				return false
			}
			return scheduledTasks[i].ScheduledTime.Before(*scheduledTasks[j].ScheduledTime)
		})

		if len(meetings) > 0 {
			fmt.Println("\n🗓️  Meetings")
			fmt.Println("───────────────────────────────────────────────────")
			for _, m := range meetings {
				timeStr := "All day"
				if m.ScheduledTime != nil {
					timeStr = m.ScheduledTime.Format("15:04")
				}
				durationStr := ""
				if m.TimeBlock != nil {
					durationStr = fmt.Sprintf(" (%s)", formatDuration(*m.TimeBlock))
				}
				fmt.Printf("  %s %s%s\n", timeStr, m.Title, durationStr)
				fmt.Printf("       [%s]\n", m.ID)
			}
		}

		if len(scheduledTasks) > 0 {
			fmt.Println("\n⏰ Scheduled Tasks")
			fmt.Println("───────────────────────────────────────────────────")
			for _, t := range scheduledTasks {
				timeStr := t.ScheduledTime.Format("15:04")
				durationStr := ""
				if t.TimeBlock != nil {
					durationStr = fmt.Sprintf(" (%s)", formatDuration(*t.TimeBlock))
				}
				fmt.Printf("  %s %s%s\n", timeStr, t.Title, durationStr)
				fmt.Printf("       [%s] Priority: %s\n", t.ID, t.Priority)
			}
		}

		if len(allDayTasks) > 0 {
			fmt.Println("\n📋 All-Day Tasks")
			fmt.Println("───────────────────────────────────────────────────")
			for _, t := range allDayTasks {
				fmt.Printf("  • %s\n", t.Title)
				fmt.Printf("    [%s] Priority: %s\n", t.ID, t.Priority)
			}
		}

		if len(meetings) == 0 && len(scheduledTasks) == 0 && len(allDayTasks) == 0 {
			fmt.Println("\n  No scheduled items for today")
		}

		fmt.Println()
		return nil
	},
}

var weekCmd = &cobra.Command{
	Use:   "week",
	Short: "Show this week's agenda",
	Long: `Display scheduled tasks and meetings for the current week.

Examples:
  # Show this week's agenda
  magenda week

  # Show for specific project
  magenda week --project backend`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		projectID, _ := cmd.Flags().GetString("project")

		boardList, err := container.ListBoardsUseCase.Execute(ctx)
		if err != nil {
			return err
		}

		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		weekStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		weekEnd := weekStart.AddDate(0, 0, 7)

		fmt.Printf("📅 Week of %s - %s\n", weekStart.Format("Jan 2"), weekEnd.AddDate(0, 0, -1).Format("Jan 2, 2006"))
		fmt.Println("═══════════════════════════════════════════════════")

		dayTasks := make(map[string][]dto.TaskDTO)

		for _, boardInfo := range boardList {
			board, err := container.GetBoardUseCase.Execute(ctx, boardInfo.ID)
			if err != nil {
				continue
			}

			if projectID != "" && board.ProjectID != projectID {
				continue
			}

			for _, column := range board.Columns {
				for _, task := range column.Tasks {
					if task.ScheduledDate != nil {
						scheduledDate := *task.ScheduledDate
						if !scheduledDate.Before(weekStart) && scheduledDate.Before(weekEnd) {
							dayKey := scheduledDate.Format("2006-01-02")
							dayTasks[dayKey] = append(dayTasks[dayKey], task)
						}
					}
				}
			}
		}

		for i := 0; i < 7; i++ {
			day := weekStart.AddDate(0, 0, i)
			dayKey := day.Format("2006-01-02")
			tasks := dayTasks[dayKey]

			isToday := day.Format("2006-01-02") == now.Format("2006-01-02")
			marker := " "
			if isToday {
				marker = "→"
			}

			fmt.Printf("\n%s %s\n", marker, day.Format("Monday, Jan 2"))
			fmt.Println("  ─────────────────────────────────────────────")

			if len(tasks) == 0 {
				fmt.Println("    (no scheduled items)")
				continue
			}

			sort.Slice(tasks, func(i, j int) bool {
				if tasks[i].ScheduledTime == nil || tasks[j].ScheduledTime == nil {
					return false
				}
				return tasks[i].ScheduledTime.Before(*tasks[j].ScheduledTime)
			})

			for _, t := range tasks {
				timeStr := "     "
				if t.ScheduledTime != nil {
					timeStr = t.ScheduledTime.Format("15:04")
				}
				icon := "•"
				if t.TaskType == "meeting" {
					icon = "🗓️"
				}
				fmt.Printf("    %s %s %s\n", timeStr, icon, t.Title)
			}
		}

		fmt.Println()
		return nil
	},
}

var scheduleCmd = &cobra.Command{
	Use:   "schedule [task-id]",
	Short: "Schedule a task",
	Long: `Schedule a task for a specific date and time.

Examples:
  # Schedule for a date
  magenda schedule TASK-123 --date 2025-01-15

  # Schedule with time
  magenda schedule TASK-123 --date 2025-01-15 --time 10:00

  # Schedule with duration
  magenda schedule TASK-123 --date 2025-01-15 --time 10:00 --duration 2h`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		dateStr, _ := cmd.Flags().GetString("date")
		timeStr, _ := cmd.Flags().GetString("time")
		durationStr, _ := cmd.Flags().GetString("duration")

		if dateStr == "" {
			return fmt.Errorf("--date is required")
		}

		_, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("invalid date format, use YYYY-MM-DD")
		}

		client := daemon.NewClient(container.Config)
		if err := client.Connect(); err != nil {
			return fmt.Errorf("failed to connect to daemon: %w", err)
		}
		defer client.Close()

		payload := daemon.ScheduleTaskPayload{
			TaskID: taskID,
			Date:   dateStr,
		}
		if timeStr != "" {
			payload.Time = &timeStr
		}
		if durationStr != "" {
			payload.Duration = &durationStr
		}

		resp, err := client.SendRequest(daemon.RequestScheduleTask, payload)
		if err != nil {
			return err
		}

		if !resp.Success {
			return fmt.Errorf("failed to schedule task: %s", resp.Error)
		}

		data := resp.Data.(map[string]interface{})
		fmt.Printf("Scheduled task %s for %s\n", data["id"], dateStr)
		if timeStr != "" {
			fmt.Printf("  Time: %s\n", timeStr)
		}
		if durationStr != "" {
			fmt.Printf("  Duration: %s\n", durationStr)
		}

		return nil
	},
}

var meetingCmd = &cobra.Command{
	Use:   "meeting [title]",
	Short: "Create a meeting",
	Long: `Create a new meeting task.

Examples:
  # Create a meeting
  magenda meeting "Sprint Planning" --date 2025-01-15 --time 14:00

  # Create with duration and attendees
  magenda meeting "Design Review" --date 2025-01-15 --time 10:00 --duration 1h --attendee john@example.com`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		title := args[0]
		dateStr, _ := cmd.Flags().GetString("date")
		timeStr, _ := cmd.Flags().GetString("time")
		durationStr, _ := cmd.Flags().GetString("duration")
		attendees, _ := cmd.Flags().GetStringSlice("attendee")
		location, _ := cmd.Flags().GetString("location")
		boardID, _ := cmd.Flags().GetString("board")

		if dateStr == "" {
			return fmt.Errorf("--date is required")
		}

		_, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("invalid date format, use YYYY-MM-DD")
		}

		client := daemon.NewClient(container.Config)
		if err := client.Connect(); err != nil {
			return fmt.Errorf("failed to connect to daemon: %w", err)
		}
		defer client.Close()

		if boardID == "" {
			boardID, err = client.GetActiveBoard(ctx)
			if err != nil || boardID == "" {
				boards, err := container.ListBoardsUseCase.Execute(ctx)
				if err != nil || len(boards) == 0 {
					return fmt.Errorf("no board available, use --board flag")
				}
				boardID = boards[0].ID
			}
		}

		payload := daemon.CreateMeetingPayload{
			BoardID:   boardID,
			Title:     title,
			Date:      dateStr,
			Attendees: attendees,
		}
		if timeStr != "" {
			payload.Time = &timeStr
		}
		if durationStr != "" {
			payload.Duration = &durationStr
		}
		if location != "" {
			payload.Location = &location
		}

		resp, err := client.SendRequest(daemon.RequestCreateMeeting, payload)
		if err != nil {
			return err
		}

		if !resp.Success {
			return fmt.Errorf("failed to create meeting: %s", resp.Error)
		}

		data := resp.Data.(map[string]interface{})
		fmt.Printf("Created meeting: %s [%s]\n", title, data["id"])
		fmt.Printf("  Date: %s\n", dateStr)
		if timeStr != "" {
			fmt.Printf("  Time: %s\n", timeStr)
		}
		if durationStr != "" {
			fmt.Printf("  Duration: %s\n", durationStr)
		}

		return nil
	},
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", minutes)
}

func init() {
	rootCmd.AddCommand(todayCmd)
	rootCmd.AddCommand(weekCmd)
	rootCmd.AddCommand(scheduleCmd)
	rootCmd.AddCommand(meetingCmd)

	scheduleCmd.Flags().StringP("date", "d", "", "Date (YYYY-MM-DD)")
	scheduleCmd.Flags().StringP("time", "t", "", "Time (HH:MM)")
	scheduleCmd.Flags().String("duration", "", "Duration (e.g., 1h, 30m)")

	meetingCmd.Flags().StringP("date", "d", "", "Date (YYYY-MM-DD)")
	meetingCmd.Flags().StringP("time", "t", "", "Time (HH:MM)")
	meetingCmd.Flags().String("duration", "1h", "Duration (e.g., 1h, 30m)")
	meetingCmd.Flags().StringSlice("attendee", nil, "Attendee email")
	meetingCmd.Flags().String("location", "", "Meeting location or URL")
	meetingCmd.Flags().StringP("board", "b", "", "Board ID")
}
