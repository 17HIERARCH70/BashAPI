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
	"syscall"
	"time"
)

// CommandHandlers Structure for organizing command handlers.
type CommandHandlers struct {
	DB     *pgxpool.Pool
	Logger *slog.Logger
}

// NewCommandHandlers creates an instance CommandHandlers.
func NewCommandHandlers(db *pgxpool.Pool, logger *slog.Logger) *CommandHandlers {
	return &CommandHandlers{
		DB:     db,
		Logger: logger,
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

	commandID, err := h.createCommandRecord(command.Script)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create command record"})
		return
	}

	// Run the script asynchronously
	go h.executeCommand(commandID, command.Script)

	c.JSON(202, gin.H{"message": "Command is being executed", "id": commandID})
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

// updateCommandStatusManually manually updating the status of the command in the database
func (h *CommandHandlers) updateCommandStatusManually(commandID int, status string) error {
	command := &models.Command{}
	err := h.DB.QueryRow(context.Background(),
		"UPDATE commands.commands SET status = $1 WHERE id = $2 RETURNING id, script, status, output",
		status, commandID).Scan(&command.ID, &command.Script, &command.Status, &command.Output)

	return err
}
