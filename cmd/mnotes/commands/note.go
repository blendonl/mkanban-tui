package commands

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"mkanban/internal/domain/entity"
)

var newCmd = &cobra.Command{
	Use:   "new [title]",
	Short: "Create a new note",
	Long: `Create a new note with the given title.

Opens your default editor to write the content.

Examples:
  # Create a general note
  mnotes new "API Design Ideas"

  # Create a meeting note
  mnotes new "Sprint Planning" --type meeting

  # Create a note with tags
  mnotes new "Bug Investigation" --tag urgent --tag backend`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		title := args[0]

		projectID, _ := cmd.Flags().GetString("project")
		noteTypeStr, _ := cmd.Flags().GetString("type")
		tags, _ := cmd.Flags().GetStringSlice("tag")

		noteType := entity.NoteType(noteTypeStr)
		if !noteType.IsValid() {
			noteType = entity.NoteTypeGeneral
		}

		id := uuid.New().String()
		note, err := entity.NewNote(id, title, noteType)
		if err != nil {
			return err
		}

		if projectID != "" {
			note.SetProjectID(projectID)
		}

		for _, tag := range tags {
			note.AddTag(tag)
		}

		content, err := openEditor("")
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}

		note.SetContent(content)

		if err := container.NoteRepo.Save(ctx, note); err != nil {
			return fmt.Errorf("failed to save note: %w", err)
		}

		fmt.Printf("Created note: %s\n", note.ID()[:8])
		return nil
	},
}

var journalCmd = &cobra.Command{
	Use:   "journal",
	Short: "Create or open today's journal entry",
	Long: `Create or open today's journal entry.

If a journal entry already exists for today, it will be opened for editing.
Otherwise, a new journal entry will be created.

Examples:
  # Create/open today's journal
  mnotes journal

  # Create journal for specific project
  mnotes journal --project my-project`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		projectID, _ := cmd.Flags().GetString("project")

		today := time.Now()
		var notes []*entity.Note
		var err error

		if projectID != "" {
			notes, err = container.NoteRepo.FindByDate(ctx, projectID, today)
		} else {
			notes, err = container.NoteRepo.FindGlobalByDate(ctx, today)
		}

		if err != nil {
			return err
		}

		var journalNote *entity.Note
		for _, note := range notes {
			if note.IsJournal() {
				journalNote = note
				break
			}
		}

		if journalNote == nil {
			id := uuid.New().String()
			title := fmt.Sprintf("Journal - %s", today.Format("2006-01-02"))
			journalNote, err = entity.NewNote(id, title, entity.NoteTypeJournal)
			if err != nil {
				return err
			}
			if projectID != "" {
				journalNote.SetProjectID(projectID)
			}
		}

		content, err := openEditor(journalNote.Content())
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}

		journalNote.SetContent(content)

		if err := container.NoteRepo.Save(ctx, journalNote); err != nil {
			return fmt.Errorf("failed to save journal: %w", err)
		}

		fmt.Printf("Saved journal entry: %s\n", journalNote.ID()[:8])
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List notes",
	Long: `List notes with optional filtering.

Examples:
  # List all notes
  mnotes list

  # List today's notes
  mnotes list --today

  # List notes by type
  mnotes list --type meeting

  # List notes with tag
  mnotes list --tag important`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		projectID, _ := cmd.Flags().GetString("project")
		today, _ := cmd.Flags().GetBool("today")
		noteTypeStr, _ := cmd.Flags().GetString("type")
		tag, _ := cmd.Flags().GetString("tag")

		var notes []*entity.Note
		var err error

		if projectID != "" {
			if tag != "" {
				notes, err = container.NoteRepo.FindByTag(ctx, projectID, tag)
			} else if noteTypeStr != "" {
				notes, err = container.NoteRepo.FindByType(ctx, projectID, entity.NoteType(noteTypeStr))
			} else if today {
				notes, err = container.NoteRepo.FindByDate(ctx, projectID, time.Now())
			} else {
				notes, err = container.NoteRepo.FindByProject(ctx, projectID)
			}
		} else {
			if today {
				notes, err = container.NoteRepo.FindGlobalByDate(ctx, time.Now())
			} else {
				notes, err = container.NoteRepo.FindGlobal(ctx)
			}
		}

		if err != nil {
			return err
		}

		if len(notes) == 0 {
			fmt.Println("No notes found")
			return nil
		}

		fmt.Println("Notes:")
		fmt.Println("─────────────────────────────────────────────────")
		for _, note := range notes {
			fmt.Printf("  [%s] %s\n", note.ID()[:8], note.Title())
			fmt.Printf("       Type: %s | Date: %s\n", note.NoteType(), note.Date().Format("2006-01-02"))
			if len(note.Tags()) > 0 {
				fmt.Printf("       Tags: %v\n", note.Tags())
			}
			fmt.Println()
		}

		return nil
	},
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search notes",
	Long: `Search notes by content or title.

Examples:
  # Search all notes
  mnotes search "api design"

  # Search in specific project
  mnotes search "bug fix" --project backend`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		query := args[0]
		projectID, _ := cmd.Flags().GetString("project")

		if projectID == "" {
			fmt.Println("Note: Searching global notes. Use --project for project-specific search.")
		}

		notes, err := container.NoteRepo.Search(ctx, projectID, query)
		if err != nil {
			return err
		}

		if len(notes) == 0 {
			fmt.Println("No notes found matching query")
			return nil
		}

		fmt.Printf("Found %d notes:\n", len(notes))
		fmt.Println("─────────────────────────────────────────────────")
		for _, note := range notes {
			fmt.Printf("  [%s] %s\n", note.ID()[:8], note.Title())
			fmt.Printf("       Date: %s | Type: %s\n", note.Date().Format("2006-01-02"), note.NoteType())
			fmt.Println()
		}

		return nil
	},
}

var viewCmd = &cobra.Command{
	Use:   "view [note-id]",
	Short: "View a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		noteID := args[0]

		note, err := container.NoteRepo.FindByID(ctx, noteID)
		if err != nil {
			return fmt.Errorf("note not found: %w", err)
		}

		fmt.Printf("Title: %s\n", note.Title())
		fmt.Printf("Type: %s\n", note.NoteType())
		fmt.Printf("Date: %s\n", note.Date().Format("2006-01-02 15:04"))
		if len(note.Tags()) > 0 {
			fmt.Printf("Tags: %v\n", note.Tags())
		}
		fmt.Println("─────────────────────────────────────────────────")
		fmt.Println(note.Content())

		return nil
	},
}

var editCmd = &cobra.Command{
	Use:   "edit [note-id]",
	Short: "Edit a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		noteID := args[0]

		note, err := container.NoteRepo.FindByID(ctx, noteID)
		if err != nil {
			return fmt.Errorf("note not found: %w", err)
		}

		content, err := openEditor(note.Content())
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}

		note.SetContent(content)

		if err := container.NoteRepo.Save(ctx, note); err != nil {
			return fmt.Errorf("failed to save note: %w", err)
		}

		fmt.Printf("Updated note: %s\n", note.ID()[:8])
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete [note-id]",
	Short: "Delete a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := getContext()
		noteID := args[0]

		if err := container.NoteRepo.Delete(ctx, noteID); err != nil {
			return fmt.Errorf("failed to delete note: %w", err)
		}

		fmt.Printf("Deleted note: %s\n", noteID)
		return nil
	},
}

func openEditor(content string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	tmpfile, err := os.CreateTemp("", "mnotes-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpfile.Name())

	if content != "" {
		if _, err := tmpfile.WriteString(content); err != nil {
			return "", err
		}
	}
	tmpfile.Close()

	cmd := exec.Command(editor, tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func init() {
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(journalCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(viewCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(deleteCmd)

	newCmd.Flags().StringP("type", "t", "general", "Note type (general, journal, meeting, standup, retrospective)")
	newCmd.Flags().StringSliceP("tag", "", nil, "Add tags to the note")

	listCmd.Flags().Bool("today", false, "Show only today's notes")
	listCmd.Flags().StringP("type", "t", "", "Filter by note type")
	listCmd.Flags().String("tag", "", "Filter by tag")
}
