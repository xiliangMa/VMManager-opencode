package routes

import (
	"vmmanager/config"
	"vmmanager/internal/api/handlers"
	"vmmanager/internal/api/middleware"
	"vmmanager/internal/libvirt"
	"vmmanager/internal/repository"
	"vmmanager/internal/websocket"

	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine, cfg *config.Config, repos *repository.Repositories, libvirtClient *libvirt.Client, wsHandler *websocket.Handler) {
	jwtMiddleware := middleware.JWTRequired(cfg.JWT.Secret)

	authHandler := handlers.NewAuthHandler(repos.User, cfg.JWT)
	vmHandler := handlers.NewVMHandler(repos.VM, repos.User, repos.Template, repos.VMStats)
	templateHandler := handlers.NewTemplateHandler(repos.Template, repos.TemplateUpload)
	adminHandler := handlers.NewAdminHandler(repos.User, repos.VM, repos.Template, repos.AuditLog)

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", jwtMiddleware, authHandler.Logout)
			auth.GET("/profile", jwtMiddleware, authHandler.GetProfile)
			auth.PUT("/profile", jwtMiddleware, authHandler.UpdateProfile)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		vms := api.Group("/vms")
		{
			vms.Use(jwtMiddleware)
			vms.GET("", vmHandler.ListVMs)
			vms.POST("", vmHandler.CreateVM)
			vms.GET("/:id", vmHandler.GetVM)
			vms.PUT("/:id", vmHandler.UpdateVM)
			vms.DELETE("/:id", vmHandler.DeleteVM)
			vms.POST("/:id/start", vmHandler.StartVM)
			vms.POST("/:id/stop", vmHandler.StopVM)
			vms.POST("/:id/force-stop", vmHandler.ForceStopVM)
			vms.POST("/:id/restart", vmHandler.RebootVM)
			vms.POST("/:id/suspend", vmHandler.SuspendVM)
			vms.POST("/:id/resume", vmHandler.ResumeVM)
			vms.GET("/:id/console", vmHandler.GetConsole)
			vms.GET("/:id/stats", vmHandler.GetVMStats)
		}

		templates := api.Group("/templates")
		{
			templates.Use(jwtMiddleware)
			templates.GET("", templateHandler.ListTemplates)
			templates.GET("/:id", templateHandler.GetTemplate)
			templates.POST("", middleware.AdminRequired(), templateHandler.CreateTemplate)
			templates.PUT("/:id", middleware.AdminRequired(), templateHandler.UpdateTemplate)
			templates.DELETE("/:id", middleware.AdminRequired(), templateHandler.DeleteTemplate)
			templates.POST("/upload/init", middleware.AdminRequired(), templateHandler.InitTemplateUpload)
			templates.POST("/upload/part", middleware.AdminRequired(), templateHandler.UploadTemplatePart)
			templates.POST("/upload/complete", middleware.AdminRequired(), templateHandler.CompleteTemplateUpload)
		}

		admin := api.Group("/admin")
		admin.Use(jwtMiddleware)
		admin.Use(middleware.AdminRequired())
		{
			users := admin.Group("/users")
			{
				users.GET("", adminHandler.ListUsers)
				users.POST("", adminHandler.CreateUser)
				users.GET("/:id", adminHandler.GetUser)
				users.PUT("/:id", adminHandler.UpdateUser)
				users.DELETE("/:id", adminHandler.DeleteUser)
				users.PUT("/:id/quota", adminHandler.UpdateUserQuota)
				users.PUT("/:id/role", adminHandler.UpdateUserRole)
			}

			admin.GET("/audit-logs", adminHandler.ListAuditLogs)
			admin.GET("/system/info", adminHandler.GetSystemInfo(libvirtClient))
			admin.GET("/system/stats", adminHandler.GetSystemStats(libvirtClient))
		}
	}
}
