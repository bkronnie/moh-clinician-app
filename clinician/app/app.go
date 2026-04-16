package app

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/moh/clinician/internals/middleware"
	"github.com/moh/clinician/internals/routes"
	"github.com/moh/clinician/internals/utilities"

	_ "github.com/lib/pq"
)

type Config struct {
	Address      string `json:"Address"`
	ReadTimeout  int64  `json:"ReadTimeout"`
	WriteTimeout int64  `json:"WriteTimeout"`
	Static       string `json:"Static"`
	Ux           string `json:"Ux"`
	Px           string `json:"Px"`
	Dx           string `json:"Dx"`
}

func Run() error {
	root, err := findAppRoot()
	if err != nil {
		return err
	}

	utilities.SetAppRoot(root)

	if err := loadDotEnv(filepath.Join(root, ".env")); err != nil {
		return err
	}

	if err := utilities.SetLogFile(filepath.Join(root, "activity.log")); err != nil {
		return fmt.Errorf("configure activity log: %w", err)
	}

	config, err := loadConfig(root)
	if err != nil {
		return err
	}

	if config.Address == "" {
		config.Address = "0.0.0.0:8081"
	}

	connStr, err := buildConnStr(config)
	if err != nil {
		return err
	}

	schemaName := getenvDefault("DB_SCHEMA", "clinician_app")

	router := gin.Default()
	router.Static("/static", filepath.Join(root, "clinician", "ui", "static"))

	sessionManager := scs.New()
	sessionManager.Lifetime = 30 * time.Minute
	sessionManager.Cookie.Persist = true
	sessionManager.Cookie.Secure = false
	sessionManager.Cookie.HttpOnly = true

	router.Use(func(c *gin.Context) {
		sessionManager.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r.WithContext(r.Context())
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	})

	db, err := openDB(connStr)
	if err != nil {
		utilities.Danger("Failed to open database")
		return err
	}

	if err := verifySchema(db, schemaName); err != nil {
		utilities.Danger(err)
		return err
	}

	router.Use(middleware.RequestLogger())
	routes.SetupRoutes(router, db, sessionManager)

	utilities.Info("starting server on", config.Address)
	return router.Run(config.Address)
}

func findAppRoot() (string, error) {
	if root := os.Getenv("CLINICIAN_APP_ROOT"); root != "" {
		return filepath.Abs(root)
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}

	current := filepath.Dir(exePath)
	for {
		if dirExists(filepath.Join(current, "clinician", "ui", "html")) {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	current = wd
	for {
		if dirExists(filepath.Join(current, "clinician", "ui", "html")) {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", errors.New("could not locate app root containing clinician/ui/html")
}

func loadConfig(root string) (Config, error) {
	paths := []string{
		filepath.Join(root, "config.json"),
		filepath.Join(root, "clinician", "cmd", "web", "config.json"),
	}

	var config Config
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}

		defer file.Close()

		if err := json.NewDecoder(file).Decode(&config); err != nil {
			return Config{}, fmt.Errorf("decode config %s: %w", path, err)
		}

		applyEnvOverrides(&config)
		return config, nil
	}

	return Config{}, fmt.Errorf("config.json not found in %s or %s", paths[0], paths[1])
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid .env entry on line %d", lineNo)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)

		if key == "" {
			return fmt.Errorf("empty .env key on line %d", lineNo)
		}

		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set environment variable %s: %w", key, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read .env file: %w", err)
	}

	return nil
}

func applyEnvOverrides(config *Config) {
	if value := os.Getenv("APP_ADDRESS"); value != "" {
		config.Address = value
	}
	if value := os.Getenv("DB_USER"); value != "" {
		config.Ux = value
	}
	if value := os.Getenv("DB_PASSWORD"); value != "" {
		config.Px = value
	}
	if value := os.Getenv("DB_NAME"); value != "" {
		config.Dx = value
	}
}

func buildConnStr(config Config) (string, error) {
	host := getenvDefault("DB_HOST", "127.0.0.1")
	port := getenvDefault("DB_PORT", "5432")
	sslmode := getenvDefault("DB_SSLMODE", "disable")
	schema := getenvDefault("DB_SCHEMA", "clinician_app")
	user := strings.TrimSpace(config.Ux)
	password := config.Px
	dbname := strings.TrimSpace(config.Dx)

	if user == "" {
		return "", errors.New("database user is missing; set DB_USER in .env")
	}
	if dbname == "" {
		return "", errors.New("database name is missing; set DB_NAME in .env")
	}

	return fmt.Sprintf(
		"host=%s port=%s user=%s password='%s' dbname=%s sslmode=%s search_path=%s",
		host,
		port,
		user,
		password,
		dbname,
		sslmode,
		schema,
	), nil
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func openDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		utilities.Danger(err, "Cannot connect to db")
		return nil, err
	}

	if err = db.Ping(); err != nil {
		utilities.Danger(err, "Cannot reach db")
		return nil, err
	}

	return db, nil
}

func verifySchema(db *sql.DB, schema string) error {
	requiredTables := []string{"users", "employees", "facilities", "departments"}

	for _, table := range requiredTables {
		qualifiedName := fmt.Sprintf("%s.%s", schema, table)
		var regclass sql.NullString

		if err := db.QueryRow(
			"SELECT to_regclass($1)",
			qualifiedName,
		).Scan(&regclass); err != nil {
			return fmt.Errorf("verify required table %s: %w", qualifiedName, err)
		}

		if !regclass.Valid || regclass.String == "" {
			return fmt.Errorf(
				"required table %s was not found; run cliniciandb/moh-clinician-app-db.sql and then cliniciandb/moh-clinician-app-seed.sql against the database configured in .env",
				qualifiedName,
			)
		}
	}

	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
