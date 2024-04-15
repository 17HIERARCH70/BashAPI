package tests_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/17HIERARCH70/BashAPI/internal/domain/models"
	"github.com/17HIERARCH70/BashAPI/internal/handlers"
	services "github.com/17HIERARCH70/BashAPI/internal/services/command"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockCommandService struct {
	mock.Mock
	services.ICommandService
}

func (m *MockCommandService) ProcessCommand(script string) (gin.H, error) {
	args := m.Called(script)
	if args.Get(0) != nil {
		return args.Get(0).(gin.H), args.Error(1)
	}
	return gin.H{}, args.Error(1)
}

func (m *MockCommandService) FetchCommands() ([]models.Command, error) {
	args := m.Called()
	return args.Get(0).([]models.Command), args.Error(1)
}

func (m *MockCommandService) FetchCommandByID(id int) (models.Command, error) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(models.Command), args.Error(1)
	}
	return models.Command{}, args.Error(1)
}

func (m *MockCommandService) StopCommand(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockCommandService) FetchQueueList() ([]models.Queue, error) {
	args := m.Called()
	return args.Get(0).([]models.Queue), args.Error(1)
}

func (m *MockCommandService) ForceStartCommand(id int) (gin.H, error) {
	args := m.Called(id)
	return args.Get(0).(gin.H), args.Error(1)
}

func (m *MockCommandService) StopAllRunningCommands() error {
	args := m.Called()
	return args.Error(0)
}

func TestCreateCommand(t *testing.T) {
	mockService := new(MockCommandService)
	mockService.On("ProcessCommand", "echo 'Hello, World!'").Return(gin.H{"message": "Command is being executed"}, nil)

	handler := handlers.NewCommandHandlers(mockService, nil) // Logger is nil for simplicity

	router := gin.Default()
	router.POST("/commands", handler.CreateCommand)

	body, _ := json.Marshal(gin.H{"script": "echo 'Hello, World!'"})
	req, _ := http.NewRequest("POST", "/commands", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.JSONEq(t, `{"message":"Command is being executed"}`, w.Body.String())

	mockService.AssertExpectations(t)
}

func TestCreateCommandFailure(t *testing.T) {
	mockService := new(MockCommandService)
	// Ensure a non-nil gin.H{} is returned even when the operation is meant to fail
	mockService.On("ProcessCommand", "fail command").Return(gin.H{}, errors.New("command processing failed"))

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.POST("/commands", handler.CreateCommand)

	body, _ := json.Marshal(gin.H{"script": "fail command"})
	req, _ := http.NewRequest("POST", "/commands", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Ensure that the status code and error message are as expected
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"command processing failed"}`, w.Body.String())
	mockService.AssertExpectations(t)
}

func TestCreateSudoCommandInvalidInput(t *testing.T) {
	mockService := new(MockCommandService)
	handler := handlers.NewCommandHandlers(mockService, nil)

	router := gin.Default()
	router.POST("/commands/sudo", handler.CreateSudoCommand)

	body, _ := json.Marshal(gin.H{})
	req, _ := http.NewRequest("POST", "/commands/sudo", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Script is required"}`, w.Body.String())
}

func TestGetCommandsList(t *testing.T) {
	mockService := new(MockCommandService)
	commands := []models.Command{{ID: 1, Script: "echo 'Hello'"}}
	mockService.On("FetchCommands").Return(commands, nil)

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.GET("/commands", handler.GetCommandsList)

	req, _ := http.NewRequest("GET", "/commands", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expectedBody, _ := json.Marshal(commands)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, string(expectedBody), w.Body.String())
	mockService.AssertExpectations(t)
}

func TestGetCommandsList_Success(t *testing.T) {
	mockService := new(MockCommandService)
	expectedCommands := []models.Command{
		{ID: 1, Script: "echo Hello World", Status: "success"},
		{ID: 2, Script: "ls -l", Status: "pending"},
	}
	mockService.On("FetchCommands").Return(expectedCommands, nil)

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.GET("/commands", handler.GetCommandsList)

	req, _ := http.NewRequest("GET", "/commands", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expectedJSON, _ := json.Marshal(expectedCommands)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, string(expectedJSON), w.Body.String())
	mockService.AssertExpectations(t)
}

func TestGetCommandByID(t *testing.T) {
	mockService := new(MockCommandService)
	command := models.Command{ID: 1, Script: "echo 'Test'"}
	mockService.On("FetchCommandByID", 1).Return(command, nil)

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.GET("/commands/:id", handler.GetCommandByID)

	req, _ := http.NewRequest("GET", "/commands/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expectedBody, _ := json.Marshal(command)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, string(expectedBody), w.Body.String())
	mockService.AssertExpectations(t)
}

func TestStopCommand(t *testing.T) {
	mockService := new(MockCommandService)
	mockService.On("StopCommand", 1).Return(nil)

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.POST("/commands/:id/stop", handler.StopCommand)

	req, _ := http.NewRequest("POST", "/commands/1/stop", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"message":"Command stopped successfully"}`, w.Body.String())
	mockService.AssertExpectations(t)
}

func TestForceStartCommand(t *testing.T) {
	mockService := new(MockCommandService)
	mockService.On("ForceStartCommand", 1).Return(gin.H{"message": "Command started"}, nil)

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.POST("/commands/:id/fstart", handler.ForceStartCommand)

	req, _ := http.NewRequest("POST", "/commands/1/fstart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"message":"Command started"}`, w.Body.String())
	mockService.AssertExpectations(t)
}

func TestGetQueueList(t *testing.T) {
	mockService := new(MockCommandService)
	queue := []models.Queue{{QueueId: 1, CommandId: 2, Status: "waiting"}}
	mockService.On("FetchQueueList").Return(queue, nil)

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.GET("/commands/queue", handler.GetQueueList)

	req, _ := http.NewRequest("GET", "/commands/queue", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expectedBody, _ := json.Marshal(queue)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, string(expectedBody), w.Body.String())
	mockService.AssertExpectations(t)
}

func TestStopCommandInvalidID(t *testing.T) {
	mockService := new(MockCommandService)
	handler := handlers.NewCommandHandlers(mockService, nil)

	router := gin.Default()
	router.POST("/commands/:id/stop", handler.StopCommand)

	req, _ := http.NewRequest("POST", "/commands/abc/stop", nil) // invalid ID format
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Invalid command ID"}`, w.Body.String())
}

func TestGetCommandByIDNotFound(t *testing.T) {
	mockService := new(MockCommandService)
	mockService.On("FetchCommandByID", 999).Return(models.Command{}, services.ErrNotFound)

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.GET("/commands/:id", handler.GetCommandByID)

	req, _ := http.NewRequest("GET", "/commands/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.JSONEq(t, `{"error":"Command not found"}`, w.Body.String())
	mockService.AssertExpectations(t)
}

func TestForceStartCommandFailure(t *testing.T) {
	mockService := new(MockCommandService)
	mockService.On("ForceStartCommand", 2).Return(gin.H{}, errors.New("internal server error"))

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.POST("/commands/:id/fstart", handler.ForceStartCommand)

	req, _ := http.NewRequest("POST", "/commands/2/fstart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())
	mockService.AssertExpectations(t)
}

func TestProcessCommandDBFailure(t *testing.T) {
	mockService := new(MockCommandService)
	mockService.On("ProcessCommand", "db fail").Return(nil, errors.New("database connection failed"))

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.POST("/commands", handler.CreateCommand)

	body, _ := json.Marshal(gin.H{"script": "db fail"})
	req, _ := http.NewRequest("POST", "/commands", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"database connection failed"}`, w.Body.String())
	mockService.AssertExpectations(t)
}

func TestCreateCommandLongScript(t *testing.T) {
	longScript := strings.Repeat("echo 'hello';", 1000) // A very long script
	mockService := new(MockCommandService)
	mockService.On("ProcessCommand", longScript).Return(gin.H{"message": "Long command processed"}, nil)

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.POST("/commands", handler.CreateCommand)

	body, _ := json.Marshal(gin.H{"script": longScript})
	req, _ := http.NewRequest("POST", "/commands", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.JSONEq(t, `{"message":"Long command processed"}`, w.Body.String())
	mockService.AssertExpectations(t)
}

func TestFetchCommandByIDDoesNotExist(t *testing.T) {
	mockService := new(MockCommandService)
	mockService.On("FetchCommandByID", 9999).Return(models.Command{}, services.ErrNotFound) // Use the specific error object

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.GET("/commands/:id", handler.GetCommandByID)

	req, _ := http.NewRequest("GET", "/commands/9999", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.JSONEq(t, `{"error":"Command not found"}`, w.Body.String())
	mockService.AssertExpectations(t)
}

func TestCreateSudoCommandWithSudo(t *testing.T) {
	sudoScript := "sudo ls"
	mockService := new(MockCommandService)
	mockService.On("ProcessCommand", sudoScript).Return(gin.H{"message": "Sudo command executed"}, nil)

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.POST("/commands/sudo", handler.CreateSudoCommand)

	body, _ := json.Marshal(gin.H{"script": sudoScript})
	req, _ := http.NewRequest("POST", "/commands/sudo", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.JSONEq(t, `{"message":"Sudo command executed"}`, w.Body.String())
	mockService.AssertExpectations(t)
}

// Assume this test simulates a server error scenario such as a crash or misconfiguration
func TestInternalServerError(t *testing.T) {
	mockService := new(MockCommandService)
	mockService.On("ProcessCommand", "crash command").Return(nil, errors.New("internal server error"))

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.POST("/commands", handler.CreateCommand)

	body, _ := json.Marshal(gin.H{"script": "crash command"})
	req, _ := http.NewRequest("POST", "/commands", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())
	mockService.AssertExpectations(t)
}

func TestCreateCommandInvalidJSON(t *testing.T) {
	handler := handlers.NewCommandHandlers(new(MockCommandService), nil) // Assume a mock service and no logger
	router := gin.Default()
	router.POST("/commands", handler.CreateCommand)

	// Providing broken JSON
	body := bytes.NewBufferString(`{"script": "echo 'Hello, World!'`) // Missing closing quote and brace
	req, _ := http.NewRequest("POST", "/commands", body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Invalid request body"}`, w.Body.String())
}

func TestCreateCommandEmptyScript(t *testing.T) {
	handler := handlers.NewCommandHandlers(new(MockCommandService), nil)
	router := gin.Default()
	router.POST("/commands", handler.CreateCommand)

	body, _ := json.Marshal(gin.H{"script": ""}) // Empty script
	req, _ := http.NewRequest("POST", "/commands", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Script is required"}`, w.Body.String())
}

func TestCreateCommandContainsSudo(t *testing.T) {
	handler := handlers.NewCommandHandlers(new(MockCommandService), nil)
	router := gin.Default()
	router.POST("/commands", handler.CreateCommand)

	body, _ := json.Marshal(gin.H{"script": "sudo reboot"}) // Script contains 'sudo'
	req, _ := http.NewRequest("POST", "/commands", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Non-Sudo command cannot contain 'sudo'"}`, w.Body.String())
}
func TestCreateSudoCommandInvalidJSON(t *testing.T) {
	handler := handlers.NewCommandHandlers(new(MockCommandService), nil) // Assume a mock service
	router := gin.Default()
	router.POST("/commands/sudo", handler.CreateSudoCommand)

	// Providing broken JSON
	body := bytes.NewBufferString(`{"script": "sudo reboot`) // Missing closing quote and brace
	req, _ := http.NewRequest("POST", "/commands/sudo", body)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Invalid request body"}`, w.Body.String())
}
func TestCreateSudoCommandInternalError(t *testing.T) {
	mockService := new(MockCommandService)
	script := "sudo reboot" // This should match the script you expect to trigger an internal error

	mockService.On("ProcessCommand", script).Return(nil, errors.New("internal server error"))

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.POST("/commands/sudo", handler.CreateSudoCommand)

	body, _ := json.Marshal(gin.H{"script": script})
	req, _ := http.NewRequest("POST", "/commands/sudo", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())
	mockService.AssertExpectations(t)
}
func TestGetCommandsListServiceError(t *testing.T) {
	mockService := new(MockCommandService)
	mockService.On("FetchCommands").Return(([]models.Command)(nil), errors.New("database error")) // Correctly handle nil slices

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.GET("/commands", handler.GetCommandsList)

	req, _ := http.NewRequest("GET", "/commands", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"Failed to fetch commands"}`, w.Body.String())
	mockService.AssertExpectations(t) // Verify that all expectations set on the mock are met
}

func TestGetCommandByIDInvalidID(t *testing.T) {
	handler := handlers.NewCommandHandlers(new(MockCommandService), nil) // Assuming no logger for simplicity
	router := gin.Default()
	router.GET("/commands/:id", handler.GetCommandByID)

	req, _ := http.NewRequest("GET", "/commands/invalid-id", nil) // Use a path with a non-integer ID
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Invalid command ID"}`, w.Body.String())
}

func TestGetCommandByIDServiceError(t *testing.T) {
	mockService := new(MockCommandService)
	mockService.On("FetchCommandByID", 1).Return(models.Command{}, errors.New("internal error"))

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.GET("/commands/:id", handler.GetCommandByID)

	req, _ := http.NewRequest("GET", "/commands/1", nil) // Use a valid integer ID
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"Failed to fetch command"}`, w.Body.String())
	mockService.AssertExpectations(t) // Verify that all expectations set on the mock are met
}

func TestStopCommandNotFound(t *testing.T) {
	mockService := new(MockCommandService)
	commandID := 1
	mockService.On("StopCommand", commandID).Return(services.ErrNotFound)

	handler := handlers.NewCommandHandlers(mockService, nil) // Assuming no logger for simplicity
	router := gin.Default()
	router.POST("/commands/:id/stop", handler.StopCommand)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/commands/%d/stop", commandID), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.JSONEq(t, `{"error":"Command not found"}`, w.Body.String())
}

func TestStopCommandInternalError(t *testing.T) {
	mockService := new(MockCommandService)
	commandID := 1
	mockService.On("StopCommand", commandID).Return(errors.New("internal error"))

	handler := handlers.NewCommandHandlers(mockService, nil) // Logger is nil for simplicity
	router := gin.Default()
	router.POST("/commands/:id/stop", handler.StopCommand)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/commands/%d/stop", commandID), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"Failed to stop command"}`, w.Body.String())
}

func TestGetQueueListInternalError(t *testing.T) {
	mockService := new(MockCommandService)
	expectedError := errors.New("database error")

	mockService.On("FetchQueueList").Return(([]models.Queue)(nil), expectedError)

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.GET("/commands/queue", handler.GetQueueList)

	req, _ := http.NewRequest("GET", "/commands/queue", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"Failed to retrieve queue data"}`, w.Body.String())
	mockService.AssertExpectations(t) // Ensure all expectations were met
}

func TestForceStartCommandInvalidID(t *testing.T) {
	handler := handlers.NewCommandHandlers(new(MockCommandService), nil)
	router := gin.Default()
	router.POST("/commands/:id/fstart", handler.ForceStartCommand)

	req, _ := http.NewRequest("POST", "/commands/invalid-id/fstart", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Invalid command ID"}`, w.Body.String())
}
func TestForceStartCommandNotFound(t *testing.T) {
	mockService := new(MockCommandService)
	commandID := 2
	// Ensure that even when returning an error, a valid gin.H{} is returned to prevent type assertion panics
	mockService.On("ForceStartCommand", commandID).Return(gin.H{}, services.ErrNotFound)

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.POST("/commands/:id/fstart", handler.ForceStartCommand)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/commands/%d/fstart", commandID), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.JSONEq(t, `{"error":"Command not found"}`, w.Body.String())
}

func TestForceStartCommandInternalError(t *testing.T) {
	mockService := new(MockCommandService)
	commandID := 2
	// Return an empty gin.H even in error scenarios
	mockService.On("ForceStartCommand", commandID).Return(gin.H{}, errors.New("internal error"))

	handler := handlers.NewCommandHandlers(mockService, nil)
	router := gin.Default()
	router.POST("/commands/:id/fstart", handler.ForceStartCommand)

	req, _ := http.NewRequest("POST", fmt.Sprintf("/commands/%d/fstart", commandID), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"internal error"}`, w.Body.String())
}
