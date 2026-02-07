package routes

import (
	"vmmanager/config"

	"vmmanager/internal/api/handlers"
	"vmmanager/internal/api/middleware"
	"vmmanager/internal/libvirt"
	"vmmanager/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag/example/basic/docs"
)

func Register(router *gin.Engine, cfg *config.Config, db interface{}, libvirtClient *libvirt.Client, wsHandler *websocket.Handler) {
	docs.SwaggerInfo.Title = "VMManager API"
	docs.SwaggerInfo.Description = "虚拟机管理平台 API 文档"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:8080"
	docs.SwaggerInfo.BasePath = "/api/v1"

	jwtMiddleware := middleware.JWTRequired(cfg.JWT.Secret)

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login(cfg))
			auth.POST("/logout", jwtMiddleware, handlers.Logout)
			auth.GET("/profile", jwtMiddleware, handlers.GetProfile)
			auth.PUT("/profile", jwtMiddleware, handlers.UpdateProfile)
			auth.POST("/refresh", handlers.RefreshToken(cfg))
		}

		vms := api.Group("/vms")
		{
			vms.Use(jwtMiddleware)
			vms.GET("", handlers.ListVMs)
			vms.POST("", handlers.CreateVM)
			vms.GET("/:id", handlers.GetVM)
			vms.PUT("/:id", handlers.UpdateVM)
			vms.DELETE("/:id", handlers.DeleteVM)
			vms.POST("/:id/start", handlers.StartVM)
			vms.POST("/:id/stop", handlers.StopVM)
			vms.POST("/:id/force-stop", handlers.ForceStopVM)
			vms.POST("/:id/restart", handlers.RebootVM)
			vms.POST("/:id/suspend", handlers.SuspendVM)
			vms.POST("/:id/resume", handlers.ResumeVM)
			vms.GET("/:id/console", handlers.GetConsole)
			vms.GET("/:id/stats", handlers.GetVMStats)
		}

		templates := api.Group("/templates")
		{
			templates.Use(jwtMiddleware)
			templates.GET("", handlers.ListTemplates)
			templates.GET("/:id", handlers.GetTemplate)
			templates.POST("", middleware.AdminRequired(), handlers.CreateTemplate)
			templates.PUT("/:id", middleware.AdminRequired(), handlers.UpdateTemplate)
			templates.DELETE("/:id", middleware.AdminRequired(), handlers.DeleteTemplate)
			templates.POST("/upload/init", middleware.AdminRequired(), handlers.InitTemplateUpload)
			templates.POST("/upload/part", middleware.AdminRequired(), handlers.UploadTemplatePart)
			templates.POST("/upload/complete", middleware.AdminRequired(), handlers.CompleteTemplateUpload)
		}

		admin := api.Group("/admin")
		admin.Use(jwtMiddleware)
		admin.Use(middleware.AdminRequired())
		{
			users := admin.Group("/users")
			{
				users.GET("", handlers.ListUsers)
				users.POST("", handlers.CreateUser)
				users.GET("/:id", handlers.GetUser)
				users.PUT("/:id", handlers.UpdateUser)
				users.DELETE("/:id", handlers.DeleteUser)
				users.PUT("/:id/quota", handlers.UpdateUserQuota)
				users.PUT("/:id/role", handlers.UpdateUserRole)
			}

			admin.GET("/audit-logs", handlers.ListAuditLogs)
			admin.GET("/system/info", handlers.GetSystemInfo(libvirtClient))
			admin.GET("/system/stats", handlers.GetSystemStats(libvirtClient))
		}
	}
}
