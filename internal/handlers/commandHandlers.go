package handlers

import (
	"errors"
	services "github.com/17HIERARCH70/BashAPI/internal/services/command"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// CommandHandlers Structure for organizing command handlers.
type CommandHandlers struct {
	Service services.ICommandService
	Logger  *slog.Logger
}

// NewCommandHandlers creates an instance CommandHandlers.
func NewCommandHandlers(service services.ICommandService, logger *slog.Logger) *CommandHandlers {
	if logger == nil {
		logger = slog.Default() // Set a default logger if none is provided
	}
	return &CommandHandlers{
		Service: service,
		Logger:  logger,
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

	response, err := h.Service.ProcessCommand(command.Script)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, response)
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

	response, err := h.Service.ProcessCommand(command.Script)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, response)
}

// GetCommandsList handler to retrieve the list of commands.
func (h *CommandHandlers) GetCommandsList(c *gin.Context) {
	commands, err := h.Service.FetchCommands()
	if err != nil {
		if h.Logger != nil {
			h.Logger.Error("Failed to fetch commands", "error", err)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch commands"})
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

	command, err := h.Service.FetchCommandByID(commandID)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) { // Assuming services.ErrNotFound is the specific error for "not found"
			c.JSON(http.StatusNotFound, gin.H{"error": "Command not found"})
		} else {
			if h.Logger != nil {
				h.Logger.Error("Failed to fetch command", "error", err)
			}
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

	err = h.Service.StopCommand(commandID)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
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
	queue, err := h.Service.FetchQueueList()
	if err != nil {
		h.Logger.Error("Failed to retrieve queue data", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve queue data"})
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

	message, err := h.Service.ForceStartCommand(commandID)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Command not found"})
		} else {
			if h.Logger != nil { // Check if Logger is not nil before logging
				h.Logger.Error("Failed to forcefully start command", "error", err)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, message)
}
