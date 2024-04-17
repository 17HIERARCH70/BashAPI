package handlers

import (
	"errors"
	_ "github.com/17HIERARCH70/BashAPI/internal/domain/models"
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

// CreateCommand godoc
//
//	@Summary		Create a new command
//	@Description	Add a new non-sudo command to the system
//	@Tags			Commands creating
//	@Accept			json
//	@Produce		json
//	@Param			command	body		string			true	"Create command"
//	@Success		202		{object}	models.Message	"Command is being executed"
//	@Success		202		{object}	models.Message	"Command is being queued"
//	@Failure		400		{object}	models.Error	"Error response"
//	@Failure		500		{object}	models.Error	"Error response on server side"
//	@Router			/ [post]
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

// CreateSudoCommand godoc
//
//	@Summary		Create a new sudo command
//	@Description	Add a new sudo command to the system
//	@Tags			Commands creating
//	@Accept			json
//	@Produce		json
//	@Param			command	body		string			true	"Create sudo command"
//	@Success		202		{object}	models.Message	"Command is being executed"
//	@Success		202		{object}	models.Message	"Command is being queued"
//	@Failure		400		{object}	models.Error	"Error response"
//	@Failure		500		{object}	models.Error	"Error response on server side"
//	@Router			/sudo [post]
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

// GetCommandsList godoc
//
//	@Summary		Retrieve all commands
//	@Description	Get a list of all commands processed by the system
//	@Tags			Getting commands
//	@Produce		json
//	@Success		200	{array}		models.Command	"List of commands"
//	@Failure		500	{object}	models.Error	"Server error"
//	@Router			/ [get]
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

// GetCommandByID godoc
//
//	@Summary		Get a command by ID
//	@Description	Retrieve a specific command by its unique ID
//	@Tags			Getting commands
//	@Produce		json
//	@Param			id	path		int				true	"Command ID"
//	@Success		200	{object}	models.Command	"Command detail"
//	@Failure		500	{object}	models.Error	"Problem on server side"
//	@Failure		404	{object}	models.Error	"Command not found"
//	@Failure		400	{object}	models.Error	"Invalid ID supplied"
//	@Router			/{id} [get]
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

// StopCommand godoc
//
//	@Summary		Stop a command
//	@Description	Stop a running command by its ID
//	@Tags			Fetching commands
//	@Produce		json
//	@Param			id	path		int				true	"Command ID"
//	@Success		200	{object}	models.Message	"Command stopped successfully"
//	@Failure		500	{object}	models.Error	"Problem on server side"
//	@Failure		404	{object}	models.Error	"Command not found"
//	@Failure		400	{object}	models.Error	"Invalid ID supplied"
//	@Router			/{id}/stop [post]
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

// GetQueueList godoc
//
//	@Summary		Retrieve command queue
//	@Description	Get a list of all commands currently in the queue
//	@Tags			Queue
//	@Produce		json
//	@Success		200	{array}		models.Queue	"List of queued items"
//	@Failure		500	{object}	models.Error	"Server error"
//	@Router			/queue [get]
func (h *CommandHandlers) GetQueueList(c *gin.Context) {
	queue, err := h.Service.FetchQueueList()
	if err != nil {
		h.Logger.Error("Failed to retrieve queue data", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve queue data"})
		return
	}
	c.JSON(http.StatusOK, queue)
}

// ForceStartCommand godoc
//
//	@Summary		Force start a command
//	@Description	Forcefully start a queued command by its ID, bypassing queue constraints
//	@Tags			Fetching commands
//	@Produce		json
//	@Param			id	path		int				true	"Command ID"
//	@Success		200	{object}	models.Message	"Command started successfully"
//	@Failure		404	{object}	models.Error	"Command not found"
//	@Failure		500	{object}	models.Error	"Server error"
//	@Failure		400	{object}	models.Error	"Invalid ID supplied"
//	@Router			/commands/{id}/fstart [post]
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
