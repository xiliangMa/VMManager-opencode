package routes

import (
	"vmmanager/config"
	"vmmanager/internal/api/handlers"
	"vmmanager/internal/api/middleware"
	"vmmanager/internal/libvirt"
	"vmmanager/internal/repository"
	"vmmanager/internal/services"
	"vmmanager/internal/websocket"

	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine, cfg *config.Config, repos *repository.Repositories, libvirtClient *libvirt.Client, wsHandler *websocket.Handler) {
	jwtMiddleware := middleware.JWTRequired(cfg.JWT.Secret)

	auditService := services.NewAuditService(repos.AuditLog)

	authHandler := handlers.NewAuthHandler(repos.User, cfg.JWT)
	authHandler.SetAuditService(auditService)
	vmHandler := handlers.NewVMHandler(repos.VM, repos.User, repos.Template, repos.VMStats, repos.ISO, libvirtClient, cfg.Storage.Path, auditService)
	templateHandler := handlers.NewTemplateHandler(repos.Template, repos.TemplateUpload, repos.VM)
	templateHandler.SetAuditService(auditService)
	adminHandler := handlers.NewAdminHandler(repos.User, repos.VM, repos.Template, repos.AuditLog)
	auditHandler := handlers.NewAuditHandler(repos.AuditLog)
	snapshotHandler := handlers.NewSnapshotHandler(repos.VM)
	batchHandler := handlers.NewBatchHandler(repos.VM)
	statsHandler := handlers.NewVMStatsHandler(repos.VMStats, repos.DB)
	alertRuleHandler := handlers.NewAlertRuleHandler(repos.AlertRule)
	alertHistoryHandler := handlers.NewAlertHistoryHandler(repos.AlertHistory)
	isoHandler := handlers.NewISOHandler(repos.ISO, repos.ISOUpload)

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", jwtMiddleware, authHandler.Logout)
			auth.GET("/profile", jwtMiddleware, authHandler.GetProfile)
			auth.PUT("/profile", jwtMiddleware, authHandler.UpdateProfile)
			auth.POST("/profile/avatar", jwtMiddleware, authHandler.UpdateAvatar)
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
			vms.GET("/:id/stats", statsHandler.GetVMStats)
			vms.GET("/:id/history", statsHandler.GetVMHistory)
			vms.POST("/:id/start-installation", vmHandler.StartInstallation)
			vms.POST("/:id/finish-installation", vmHandler.FinishInstallation)
			vms.POST("/:id/install-agent", vmHandler.InstallAgent)
			vms.GET("/:id/installation-status", vmHandler.GetInstallationStatus)
			vms.POST("/:id/mount-iso", vmHandler.MountISO)
			vms.DELETE("/:id/mount-iso", vmHandler.UnmountISO)
			vms.GET("/:id/mounted-iso", vmHandler.GetMountedISO)
			vms.POST("/:id/clone", vmHandler.CloneVM)

			snapshots := vms.Group("/:id/snapshots")
			{
				snapshots.POST("", snapshotHandler.CreateSnapshot)
				snapshots.GET("", snapshotHandler.ListSnapshots)
				snapshots.GET("/:name", snapshotHandler.GetSnapshot)
				snapshots.POST("/:name/restore", snapshotHandler.RestoreSnapshot)
				snapshots.DELETE("/:name", snapshotHandler.DeleteSnapshot)
			}

			batch := vms.Group("/batch")
			{
				batch.POST("/start", batchHandler.BatchStart)
				batch.POST("/stop", batchHandler.BatchStop)
				batch.POST("/restart", batchHandler.BatchRestart)
				batch.DELETE("", batchHandler.BatchDelete)
			}
		}

		templates := api.Group("/templates")
		{
			templates.Use(jwtMiddleware)
			templates.GET("", templateHandler.ListTemplates)
			templates.GET("/:id", templateHandler.GetTemplate)
			templates.GET("/:id/vms", templateHandler.GetTemplateVMs)
			templates.POST("", middleware.AdminRequired(), templateHandler.CreateTemplate)
			templates.PUT("/:id", middleware.AdminRequired(), templateHandler.UpdateTemplate)
			templates.DELETE("/:id", middleware.AdminRequired(), templateHandler.DeleteTemplate)

			uploads := templates.Group("/upload")
			{
				uploads.POST("/init", middleware.AdminRequired(), templateHandler.InitTemplateUpload)
				uploads.POST("/part", middleware.AdminRequired(), templateHandler.UploadTemplatePart)
				uploads.POST("/complete/:id", middleware.AdminRequired(), templateHandler.CompleteTemplateUpload)
				uploads.DELETE("/:id", middleware.AdminRequired(), templateHandler.AbortUpload)
				uploads.GET("/:id/status", middleware.AdminRequired(), templateHandler.GetUploadStatus)
			}
		}

		isos := api.Group("/isos")
		{
			isos.Use(jwtMiddleware)
			isos.GET("", isoHandler.ListISOs)
			isos.GET("/:id", isoHandler.GetISO)
			isos.DELETE("/:id", middleware.AdminRequired(), isoHandler.DeleteISO)

			isoUploads := isos.Group("/upload")
			{
				isoUploads.POST("/init", middleware.AdminRequired(), isoHandler.InitISOUpload)
				isoUploads.POST("/part", middleware.AdminRequired(), isoHandler.UploadISOPart)
				isoUploads.POST("/complete", middleware.AdminRequired(), isoHandler.CompleteISOUpload)
				isoUploads.GET("/status", middleware.AdminRequired(), isoHandler.GetUploadStatus)
			}
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
				users.GET("/:id/resource-usage", adminHandler.GetUserResourceUsage)
				users.PUT("/:id", adminHandler.UpdateUser)
				users.DELETE("/:id", adminHandler.DeleteUser)
				users.PUT("/:id/quota", adminHandler.UpdateUserQuota)
				users.PUT("/:id/role", adminHandler.UpdateUserRole)
			}

			alertRules := admin.Group("/alert-rules")
			{
				alertRules.GET("", alertRuleHandler.ListAlertRules)
				alertRules.GET("/:id", alertRuleHandler.GetAlertRule)
				alertRules.POST("", alertRuleHandler.CreateAlertRule)
				alertRules.PUT("/:id", alertRuleHandler.UpdateAlertRule)
				alertRules.DELETE("/:id", alertRuleHandler.DeleteAlertRule)
				alertRules.POST("/:id/toggle", alertRuleHandler.ToggleAlertRule)
				alertRules.GET("/stats/summary", alertRuleHandler.GetAlertStats)
			}

			alertHistories := admin.Group("/alert-histories")
			{
				alertHistories.GET("", alertHistoryHandler.ListAlertHistories)
				alertHistories.GET("/:id", alertHistoryHandler.GetAlertHistory)
				alertHistories.POST("/:id/resolve", alertHistoryHandler.ResolveAlertHistory)
				alertHistories.GET("/active", alertHistoryHandler.GetActiveAlerts)
				alertHistories.GET("/stats", alertHistoryHandler.GetAlertStats)
			}

			admin.GET("/audit-logs", auditHandler.ListAuditLogs)
			admin.GET("/audit-logs/:id", auditHandler.GetAuditLog)
			admin.GET("/audit-logs/user/:id", auditHandler.ListByUser)
			admin.GET("/audit-logs/action/:action", auditHandler.ListByAction)
			admin.GET("/audit-logs/export", auditHandler.ExportAuditLogs)
			admin.GET("/system/resources", adminHandler.GetSystemResources(libvirtClient))
			admin.GET("/system/stats", adminHandler.GetSystemStats(libvirtClient))
			admin.GET("/system/storage", statsHandler.GetStorageStats)
		}
	}
}
