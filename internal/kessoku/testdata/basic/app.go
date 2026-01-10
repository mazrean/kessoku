package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

// App represents the main application.
type App struct {
	config      *Config
	userService *UserService
	logger      *slog.Logger
}

// NewApp creates a new application instance.
// wire: provider
func NewApp(config *Config, userService *UserService, logger *slog.Logger) *App {
	return &App{
		config:      config,
		userService: userService,
		logger:      logger,
	}
}

// Run starts the application.
func (a *App) Run() error {
	a.logger.Info("Starting application", "port", a.config.Port)

	http.HandleFunc("/users/", a.handleGetUser)

	addr := fmt.Sprintf(":%d", a.config.Port)
	return http.ListenAndServe(addr, nil)
}

// handleGetUser handles GET /users/{id} requests.
func (a *App) handleGetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/users/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := a.userService.GetUser(id)
	if err != nil {
		a.logger.Error("Failed to get user", "error", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if _, err := fmt.Fprintf(w, "User: %+v\n", user); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
