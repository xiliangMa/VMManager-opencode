//go:build !linux || mock
// +build !linux mock

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vmmanager/config"
	"vmmanager/internal/api/errors"
	"vmmanager/internal/api/routes"
	"vmmanager/internal/database"
	"vmmanager/internal/libvirt"
	"vmmanager/internal/middleware"
	"vmmanager/internal/repository"
	"vmmanager/internal/tasks"
	"vmmanager/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting VMManager with %s database...", cfg.Database.Driver)
	if cfg.Database.Driver == "sqlite" {
		log.Printf("SQLite database path: %s", cfg.Database.Path)
	}

	db, err := database.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	if err := database.Seed(db); err != nil {
		log.Printf("Warning: Failed to seed database: %v", err)
	}

	repos := repository.NewRepositories(db)

	libvirtClient, err := libvirt.NewClient(cfg.Libvirt.URI)
	if err != nil {
		log.Printf("Warning: Failed to connect to libvirt: %v", err)
		log.Println("VM functionality will be limited")
	} else {
		defer libvirtClient.Close()
	}

	wsHandler := websocket.NewHandler(libvirtClient)

	scheduler := tasks.NewScheduler(db, libvirtClient)
	go scheduler.Start()

	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.Use(middleware.CORS())
	router.Use(middleware.Logger())
	router.Use(middleware.MetricsMiddleware())
	router.Use(errors.Recovery())
	router.Use(errors.ErrorHandler())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	router.GET("/metrics", middleware.MetricsHandler())

	router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(302, "/swagger/")
	})

	router.Static("/swagger/", "./docs/swagger")

	router.GET("/ws/vnc/:vm_id", func(c *gin.Context) {
		vmID := c.Param("vm_id")
		wsHandler.HandleVNC(c.Writer, c.Request, vmID)
	})

	routes.Register(router, cfg, repos, libvirtClient, wsHandler)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.HTTPPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	wsServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.WSPort),
		Handler: wsHandler,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down servers...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}

		if err := wsServer.Shutdown(ctx); err != nil {
			log.Printf("WebSocket server shutdown error: %v", err)
		}

		scheduler.Stop()
		log.Println("Servers stopped")
	}()

	log.Printf("Starting HTTP server on %s:%d", cfg.App.Host, cfg.App.HTTPPort)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	log.Printf("Starting WebSocket server on %s:%d", cfg.App.Host, cfg.App.WSPort)
	if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("WebSocket server error: %v", err)
	}
}
