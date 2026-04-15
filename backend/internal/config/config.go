package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	LogLevel  string
	LogFormat string
	DBDriver  string
	DBDSN     string
	Seed      bool
	SeedOnly  bool
}

func Load() (*Config, error) {
	loadEnvFiles()

	port := getEnv("PORT", "8080")
	logLevel := getEnv("LOG_LEVEL", "info")
	logFormat := getEnv("LOG_FORMAT", "console")
	dbDriver := getEnv("DB_DRIVER", "sqlite")
	dsn, err := resolveDSN(dbDriver, getEnv("DB_DSN", defaultDBDSN()))
	if err != nil {
		return nil, err
	}
	seed := getEnvBool("SEED", false)
	seedOnly := getEnvBool("SEED_ONLY", false)

	return &Config{
		Port:      port,
		LogLevel:  logLevel,
		LogFormat: logFormat,
		DBDriver:  dbDriver,
		DBDSN:     dsn,
		Seed:      seed,
		SeedOnly:  seedOnly,
	}, nil
}

func loadEnvFiles() {
	paths := []string{".env", "backend/.env"}
	if root := projectRootFromExecutable(); root != "" {
		paths = append(paths,
			filepath.Join(root, ".env"),
			filepath.Join(root, "backend", ".env"),
		)
	}
	_ = godotenv.Load(paths...)
}

func defaultDBDSN() string {
	candidates := []string{"./data/app.db", "./backend/data/app.db"}
	for _, candidate := range candidates {
		dir := filepath.Dir(candidate)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return candidate
		}
	}

	if appSupportPath, err := defaultAppSupportDBPath(); err == nil {
		return appSupportPath
	}

	return "./data/app.db"
}

func resolveDSN(driver, dsn string) (string, error) {
	if driver != "sqlite" {
		return dsn, nil
	}

	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" || trimmed == ":memory:" || strings.HasPrefix(trimmed, "file:") {
		return trimmed, nil
	}

	if !filepath.IsAbs(trimmed) {
		baseDir := sqliteBaseDir(trimmed)
		trimmed = filepath.Join(baseDir, trimmed)
	}

	if err := os.MkdirAll(filepath.Dir(trimmed), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}

	return trimmed, nil
}

func sqliteBaseDir(dsn string) string {
	for _, candidate := range []string{"./backend/data", "./data"} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return "."
		}
	}

	if root := projectRootFromExecutable(); root != "" {
		for _, candidate := range []string{
			filepath.Join(root, "backend", "data"),
			filepath.Join(root, "data"),
		} {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return root
			}
		}
	}

	if appSupportPath, err := defaultAppSupportDBPath(); err == nil {
		return filepath.Dir(appSupportPath)
	}

	return "."
}

func projectRootFromExecutable() string {
	executable, err := os.Executable()
	if err != nil {
		return ""
	}

	dir := filepath.Dir(executable)
	for i := 0; i < 6; i++ {
		if dir == "." || dir == string(filepath.Separator) {
			break
		}
		if info, err := os.Stat(filepath.Join(dir, "backend", "data")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

func defaultAppSupportDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	appDirName := "SKM"
	if runtime.GOOS == "darwin" {
		return filepath.Join(configDir, appDirName, "app.db"), nil
	}

	return filepath.Join(configDir, appDirName, "app.db"), nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
