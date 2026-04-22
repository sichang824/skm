package app

import (
	"backend-go/internal/config"
	"backend-go/internal/http/handlers"
	"backend-go/internal/http/middleware"
	"backend-go/internal/platform/db"
	logpkg "backend-go/internal/platform/log"
	seedpkg "backend-go/internal/platform/seed"
	"backend-go/internal/service"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const Version = "1.0.0"

type App struct {
	Config  *config.Config
	Logger  *zap.Logger
	Handler http.Handler
	server  *http.Server
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger, err := logpkg.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	logger.Info("starting server",
		zap.String("version", Version),
		zap.String("db_driver", cfg.DBDriver),
		zap.String("port", cfg.Port),
	)

	gdb, err := db.Open(db.Config{
		Driver:  cfg.DBDriver,
		DSN:     cfg.DBDSN,
		LogMode: "silent",
	})
	if err != nil {
		_ = logger.Sync()
		return nil, fmt.Errorf("open database: %w", err)
	}

	if cfg.Seed {
		result, err := seedpkg.SeedDefaultProviders(gdb)
		if err != nil {
			_ = logger.Sync()
			return nil, fmt.Errorf("seed default providers: %w", err)
		}
		logger.Info("default providers seeded",
			zap.Int("created", result.Created),
			zap.Int("existing", result.Existing),
			zap.Int("missing", result.Missing),
		)
		for _, message := range result.Messages {
			logger.Info("provider seed detail", zap.String("message", message))
		}
	}

	return &App{
		Config:  cfg,
		Logger:  logger,
		Handler: newRouter(gdb, logger),
		server:  &http.Server{Addr: fmt.Sprintf(":%s", cfg.Port)},
	}, nil
}

func (app *App) Run() error {
	if app.Config.SeedOnly {
		app.Logger.Info("seed-only mode complete, exiting")
		return nil
	}

	app.server.Handler = app.Handler
	app.Logger.Info("server listening", zap.String("addr", app.server.Addr))

	err := app.server.ListenAndServe()
	if err == nil || err == http.ErrServerClosed {
		return nil
	}

	return err
}

func (app *App) Close() error {
	if app.Logger != nil {
		return app.Logger.Sync()
	}
	return nil
}

func newRouter(gdb *gorm.DB, logger *zap.Logger) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestLogger(logger))

	healthHandler := handlers.NewHealthHandler(Version)
	r.GET("/healthz", healthHandler.Healthz)
	r.GET("/version", healthHandler.Version)

	catalogService := service.NewCatalogService(gdb)
	scanService := service.NewScanService(gdb)
	desktopService := service.NewDesktopService()

	dashboardHandler := handlers.NewDashboardHandler(catalogService)
	providerHandler := handlers.NewProviderHandler(catalogService, scanService)
	skillHandler := handlers.NewSkillHandler(catalogService, scanService)
	scanHandler := handlers.NewScanHandler(catalogService, scanService)
	desktopHandler := handlers.NewDesktopHandler(desktopService)

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
		api.GET("/desktop/cli", desktopHandler.CLIStatus)
		api.POST("/desktop/cli/install", desktopHandler.InstallCLI)
		api.GET("/skills", skillHandler.List)
		api.GET("/skills/:zid", skillHandler.Get)
		api.DELETE("/skills/:zid", skillHandler.Delete)
		api.POST("/skills/:zid/sync", skillHandler.Sync)
		api.GET("/skills/:zid/files", skillHandler.Files)
		api.GET("/skills/:zid/file-content", skillHandler.FileContent)
		api.POST("/skills/:zid/attach", skillHandler.Attach)
	}

	return r
}
