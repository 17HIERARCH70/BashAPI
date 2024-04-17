// @title			BashAPi service
// @version		1.0
// @description	RestAPI for executing bash commands in Docker with a queue system.
// @BasePath		/api/commands
package main

import (
	"fmt"
	server "github.com/17HIERARCH70/BashAPI/internal/app"
	"github.com/17HIERARCH70/BashAPI/internal/config"
	"github.com/17HIERARCH70/BashAPI/internal/logger"
	"github.com/17HIERARCH70/BashAPI/internal/storage/postgresql"
)

func main() {
	cfg := config.MustLoad()
	log := logger.SetupLogger(cfg.Env)
	db, err := postgresql.InitializeDB(cfg)
	if err != nil {
		log.Error("Failed to initialize the database", "error", err)
		return
	}
	defer db.Close()
	srv := server.NewServer(cfg, log, db)
	go func() {
		srv.Start(cfg.Server.Host + ":" + fmt.Sprintf("%d", cfg.Server.Port))
	}()
	srv.GracefulShutdown()
}
