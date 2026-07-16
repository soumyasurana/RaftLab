package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
)

const DefaultBuildVersion = "dev"

// Server owns the Fiber app and routes for the management API.
type Server struct {
	app          *fiber.App
	manager      Manager
	buildVersion string
	startedAt    time.Time
}

// Option customizes server construction.
type Option func(*Server)

// WithBuildVersion overrides the build version returned by /health.
func WithBuildVersion(version string) Option {
	return func(s *Server) {
		if version != "" {
			s.buildVersion = version
		}
	}
}

// NewServer creates a management API server.
func NewServer(manager Manager, opts ...Option) *Server {
	s := &Server{
		manager:      manager,
		buildVersion: DefaultBuildVersion,
		startedAt:    time.Now().UTC(),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.app = fiber.New(fiber.Config{
		AppName:       "RaftLab Management API",
		ErrorHandler:  s.errorHandler,
		ServerHeader:  "RaftLab",
		StrictRouting: true,
	})

	s.app.Use(requestMetadataMiddleware())
	s.app.Use(recoveryMiddleware())

	registerRoutes(s.app, s)

	return s
}

// App returns the underlying Fiber application.
func (s *Server) App() *fiber.App {
	return s.app
}

// Listen starts the HTTP server.
func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}

func (s *Server) errorHandler(c fiber.Ctx, err error) error {
	status := statusFromError(err)

	return c.Status(status).JSON(ErrorResponse{
		Error: messageFromError(err),
	})
}

func (s *Server) ensureManager() error {
	if s.manager == nil {
		return fmt.Errorf("management interface is nil")
	}
	return nil
}
