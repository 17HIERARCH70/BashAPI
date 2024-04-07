package server

import (
	"fmt"
	"github.com/17HIERARCH70/BashAPI/internal/config"
	"github.com/17HIERARCH70/BashAPI/internal/handlers"
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
	Config     *config.Config
	Logger     *slog.Logger
	DB         *pgxpool.Pool
	Router     *gin.Engine
	HttpServer *http.Server
}

// NewServer creates a new HTTP server and sets up routing.
func NewServer(cfg *config.Config, log *slog.Logger, db *pgxpool.Pool) *Server {
	router := gin.New()

	httpServer := &http.Server{
		Addr:         cfg.Server.Host + ":" + fmt.Sprintf("%d", cfg.Server.Port),
		Handler:      router, // Gin router
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	server := &Server{
		Config:     cfg,
		Logger:     log,
		DB:         db,
		Router:     gin.New(),
		HttpServer: httpServer,
	}

	server.setupRoutes()
	return server
}

// slogLoggerMiddleware Creates middleware for logging requests using slog.
func (s *Server) slogLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		c.Next()
		latency := time.Since(startTime)

		s.Logger.Info("request",
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

// setupRoutes sets up the routes for the server.
func (s *Server) setupRoutes() {
	commandHandlers := handlers.NewCommandHandlers(s.DB, s.Logger, s.Config.Server.MaxConcurrent, s.Config.Server.OSpas)
	s.Router.Use(s.slogLoggerMiddleware())
	api := s.Router.Group("/api")
	{
		commands := api.Group("/commands")
		{
			// Create a new command
			commands.POST("/", commandHandlers.CreateCommand)
			// Create a sudo command
			commands.POST("/sudo", commandHandlers.CreateSudoCommand)
			// Get list of all commands
			commands.GET("/", commandHandlers.GetCommandsList)
			// Get one command by its ID
			commands.GET("/:id", commandHandlers.GetCommandByID)
			// Stop command by ID
			commands.POST("/:id/stop", commandHandlers.StopCommand)
			// Force start command by ID
			commands.POST("/:id/fstart", commandHandlers.ForceStartCommand)
			// Get list of queue
			commands.GET("/queue", commandHandlers.GetQueueList)
		}
	}
}

// Start runs the HTTP server on a specific address.
func (s *Server) Start(address string) {
	if err := s.Router.Run(address); err != nil {
		s.Logger.Error("Failed to run server", "error", err)
	}
}

// GracefulShutdown adds a graceful shutdown mechanism to the server.
func (s *Server) GracefulShutdown() {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	<-stopChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = s.HttpServer.Shutdown(ctx)
	s.Logger.Info("Shutting down gracefully, press Ctrl+C again to force")
}
