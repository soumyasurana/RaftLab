package api

import "github.com/gofiber/fiber/v3"

// SetupRoutes registers all HTTP API endpoints on a Fiber app instance.
func SetupRoutes(app *fiber.App, s *Server) {
	registerRoutes(app, s)
}
