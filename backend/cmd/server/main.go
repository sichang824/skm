package main

import (
	"backend-go/internal/config"
	"backend-go/internal/http/handlers"
	"backend-go/internal/http/middleware"
	"backend-go/internal/platform/db"
	"backend-go/internal/platform/log"
	"backend-go/internal/service"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const Version = "1.0.0"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Initialize logger
	logger, err := log.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}
	defer func(l *zap.Logger) { _ = l.Sync() }(logger)

	logger.Info("starting server",
		zap.String("version", Version),
		zap.String("db_driver", cfg.DBDriver),
		zap.String("port", cfg.Port),
	)

	// Open database connection
	gdb, err := db.Open(db.Config{
		Driver:  cfg.DBDriver,
		DSN:     cfg.DBDSN,
		LogMode: "silent",
	})
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}

	// Initialize Gin router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLogger(logger))

	// Health check
	healthHandler := handlers.NewHealthHandler(Version)
	r.GET("/healthz", healthHandler.Healthz)
	r.GET("/version", healthHandler.Version)

	catalogService := service.NewCatalogService(gdb)
	scanService := service.NewScanService(gdb)

	dashboardHandler := handlers.NewDashboardHandler(catalogService)
	providerHandler := handlers.NewProviderHandler(catalogService, scanService)
	skillHandler := handlers.NewSkillHandler(catalogService)
	scanHandler := handlers.NewScanHandler(catalogService, scanService)

	api := r.Group("/api")
	{
		api.GET("/dashboard", dashboardHandler.Get)
		api.GET("/providers", providerHandler.List)
		api.POST("/providers", providerHandler.Create)
		api.GET("/providers/:zid", providerHandler.Get)
		api.PUT("/providers/:zid", providerHandler.Update)
		api.DELETE("/providers/:zid", providerHandler.Delete)
		api.POST("/providers/:zid/scan", providerHandler.Scan)
		api.POST("/scan", scanHandler.ScanAll)
		api.GET("/scan-jobs", scanHandler.ListJobs)
		api.GET("/scan-jobs/:zid", scanHandler.GetJob)
		api.GET("/issues", scanHandler.ListIssues)
		api.GET("/conflicts", scanHandler.ListConflicts)
		api.GET("/skills", skillHandler.List)
		api.GET("/skills/:zid", skillHandler.Get)
		api.GET("/skills/:zid/files", skillHandler.Files)
		api.GET("/skills/:zid/file-content", skillHandler.FileContent)
	}

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("server listening", zap.String("addr", addr))

	if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
		logger.Fatal("server error", zap.Error(err))
	}
}
