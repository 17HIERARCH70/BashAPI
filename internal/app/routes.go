package server

import (
	_ "github.com/17HIERARCH70/BashAPI/docs"
	"github.com/17HIERARCH70/BashAPI/internal/handlers"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRoutes sets up the routes for the server.
func SetupRoutes(router *gin.Engine, commandHandlers *handlers.CommandHandlers, loggerMiddleware gin.HandlerFunc) {
	router.Use(loggerMiddleware)
	api := router.Group("/api")
	{
		commands := api.Group("/commands")
		{
			// Create a command
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
			// Get queue list
			commands.GET("/queue", commandHandlers.GetQueueList)
			// Swagger UI route
			api.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		}
	}
}
