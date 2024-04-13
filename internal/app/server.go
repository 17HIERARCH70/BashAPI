package server

import (
	"fmt"
	"github.com/17HIERARCH70/BashAPI/internal/config"
	"github.com/17HIERARCH70/BashAPI/internal/handlers"
	services "github.com/17HIERARCH70/BashAPI/internal/services/command"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/net/context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server represents an HTTP server.
type Server struct {
	Config         *config.Config
	Logger         *slog.Logger
	DB             *pgxpool.Pool
	Router         *gin.Engine
	HttpServer     *http.Server
	CommandService *services.CommandService
}

// NewServer creates a new HTTP server and sets up routing.
func NewServer(cfg *config.Config, log *slog.Logger, db *pgxpool.Pool) *Server {
	router := gin.New()
	commandService := services.NewCommandService(db, log, cfg.Server.MaxConcurrent)
	commandHandlers := handlers.NewCommandHandlers(commandService, log)
	loggerMiddleware := createLoggerMiddleware(log)

	httpServer := &http.Server{
		Addr:         cfg.Server.Host + ":" + fmt.Sprintf("%d", cfg.Server.Port),
		Handler:      router, // Assigning the gin router as the handler
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	server := &Server{
		Config:         cfg,
		Logger:         log,
		DB:             db,
		Router:         router,
		HttpServer:     httpServer,
		CommandService: commandService,
	}
	server.executeQueuedCommands()
	SetupRoutes(router, commandHandlers, loggerMiddleware)
	return server
}

func (s *Server) executeQueuedCommands() {
	s.Logger.Info("Checking for queued commands to execute on startup...")
	queuedCommands, err := s.CommandService.FetchQueueList()
	if err != nil {
		s.Logger.Error("Failed to fetch queued commands", "error", err)
		return
	}

	for _, queueItem := range queuedCommands {
		queueItem := queueItem
		go func() {
			_, _ = s.CommandService.ForceStartCommand(queueItem.CommandId)
		}()
	}
	s.Logger.Info("Queued commands are being processed...")
}

// createLoggerMiddleware creates middleware for logging requests using slog.
func createLoggerMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		latency := time.Since(startTime)
		log.Info("request",
			"status", c.Writer.Status(),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"ip", c.ClientIP(),
			"latency", latency,
			"user-agent", c.Request.UserAgent(),
			"errors", c.Errors.ByType(gin.ErrorTypePrivate).String(),
		)
	}
}

// Start runs the HTTP server on a specific address.
func (s *Server) Start(address string) {
	if err := s.Router.Run(address); err != nil {
		s.Logger.Error("Failed to run server", "error", err)
	}
}

// GracefulShutdown handles the graceful shutdown of the server,
// stopping all running commands and preserving the queue.
func (s *Server) GracefulShutdown() {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan // Wait for SIGINT or SIGTERM

	s.Logger.Info("Initiating graceful shutdown, stopping all running commands.")

	// Stop all running commands
	if err := s.CommandService.StopAllRunningCommands(); err != nil {
		s.Logger.Error("Failed to stop running commands during shutdown", "error", err)
	} else {
		s.Logger.Info("All running commands have been stopped.")
	}

	// Shutdown the HTTP server with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.HttpServer.Shutdown(ctx); err != nil {
		s.Logger.Error("HTTP server shutdown failed", "error", err)
	} else {
		s.Logger.Info("HTTP server shutdown successfully")
	}
	// Close the database connection
	s.DB.Close()
	s.Logger.Info("Database connection closed successfully")

	s.Logger.Info("Server shutdown complete. Commands in the queue will resume on next start.")
}
