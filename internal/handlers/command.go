package handlers

import (
	"bytes"
	"errors"
	"github.com/17HIERARCH70/BashAPI/internal/domain/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/net/context"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// CommandHandlers Structure for organizing command handlers.
type CommandHandlers struct {
	DB            *pgxpool.Pool
	Logger        *slog.Logger
	MaxConcurrent int
	OSpas         string
}

// NewCommandHandlers creates an instance CommandHandlers.
func NewCommandHandlers(db *pgxpool.Pool, logger *slog.Logger, maxConcurrent int, OSpas string) *CommandHandlers {
	return &CommandHandlers{
		DB:            db,
		Logger:        logger,
		MaxConcurrent: maxConcurrent,
	}
}

// CreateSudoCommand handler to create a new sudo command
func (h *CommandHandlers) CreateSudoCommand(c *gin.Context) {
	var command struct {
		Script string `json:"script"`
	}

	if err := c.ShouldBindJSON(&command); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	if command.Script == "" {
		c.JSON(400, gin.H{"error": "Script is required"})
		return
	}

	if h.manageQueue() {
		commandID, err := h.createCommandQueueRecord(command.Script)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create command record"})
			return
		}
		c.JSON(202, gin.H{"message": "Command is being queued", "id": commandID})
	} else {
		commandID, err := h.createCommandRecord(command.Script)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create command record"})
			return
		}
		go h.executeCommand(commandID, command.Script)
		c.JSON(202, gin.H{"message": "Command is being executed", "id": commandID})
	}
}

// CreateCommand handler to create a new command.
func (h *CommandHandlers) CreateCommand(c *gin.Context) {
	var command struct {
		Script string `json:"script"`
	}

	if err := c.ShouldBindJSON(&command); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	if command.Script == "" {
		c.JSON(400, gin.H{"error": "Script is required"})
		return
	}

	if strings.Contains(command.Script, "sudo") {
		c.JSON(400, gin.H{"error": "Non-Sudo command cannot contain 'sudo'"})
		return
	}

	if h.manageQueue() {
		commandID, err := h.createCommandQueueRecord(command.Script)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create command record"})
			return
		}
		c.JSON(202, gin.H{"message": "Command is being queued", "id": commandID})
	} else {
		commandID, err := h.createCommandRecord(command.Script)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create command record"})
			return
		}
		go h.executeCommand(commandID, command.Script)
		c.JSON(202, gin.H{"message": "Command is being executed", "id": commandID})
	}
}

// GetCommandsList handler to retrieve the list of commands.
func (h *CommandHandlers) GetCommandsList(c *gin.Context) {
	var commands []models.Command

	rows, err := h.DB.Query(context.Background(), "SELECT id, script, status, pid, output, created_at, updated_at FROM commands.commands ORDER BY id")
	if err != nil {
		h.Logger.Error("Failed to query commands list", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch commands"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cmd models.Command
		err := rows.Scan(&cmd.ID, &cmd.Script, &cmd.Status, &cmd.PID, &cmd.Output, &cmd.CreatedAt, &cmd.UpdatedAt)
		if err != nil {
			h.Logger.Error("Failed to scan command", "error", err)
			continue // Skip invalid entries
		}
		commands = append(commands, cmd)
	}
	if err = rows.Err(); err != nil {
		h.Logger.Error("Error iterating over commands", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing commands list"})
		return
	}

	c.JSON(http.StatusOK, commands)
}

// GetCommandByID handler to retrieve the command by ID.
func (h *CommandHandlers) GetCommandByID(c *gin.Context) {
	commandIDParam := c.Param("id")
	commandID, err := strconv.Atoi(commandIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid command ID"})
		return
	}

	var command models.Command
	err = h.DB.QueryRow(context.Background(),
		"SELECT id, script, status, pid, output, created_at, updated_at FROM commands.commands WHERE id = $1",
		commandID).Scan(&command.ID, &command.Script, &command.Status, &command.PID, &command.Output, &command.CreatedAt, &command.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Command not found"})
		} else {
			h.Logger.Error("Failed to query command by ID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch command"})
		}
		return
	}

	c.JSON(http.StatusOK, command)
}

// StopCommand handler to stop the command.
func (h *CommandHandlers) StopCommand(c *gin.Context) {
	commandIDParam := c.Param("id")
	commandID, err := strconv.Atoi(commandIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid command ID"})
		return
	}

	var pid int
	err = h.DB.QueryRow(context.Background(), "SELECT pid FROM commands.commands WHERE id = $1", commandID).Scan(&pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not find command"})
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find process"})
		return
	}

	if err := process.Signal(syscall.SIGINT); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop command"})
		return
	}

	err = h.updateCommandStatusManually(commandID, "stopped")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Command not found"})
		} else {
			h.Logger.Error("Failed to stop command", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop command"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Command stopped successfully"})
}

// GetQueueList retrieves a list of all queue items ordered by QueueId
func (h *CommandHandlers) GetQueueList(c *gin.Context) {
	var queue []models.Queue
	rows, err := h.DB.Query(context.Background(), "SELECT queue_id, command_id, status FROM commands.queue ORDER BY queue_id")
	if err != nil {
		h.Logger.Error("Failed to fetch queue items", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve queue data"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var q models.Queue
		err := rows.Scan(&q.QueueId, &q.CommandId, &q.Status)
		if err != nil {
			h.Logger.Error("Failed to scan queue item", "error", err)
			continue // Optionally handle partial data or halt processing
		}
		queue = append(queue, q)
	}

	if err = rows.Err(); err != nil {
		h.Logger.Error("Error iterating over queue items", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing queue data"})
		return
	}

	c.JSON(http.StatusOK, queue)
}

// ForceStartCommand forcefully starts a command by its ID, ignoring queue constraints.
func (h *CommandHandlers) ForceStartCommand(c *gin.Context) {
	commandIDParam := c.Param("id")
	commandID, err := strconv.Atoi(commandIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid command ID"})
		return
	}

	// Retrieve the command details
	var script string
	var currentStatus string
	err = h.DB.QueryRow(context.Background(),
		"SELECT script, status FROM commands.commands WHERE id = $1", commandID).Scan(&script, &currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Command not found"})
		} else {
			h.Logger.Error("Failed to query command by ID", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch command details"})
		}
		return
	}

	// Check if the command is already running or completed
	if currentStatus == "running" || currentStatus == "completed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Command is already " + currentStatus})
		return
	}

	// Update the command status to 'running'
	_, err = h.DB.Exec(context.Background(),
		"UPDATE commands.commands SET status = 'running' WHERE id = $1", commandID)
	if err != nil {
		h.Logger.Error("Failed to update command status to running", "commandID", commandID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update command status"})
		return
	}

	// Execute the command
	go h.executeCommand(commandID, script)
	c.JSON(http.StatusOK, gin.H{"message": "Command is being forcibly started", "id": commandID})
}

// manageQueue checking if count bigger than maxConcurrent, returning bool
func (h *CommandHandlers) manageQueue() bool {
	count, err := h.getRunningCommandsCount()
	if err != nil {
		h.Logger.Error("Error getting running commands count", "error", err)
		return true // Assume queuing is necessary if there's an error fetching the count.
	}
	return count >= h.MaxConcurrent
}

// getRunningCommandsCount gets the count of running commands.
func (h *CommandHandlers) getRunningCommandsCount() (int, error) {
	var count int
	err := h.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM commands.commands WHERE status = 'running'").Scan(&count)
	if err != nil {
		h.Logger.Error("Failed to get running commands count", "error", err)
		return 0, err
	}
	return count, nil
}

// createCommandQueueRecord creates a new record in the queue table for the given script.
func (h *CommandHandlers) createCommandQueueRecord(script string) (int, error) {
	// Start a transaction
	tx, err := h.DB.Begin(context.Background())
	if err != nil {
		h.Logger.Error("Failed to start transaction", "error", err)
		return 0, err
	}
	defer tx.Rollback(context.Background())

	// Create the command record and get the ID
	var commandID int
	err = tx.QueryRow(context.Background(),
		"INSERT INTO commands.commands (script, status) VALUES ($1, 'waiting') RETURNING id",
		script).Scan(&commandID)
	if err != nil {
		h.Logger.Error("Failed to create command record", "error", err)
		return 0, err
	}

	// Insert into queue table
	_, err = tx.Exec(context.Background(), "INSERT INTO commands.queue (command_id, status) VALUES ($1, 'waiting')", commandID)
	if err != nil {
		h.Logger.Error("Failed to enqueue command", "error", err)
		return 0, err
	}

	// Commit the transaction
	if err := tx.Commit(context.Background()); err != nil {
		h.Logger.Error("Failed to commit transaction", "error", err)
		return 0, err
	}

	// Launch the execution manager asynchronously
	go h.manageExecution(commandID)
	return commandID, nil
}

// manageExecution checks every 10 sec and manages the queue.
func (h *CommandHandlers) manageExecution(commandID int) {
	for {
		time.Sleep(10 * time.Second) // Check every 10 seconds

		status := h.manageQueue()
		if status == false {
			// Remove command from the queue
			_, err := h.DB.Exec(context.Background(), "DELETE FROM commands.queue WHERE command_id = $1", commandID)
			if err != nil {
				h.Logger.Error("Failed to delete command from queue", "commandID", commandID, "error", err)
				continue
			}

			// Update the command status to running
			_, err = h.DB.Exec(context.Background(), "UPDATE commands.commands SET status = 'running' WHERE id = $1", commandID)
			if err != nil {
				h.Logger.Error("Failed to update command status to running", "commandID", commandID, "error", err)
				continue
			}

			// Retrieve the script for the command
			var script string
			err = h.DB.QueryRow(context.Background(), "SELECT script FROM commands.commands WHERE id = $1", commandID).Scan(&script)
			if err != nil {
				h.Logger.Error("Failed to retrieve script for execution", "commandID", commandID, "error", err)
				continue
			}

			// Execute the command
			go h.executeCommand(commandID, script)
			break
		}
	}
}

// createCommandRecordWithStatus inserts a command record with the given script and status.
func (h *CommandHandlers) createCommandRecordWithStatus(script string, status string) (int, error) {
	var commandID int
	err := h.DB.QueryRow(context.Background(),
		"INSERT INTO commands.commands (script, status) VALUES ($1, $2) RETURNING id",
		script, status).Scan(&commandID)

	if err != nil {
		h.Logger.Error("Failed to create command record", "error", err)
		return 0, err
	}
	return commandID, nil
}

// executeCommand main func to execute bash scripts
func (h *CommandHandlers) executeCommand(commandID int, script string) {
	done := make(chan struct{})     // Channel to signal when updating output is done
	finished := make(chan struct{}) // Channel to signal when command is finished

	// Asynchronous bash script execution
	cmd := exec.Command("bash", "-c", script)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Start()
	if err != nil {
		h.Logger.Error("Failed to start command", "error", err)
		h.updateCommandStatus(commandID, "error", output.String())
		return
	}

	// Save command PID to the database
	pid := cmd.Process.Pid
	_, err = h.DB.Exec(context.Background(), "UPDATE commands.commands SET pid = $1 WHERE id = $2", pid, commandID)
	if err != nil {
		h.Logger.Error("Failed to save command PID", "error", err)
		return
	}

	// Asynchronously update the output of the command in the database
	go func() {
		defer close(done) // Close the channel when the function exits

		for {
			time.Sleep(1 * time.Second) // wait before each update
			h.updateCommandOutput(commandID, output.String())

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
		h.Logger.Error("Command execution failed", "error", err)
		h.updateCommandStatus(commandID, "error", output.String())
	} else {
		h.updateCommandStatus(commandID, "completed", output.String())
	}

	// Signal that the command is finished
	close(finished)

	// Wait for the updating output to complete
	<-done
}

// createCommandRecord starting logging in db
func (h *CommandHandlers) createCommandRecord(script string) (int, error) {
	var commandID int
	err := h.DB.QueryRow(context.Background(),
		"INSERT INTO commands.commands (script, status) VALUES ($1, 'running') RETURNING id",
		script).Scan(&commandID)

	if err != nil {
		return 0, err
	}
	return commandID, nil
}

// updateCommandOutput updating command output in database
func (h *CommandHandlers) updateCommandOutput(commandID int, output string) {
	_, err := h.DB.Exec(context.Background(),
		"UPDATE commands.commands SET output = $1 WHERE id = $2",
		output, commandID)
	if err != nil {
		h.Logger.Error("Failed to update command output", "error", err)
	}
}

// updateCommandStatus updating status code of script in db
func (h *CommandHandlers) updateCommandStatus(commandID int, status string, output string) {
	_, err := h.DB.Exec(context.Background(),
		"UPDATE commands.commands SET status = $1, output = $2 WHERE id = $3",
		status, output, commandID)
	if err != nil {
		h.Logger.Error("Failed to update command status", "error", err)
	}
}

// updateCommandStatusManually manually updating the status of the command in the database
func (h *CommandHandlers) updateCommandStatusManually(commandID int, status string) error {
	command := &models.Command{}
	err := h.DB.QueryRow(context.Background(),
		"UPDATE commands.commands SET status = $1 WHERE id = $2 RETURNING id, script, status, output",
		status, commandID).Scan(&command.ID, &command.Script, &command.Status, &command.Output)

	return err
}
