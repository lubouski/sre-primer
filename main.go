package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Global variables
var (
	db *sql.DB

	// Prometheus metrics
	appReadiness = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "app_ready",
		Help: "Indicates if the application is ready to receive traffic (1 for ready, 0 for not ready)",
	})

	dbConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_connections_active",
		Help: "Number of active database connections",
	})

	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Count of HTTP requests by endpoint and status",
		},
		[]string{"endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// Regular expression to validate username (only letters)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z]+$`)

	// Table name from environment variable
	dbTableName string

	// Logger
	logger *slog.Logger
)

// User represents the user data with date of birth
type User struct {
	DateOfBirth string `json:"dateOfBirth"`
}

func main() {
	// Initialize structured logger
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})
	logger = slog.New(logHandler)
	slog.SetDefault(logger)

	logger.Info("Starting application")

	// Set the application as not ready initially
	appReadiness.Set(0)

	// Initialize database connection
	var err error
	connStr := getDBConnectionString()
	logger.Info("Connecting to database", "connection", hidePassword(connStr))

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		logger.Error("Failed to open database connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Get table name from environment variable
	dbTableName = getEnv("DB_TABLE", "users")
	logger.Info("Using database table", "table", dbTableName)

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Start a background goroutine to check database connectivity
	go monitorDBConnectivity()

	// Create router using standard HTTP mux (Go 1.22+)
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("PUT /hello/{username}", putUserHandler)
	mux.HandleFunc("GET /hello/{username}", getUserHandler)

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Health endpoint
	mux.HandleFunc("GET /health", healthCheckHandler)
	mux.HandleFunc("GET /readiness", readinessCheckHandler)

	// Start the server
	port := getEnv("PORT", "8080")

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so it doesn't block
	go func() {
		logger.Info("Starting server", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Setup signal catching
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig

	logger.Info("Received shutdown signal", "signal", s)
	logger.Info("Shutting down gracefully...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt to gracefully shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited")
}

// loggingMiddleware wraps all requests with logging
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response wrapper to capture status code
		wrapper := newResponseWriter(w)

		// Process the request
		next.ServeHTTP(wrapper, r)

		// Log after request is processed
		duration := time.Since(start)
		logger.Info("Request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapper.status,
			"duration", duration,
			"user_agent", r.UserAgent(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// responseWriter is a wrapper for http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

// newResponseWriter creates a new responseWriter
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

// WriteHeader captures the status before writing it
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// hidePassword masks the password in connection strings for logging
func hidePassword(connStr string) string {
	re := regexp.MustCompile(`password=([^\\s]+)`)
	return re.ReplaceAllString(connStr, "password=*****")
}

// getDBConnectionString constructs the PostgreSQL connection string
func getDBConnectionString() string {
	// These should be provided via environment variables in production
	host := getEnv("DB_HOST", "pgdb")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "hello_service_db_user")
	password := getEnv("DB_PASSWORD", "replace_me")
	dbname := getEnv("DB_NAME", "hello_service_db")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
}

// getEnv retrieves environment variables with a fallback value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// monitorDBConnectivity periodically checks database connectivity and updates metrics
func monitorDBConnectivity() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		// Check database connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		err := db.PingContext(ctx)
		cancel()

		// Update stats about DB connections
		stats := db.Stats()
		dbConnections.Set(float64(stats.InUse))

		if err != nil {
			logger.Warn("Database connection check failed", "error", err)
			appReadiness.Set(0) // Set app as not ready
		} else {
			appReadiness.Set(1) // Set app as ready
		}
	}
}

// healthCheckHandler returns 200 OK if the server is running
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// readinessCheckHandler checks if the application is ready to serve traffic
func readinessCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := db.PingContext(ctx)
	if err != nil {
		logger.Warn("Readiness check failed", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Database connection not available"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

// validateUsername checks if the username contains only letters
func validateUsername(username string) bool {
	return usernameRegex.MatchString(username)
}

// validateDateOfBirth checks if the date is in the past
func validateDateOfBirth(dateStr string) error {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("invalid date format. Use YYYY-MM-DD")
	}

	// Check if date is in the future
	if date.After(time.Now()) {
		return fmt.Errorf("date of birth cannot be in the future")
	}

	return nil
}

// putUserHandler handles the PUT request to store a user's date of birth
func putUserHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	endpoint := "/hello/{username}"
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Get context for logging
	ctx := context.WithValue(r.Context(), "requestID", requestID)
	logCtx := logger.With("requestID", requestID, "endpoint", endpoint, "method", "PUT")

	// Record metrics when the function completes
	defer func() {
		httpRequestDuration.WithLabelValues(endpoint).Observe(time.Since(startTime).Seconds())
	}()

	// Check if the database is ready
	if !isDatabaseReady() {
		logCtx.Warn("Database not ready, rejecting request")
		httpRequestsTotal.WithLabelValues(endpoint, "503").Inc()
		http.Error(w, "Service unavailable: database connection not ready", http.StatusServiceUnavailable)
		return
	}

	// Get username from the path parameter
	username := r.PathValue("username")
	logCtx = logCtx.With("username", username)

	if username == "" {
		logCtx.Warn("Missing username parameter")
		httpRequestsTotal.WithLabelValues(endpoint, "400").Inc()
		http.Error(w, "Missing username parameter", http.StatusBadRequest)
		return
	}

	// Validate username (only letters)
	if !validateUsername(username) {
		logCtx.Warn("Invalid username format", "username", username)
		httpRequestsTotal.WithLabelValues(endpoint, "400").Inc()
		http.Error(w, "Username must contain only letters", http.StatusBadRequest)
		return
	}

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		logCtx.Warn("Invalid request body", "error", err)
		httpRequestsTotal.WithLabelValues(endpoint, "400").Inc()
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logCtx = logCtx.With("dateOfBirth", user.DateOfBirth)

	// Validate date of birth
	if err := validateDateOfBirth(user.DateOfBirth); err != nil {
		logCtx.Warn("Invalid date of birth", "error", err)
		httpRequestsTotal.WithLabelValues(endpoint, "400").Inc()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Store in database
	query := fmt.Sprintf(`
    	INSERT INTO %s (username, date_of_birth) 
    	VALUES ($1, $2) 
    	ON CONFLICT (username) 
    	DO UPDATE SET date_of_birth = $2
	`, dbTableName)

	_, err = db.ExecContext(ctx, query, username, user.DateOfBirth)
	if err != nil {
		logCtx.Error("Database error when storing user data", "error", err)
		httpRequestsTotal.WithLabelValues(endpoint, "500").Inc()
		http.Error(w, "Error storing data", http.StatusInternalServerError)
		return
	}

	// Success
	logCtx.Info("Successfully stored user data")
	httpRequestsTotal.WithLabelValues(endpoint, "204").Inc()
	w.WriteHeader(http.StatusNoContent)
}

// getUserHandler handles the GET request to retrieve a user's birthday greeting
func getUserHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	endpoint := "/hello/{username}"
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Get context for logging
	ctx := context.WithValue(r.Context(), "requestID", requestID)
	logCtx := logger.With("requestID", requestID, "endpoint", endpoint, "method", "GET")

	// Record metrics when the function completes
	defer func() {
		httpRequestDuration.WithLabelValues(endpoint).Observe(time.Since(startTime).Seconds())
	}()

	// Check if the database is ready
	if !isDatabaseReady() {
		logCtx.Warn("Database not ready, rejecting request")
		httpRequestsTotal.WithLabelValues(endpoint, "503").Inc()
		http.Error(w, "Service unavailable: database connection not ready", http.StatusServiceUnavailable)
		return
	}

	// Get username from the path parameter
	username := r.PathValue("username")
	logCtx = logCtx.With("username", username)

	if username == "" {
		logCtx.Warn("Missing username parameter")
		httpRequestsTotal.WithLabelValues(endpoint, "400").Inc()
		http.Error(w, "Missing username parameter", http.StatusBadRequest)
		return
	}

	// Validate username (only letters)
	if !validateUsername(username) {
		logCtx.Warn("Invalid username format", "username", username)
		httpRequestsTotal.WithLabelValues(endpoint, "400").Inc()
		http.Error(w, "Username must contain only letters", http.StatusBadRequest)
		return
	}

	// Query database
	var dateOfBirth string
	query := fmt.Sprintf("SELECT date_of_birth FROM %s WHERE username = $1", dbTableName)
	err := db.QueryRowContext(ctx, query, username).Scan(&dateOfBirth)
	if err != nil {
		if err == sql.ErrNoRows {
			logCtx.Info("User not found", "username", username)
			httpRequestsTotal.WithLabelValues(endpoint, "404").Inc()
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			logCtx.Error("Database error when retrieving user data", "error", err)
			httpRequestsTotal.WithLabelValues(endpoint, "500").Inc()
			http.Error(w, "Error retrieving data", http.StatusInternalServerError)
		}
		return
	}

	logCtx = logCtx.With("dateOfBirth", dateOfBirth)

	// Parse date of birth
	dob, err := time.Parse("2006-01-02T15:04:05Z", dateOfBirth)
	if err != nil {
		logCtx.Error("Invalid date format stored in database", "error", err)
		httpRequestsTotal.WithLabelValues(endpoint, "500").Inc()
		http.Error(w, "Invalid date stored", http.StatusInternalServerError)
		return
	}

	// Get current date
	today := time.Now()

	// Create birthday message
	message := createBirthdayMessage(username, dob, today)
	logCtx = logCtx.With("message", message)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	httpRequestsTotal.WithLabelValues(endpoint, "200").Inc()

	response := map[string]string{"message": message}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		logCtx.Error("Error encoding JSON response", "error", err)
		return
	}

	logCtx.Info("Successfully retrieved user data and generated greeting")
}

// createBirthdayMessage generates the appropriate birthday message
func createBirthdayMessage(username string, dob, today time.Time) string {
	// Get month and day for comparison
	dobMonth, dobDay := dob.Month(), dob.Day()
	todayMonth, todayDay := today.Month(), today.Day()

	// Check if today is the user's birthday
	if dobMonth == todayMonth && dobDay == todayDay {
		return fmt.Sprintf("Hello, %s! Happy birthday!", username)
	}

	// Calculate days until next birthday
	nextBirthday := time.Date(today.Year(), dobMonth, dobDay, 0, 0, 0, 0, time.UTC)

	// If the birthday has already occurred this year, use next year's date
	if nextBirthday.Before(today) {
		nextBirthday = time.Date(today.Year()+1, dobMonth, dobDay, 0, 0, 0, 0, time.UTC)
	}

	daysUntil := int(nextBirthday.Sub(today).Hours() / 24)

	return fmt.Sprintf("Hello, %s! Your birthday is in %d day(s)", username, daysUntil)
}

// isDatabaseReady checks if the database connection is ready
func isDatabaseReady() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := db.PingContext(ctx)
	return err == nil
}

