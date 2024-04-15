package services

import (
	"bytes"
	context2 "context"
	"errors"
	"github.com/17HIERARCH70/BashAPI/internal/domain/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/net/context"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type ICommandService interface {
	ProcessCommand(script string) (gin.H, error)
	FetchCommands() ([]models.Command, error)
	FetchCommandByID(id int) (models.Command, error)
	StopCommand(id int) error
	FetchQueueList() ([]models.Queue, error)
	ForceStartCommand(id int) (gin.H, error)
	StopAllRunningCommands() error
}

var _ ICommandService = &CommandService{}

type CommandService struct {
	DB            *pgxpool.Pool
	Logger        *slog.Logger
	MaxConcurrent int
}

func NewCommandService(db *pgxpool.Pool, logger *slog.Logger, maxConcurrent int) *CommandService {
	return &CommandService{
		DB:            db,
		Logger:        logger,
		MaxConcurrent: maxConcurrent,
	}
}

var ErrNotFound = errors.New("command not found")

// ProcessCommand manages the creation and execution of a sudo command.
func (s *CommandService) ProcessCommand(script string) (gin.H, error) {
	if s.manageQueue() {
		id, err := s.createCommandQueueRecord(script)
		if err != nil {
			return nil, err
		}
		return gin.H{"message": "Command is being queued", "id": id}, nil
	} else {
		id, err := s.createCommandRecord(script, "running")
		if err != nil {
			return nil, err
		}
		go s.executeCommand(id, script)
		return gin.H{"message": "Command is being executed", "id": id}, nil
	}
}

// FetchCommands retrieves a list of all commands.
func (s *CommandService) FetchCommands() ([]models.Command, error) {
	var commands []models.Command
	rows, err := s.DB.Query(context.Background(), "SELECT id, script, status, pid, output, created_at, updated_at FROM commands.commands ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cmd models.Command
		if err := rows.Scan(&cmd.ID, &cmd.Script, &cmd.Status, &cmd.PID, &cmd.Output, &cmd.CreatedAt, &cmd.UpdatedAt); err != nil {
			s.Logger.Error("Error scanning command", "error", err)
			continue
		}
		commands = append(commands, cmd)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return commands, nil
}

// FetchCommandByID retrieves a command by its ID.
func (s *CommandService) FetchCommandByID(id int) (models.Command, error) {
	var command models.Command
	err := s.DB.QueryRow(context.Background(),
		"SELECT id, script, status, pid, output, created_at, updated_at FROM commands.commands WHERE id = $1",
		id).Scan(&command.ID, &command.Script, &command.Status, &command.PID, &command.Output, &command.CreatedAt, &command.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Command{}, ErrNotFound
		}
		return models.Command{}, err
	}
	return command, nil
}

// StopCommand stops a command by its ID.
func (s *CommandService) StopCommand(id int) error {
	var pid *int // Use *int to properly handle NULL values
	err := s.DB.QueryRow(context.Background(), "SELECT pid FROM commands.commands WHERE id = $1", id).Scan(&pid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound // No command with the given ID was found
		}
		return err // Handle other errors (e.g., SQL errors)
	}

	if pid == nil {
		return errors.New("no PID found for the command; it may not have been started or already stopped")
	}

	process, err := os.FindProcess(*pid)
	if err != nil {
		return err // Failed to find the process, handle error
	}

	if err := process.Signal(syscall.SIGINT); err != nil {
		return err // Error sending the interrupt signal
	}

	return s.updateCommandStatusManually(id, "stopped")
}

// FetchQueueList retrieves all queue items ordered by QueueId.
func (s *CommandService) FetchQueueList() ([]models.Queue, error) {
	var queue []models.Queue
	rows, err := s.DB.Query(context.Background(), "SELECT queue_id, command_id, status FROM commands.queue ORDER BY queue_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var q models.Queue
		if err := rows.Scan(&q.QueueId, &q.CommandId, &q.Status); err != nil {
			continue // Optionally handle partial data or halt processing
		}
		queue = append(queue, q)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return queue, nil
}

// ForceStartCommand forcefully starts a command by its ID, ignoring queue constraints.
func (s *CommandService) ForceStartCommand(id int) (gin.H, error) {
	var script, currentStatus string
	err := s.DB.QueryRow(context.Background(), "SELECT script, status FROM commands.commands WHERE id = $1", id).Scan(&script, &currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if currentStatus == "running" || currentStatus == "completed" {
		return gin.H{"error": "Command is already " + currentStatus}, nil
	}

	// Обновляем статус команды на 'running' и удаляем из очереди
	tx, err := s.DB.Begin(context.Background())
	if err != nil {
		s.Logger.Error("Failed to begin transaction", "error", err)
		return nil, err
	}

	_, err = tx.Exec(context.Background(), "UPDATE commands.commands SET status = 'running' WHERE id = $1", id)
	if err != nil {
		_ = tx.Rollback(context.Background())
		s.Logger.Error("Failed to update command status", "error", err)
		return nil, err
	}

	_, err = tx.Exec(context.Background(), "DELETE FROM commands.queue WHERE command_id = $1", id)
	if err != nil {
		_ = tx.Rollback(context.Background())
		s.Logger.Error("Failed to delete command from queue", "commandID", id, "error", err)
		return nil, err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		s.Logger.Error("Failed to commit transaction", "error", err)
		return nil, err
	}

	go s.executeCommand(id, script)
	return gin.H{"message": "Command is being forcibly started", "id": id}, nil
}

// StopAllRunningCommands to stop all running commands
func (s *CommandService) StopAllRunningCommands() error {
	var commandIDs []int
	rows, err := s.DB.Query(context.Background(), "SELECT id FROM commands.commands WHERE status = 'running'")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			s.Logger.Error("Error scanning command", "error", err)
			continue
		}
		commandIDs = append(commandIDs, id)
	}

	for _, id := range commandIDs {
		if err := s.StopCommand(id); err != nil {
			s.Logger.Error("Failed to stop command", "commandID", id, "error", err)
		}
	}

	return nil
}

// manageQueue checking if count bigger than maxConcurrent, returning bool
func (s *CommandService) manageQueue() bool {
	count, err := s.getRunningCommandsCount()
	if err != nil {
		s.Logger.Error("Error getting running commands count", "error", err)
		return true
	}
	return count >= s.MaxConcurrent
}

// getRunningCommandsCount gets the count of running commands.
func (s *CommandService) getRunningCommandsCount() (int, error) {
	var count int
	err := s.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM commands.commands WHERE status = 'running'").Scan(&count)
	if err != nil {
		s.Logger.Error("Failed to get running commands count", "error", err)
		return 0, err
	}
	return count, nil
}

// createCommandQueueRecord creates a new record in the queue table for the given script.
func (s *CommandService) createCommandQueueRecord(script string) (int, error) {
	// Start a transaction
	tx, err := s.DB.Begin(context.Background())
	if err != nil {
		s.Logger.Error("Failed to start transaction", "error", err)
		return 0, err
	}
	defer func(tx pgx.Tx, ctx context2.Context) {
		_ = tx.Rollback(ctx)
	}(tx, context.Background())

	// Create the command record and get the ID
	var commandID int
	err = tx.QueryRow(context.Background(),
		"INSERT INTO commands.commands (script, status) VALUES ($1, 'waiting') RETURNING id",
		script).Scan(&commandID)
	if err != nil {
		s.Logger.Error("Failed to create command record", "error", err)
		return 0, err
	}

	// Insert into queue table
	_, err = tx.Exec(context.Background(), "INSERT INTO commands.queue (command_id, status) VALUES ($1, 'waiting')", commandID)
	if err != nil {
		s.Logger.Error("Failed to enqueue command", "error", err)
		return 0, err
	}

	// Commit the transaction
	if err := tx.Commit(context.Background()); err != nil {
		s.Logger.Error("Failed to commit transaction", "error", err)
		return 0, err
	}

	// Launch the execution manager asynchronously
	go s.manageExecution(commandID)
	return commandID, nil
}

// manageExecution checks every 10 sec and manages the queue.
func (s *CommandService) manageExecution(commandID int) {
	for {
		time.Sleep(10 * time.Second) // Check every 10 seconds

		status := s.manageQueue()
		if status == false {
			// Remove command from the queue
			_, err := s.DB.Exec(context.Background(), "DELETE FROM commands.queue WHERE command_id = $1", commandID)
			if err != nil {
				s.Logger.Error("Failed to delete command from queue", "commandID", commandID, "error", err)
				continue
			}

			// Update the command status to running
			_, err = s.DB.Exec(context.Background(), "UPDATE commands.commands SET status = 'running' WHERE id = $1", commandID)
			if err != nil {
				s.Logger.Error("Failed to update command status to running", "commandID", commandID, "error", err)
				continue
			}

			// Retrieve the script for the command
			var script string
			err = s.DB.QueryRow(context.Background(), "SELECT script FROM commands.commands WHERE id = $1", commandID).Scan(&script)
			if err != nil {
				s.Logger.Error("Failed to retrieve script for execution", "commandID", commandID, "error", err)
				continue
			}

			// Execute the command
			go s.executeCommand(commandID, script)
			break
		}
	}
}

// createCommandRecordWithStatus inserts a command record with the given script and status.
func (s *CommandService) createCommandRecordWithStatus(script string, status string) (int, error) {
	var commandID int
	err := s.DB.QueryRow(context.Background(),
		"INSERT INTO commands.commands (script, status) VALUES ($1, $2) RETURNING id",
		script, status).Scan(&commandID)

	if err != nil {
		s.Logger.Error("Failed to create command record", "error", err)
		return 0, err
	}
	return commandID, nil
}

// executeCommand main func to execute bash scripts
func (s *CommandService) executeCommand(commandID int, script string) {
	done := make(chan struct{})     // Channel to signal when updating output is done
	finished := make(chan struct{}) // Channel to signal when command is finished

	// Asynchronous bash script execution
	cmd := exec.Command("bash", "-c", script)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Start()
	if err != nil {
		s.Logger.Error("Failed to start command", "error", err)
		s.updateCommandStatus(commandID, "error", output.String())
		return
	}

	// Save command PID to the database
	pid := cmd.Process.Pid
	_, err = s.DB.Exec(context.Background(), "UPDATE commands.commands SET pid = $1 WHERE id = $2", pid, commandID)
	if err != nil {
		s.Logger.Error("Failed to save command PID", "error", err)
		return
	}

	// Asynchronously update the output of the command in the database
	go func() {
		defer close(done) // Close the channel when the function exits

		for {
			// There could be possible to recheck every 10 bytes.
			time.Sleep(3 * time.Second) // wait before each update
			s.updateCommandOutput(commandID, output.String())

			// Check if the command has completed
			select {
			case <-finished:
				return
			default:
				// If the command is still running, continue updating output
			}
		}
	}()

	// Wait for command to complete
	err = cmd.Wait()
	if err != nil {
		s.Logger.Error("Command execution failed", "error", err)
		s.updateCommandStatus(commandID, "error", output.String())
	} else {
		s.updateCommandStatus(commandID, "completed", output.String())
	}

	// Signal that the command is finished
	close(finished)

	// Wait for the updating output to complete
	<-done
}

// createCommandRecord starting logging in db
func (s *CommandService) createCommandRecord(script, status string) (int, error) {
	var commandID int
	err := s.DB.QueryRow(context.Background(),
		"INSERT INTO commands.commands (script, status) VALUES ($1, $2) RETURNING id",
		script, status).Scan(&commandID)

	if err != nil {
		return 0, err
	}
	return commandID, nil
}

// updateCommandOutput updating command output in database
func (s *CommandService) updateCommandOutput(commandID int, output string) {
	_, err := s.DB.Exec(context.Background(),
		"UPDATE commands.commands SET output = $1 WHERE id = $2",
		output, commandID)
	if err != nil {
		s.Logger.Error("Failed to update command output", "error", err)
	}
}

// updateCommandStatus updating status code of script in db
func (s *CommandService) updateCommandStatus(commandID int, status string, output string) {
	_, err := s.DB.Exec(context.Background(),
		"UPDATE commands.commands SET status = $1, output = $2 WHERE id = $3",
		status, output, commandID)
	if err != nil {
		s.Logger.Error("Failed to update command status", "error", err)
	}
}

// updateCommandStatusManually manually updating the status of the command in the database
func (s *CommandService) updateCommandStatusManually(id int, status string) error {
	_, err := s.DB.Exec(context.Background(),
		"UPDATE commands.commands SET status = $1 WHERE id = $2",
		status, id)
	if err != nil {
		return err
	}
	return nil
}
