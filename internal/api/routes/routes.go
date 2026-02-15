package routes

import (
	"vmmanager/config"
	"vmmanager/internal/api/handlers"
	"vmmanager/internal/libvirt"
	"vmmanager/internal/middleware"
	"vmmanager/internal/repository"
	"vmmanager/internal/services"
	"vmmanager/internal/websocket"

	"github.com/gin-gonic/gin"
)

func Register(router *gin.Engine, cfg *config.Config, repos *repository.Repositories, libvirtClient *libvirt.Client, wsHandler *websocket.Handler, backupService *services.BackupService) {
	jwtMiddleware := middleware.JWTRequired(cfg.JWT.Secret)

	auditService := services.NewAuditService(repos.AuditLog)

	authHandler := handlers.NewAuthHandler(repos.User, cfg.JWT)
	authHandler.SetAuditService(auditService)
	vmHandler := handlers.NewVMHandler(repos.VM, repos.User, repos.Template, repos.VMStats, repos.ISO, libvirtClient, cfg.Storage.Path, auditService)
	templateHandler := handlers.NewTemplateHandler(repos.Template, repos.TemplateUpload, repos.VM)
	templateHandler.SetAuditService(auditService)
	adminHandler := handlers.NewAdminHandler(repos.User, repos.VM, repos.Template, repos.AuditLog)
	auditHandler := handlers.NewAuditHandler(repos.AuditLog)
	snapshotHandler := handlers.NewSnapshotHandler(repos.VM, repos.VMSnapshot, libvirtClient)
	batchHandler := handlers.NewBatchHandler(repos.VM, libvirtClient, cfg.Storage.Path, auditService)
	statsHandler := handlers.NewVMStatsHandler(repos.VMStats, repos.DB)
	alertRuleHandler := handlers.NewAlertRuleHandler(repos.AlertRule)
	alertHistoryHandler := handlers.NewAlertHistoryHandler(repos.AlertHistory)
	isoHandler := handlers.NewISOHandler(repos.ISO, repos.ISOUpload)
	networkHandler := handlers.NewVirtualNetworkHandler(repos.VirtualNetwork, libvirtClient)
	storageHandler := handlers.NewStorageHandler(repos, libvirtClient)
	backupHandler := handlers.NewBackupHandler(repos, backupService)

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
			vms.GET("/:id/hotplug", vmHandler.GetHotplugStatus)
			vms.POST("/:id/hotplug/cpu", vmHandler.HotplugCPU)
			vms.POST("/:id/hotplug/memory", vmHandler.HotplugMemory)
			vms.POST("/:id/sync", vmHandler.SyncVMStatus)

			vms.GET("/statuses", vmHandler.GetAllVMStatuses)

			snapshots := vms.Group("/:id/snapshots")
			{
				snapshots.POST("", snapshotHandler.CreateSnapshot)
				snapshots.GET("", snapshotHandler.ListSnapshots)
				snapshots.POST("/sync", snapshotHandler.SyncSnapshots)
				snapshots.GET("/:snapshot_id", snapshotHandler.GetSnapshot)
				snapshots.POST("/:snapshot_id/restore", snapshotHandler.RestoreSnapshot)
				snapshots.DELETE("/:snapshot_id", snapshotHandler.DeleteSnapshot)
			}

			batch := vms.Group("/batch")
			{
				batch.POST("/start", batchHandler.BatchStart)
				batch.POST("/stop", batchHandler.BatchStop)
				batch.POST("/force-stop", batchHandler.BatchStop)
				batch.DELETE("", batchHandler.BatchDelete)
				batch.POST("/:operation", batchHandler.BatchOperation)
			}

			backups := vms.Group("/:id/backups")
			{
				backups.GET("", backupHandler.ListBackups)
				backups.POST("", backupHandler.CreateBackup)
				backups.GET("/:backup_id", backupHandler.GetBackup)
				backups.DELETE("/:backup_id", backupHandler.DeleteBackup)
				backups.POST("/:backup_id/restore", backupHandler.RestoreBackup)

				schedules := backups.Group("/schedules")
				{
					schedules.GET("", backupHandler.ListSchedules)
					schedules.POST("", backupHandler.CreateSchedule)
					schedules.PUT("/:schedule_id", backupHandler.UpdateSchedule)
					schedules.DELETE("/:schedule_id", backupHandler.DeleteSchedule)
					schedules.POST("/:schedule_id/toggle", backupHandler.ToggleSchedule)
				}
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
			admin.GET("/audit-logs/export", auditHandler.ExportAuditLogsCSV)
			admin.GET("/system/resources", adminHandler.GetSystemResources(libvirtClient))
			admin.GET("/system/stats", adminHandler.GetSystemStats(libvirtClient))
			admin.GET("/system/storage", statsHandler.GetStorageStats)

			networks := admin.Group("/networks")
			{
				networks.GET("", networkHandler.List)
				networks.GET("/:id", networkHandler.Get)
				networks.POST("", networkHandler.Create)
				networks.PUT("/:id", networkHandler.Update)
				networks.DELETE("/:id", networkHandler.Delete)
				networks.POST("/:id/start", networkHandler.Start)
				networks.POST("/:id/stop", networkHandler.Stop)
			}

			storage := admin.Group("/storage")
			{
				storage.GET("/pools", storageHandler.ListPools)
				storage.GET("/pools/:id", storageHandler.GetPool)
				storage.POST("/pools", storageHandler.CreatePool)
				storage.PUT("/pools/:id", storageHandler.UpdatePool)
				storage.DELETE("/pools/:id", storageHandler.DeletePool)
				storage.POST("/pools/:id/start", storageHandler.StartPool)
				storage.POST("/pools/:id/stop", storageHandler.StopPool)
				storage.POST("/pools/:id/refresh", storageHandler.RefreshPool)
				storage.GET("/pools/:id/volumes", storageHandler.ListVolumes)
				storage.POST("/pools/:id/volumes", storageHandler.CreateVolume)
				storage.DELETE("/pools/:id/volumes/:volume_id", storageHandler.DeleteVolume)
			}
		}
	}
}
