package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vmmanager/config"
	"vmmanager/internal/api/routes"
	"vmmanager/internal/database"
	"vmmanager/internal/libvirt"
	"vmmanager/internal/middleware"
	"vmmanager/internal/tasks"
	"vmmanager/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 初始化数据库
	db, err := database.NewPostgreSQL(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 自动迁移数据库表
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 初始化Libvirt客户端
	libvirtClient, err := libvirt.NewClient(cfg.Libvirt.URI)
	if err != nil {
		log.Printf("Warning: Failed to connect to libvirt: %v", err)
		log.Println("VM functionality will be limited")
	} else {
		defer libvirtClient.Close()
	}

	// 初始化WebSocket处理器
	wsHandler := websocket.NewHandler(libvirtClient)

	// 初始化定时任务
	scheduler := tasks.NewScheduler(db, libvirtClient)
	go scheduler.Start()

	// 设置Gin模式
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin路由
	router := gin.Default()

	// 全局中间件
	router.Use(middleware.CORS())
	router.Use(middleware.Logger())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// API文档
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 注册路由
	routes.Register(router, cfg, db, libvirtClient, wsHandler)

	// 创建HTTP服务器
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.HTTP_PORT),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 创建WebSocket服务器
	wsServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.WS_PORT),
		Handler: wsHandler,
	}

	// 优雅关闭
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

	// 启动HTTP服务器
	log.Printf("Starting HTTP server on %s:%d", cfg.App.Host, cfg.App.HTTP_PORT)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 启动WebSocket服务器
	log.Printf("Starting WebSocket server on %s:%d", cfg.App.Host, cfg.App.WS_PORT)
	if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("WebSocket server error: %v", err)
	}
}
