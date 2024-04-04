package main

import (
	"fmt"
	server "github.com/17HIERARCH70/BashAPI/internal/app"
	"github.com/17HIERARCH70/BashAPI/internal/config"
	"github.com/17HIERARCH70/BashAPI/internal/logger"
	"github.com/17HIERARCH70/BashAPI/internal/storage/postgresql"
)

func main() {
	// Init config.
	cfg := config.MustLoad()
	// Init pretty logger.
	log := logger.SetupLogger(cfg.Env)
	// Init database.
	db, err := postgresql.InitializeDB(cfg)
	if err != nil {
		log.Error("Failed to initialize the database", "error", err)
		return
	}
	defer db.Close()
	// Init app. Will use Gin for high-performance.
	srv := server.NewServer(cfg, log, db)

	// Start server in a goroutine to not block graceful shutdown listening
	go func() {
		srv.Start(cfg.Server.Host + ":" + fmt.Sprintf("%d", cfg.Server.Port))
	}()

	// Listen for shutdown signals
	srv.GracefulShutdown()
}
