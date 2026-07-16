package api

import "github.com/gofiber/fiber/v3"

func registerRoutes(app *fiber.App, s *Server) {
	app.Get("/health", s.handleHealth)
	app.Get("/status", s.handleStatus)
	app.Get("/peers", s.handlePeers)
	app.Get("/state", s.handleState)
	app.Get("/metrics", s.handleMetrics)

	app.Post("/snapshot", s.handleSnapshot)

	app.Post("/chaos/enable", s.handleChaosEnable)
	app.Post("/chaos/disable", s.handleChaosDisable)
	app.Post("/chaos/reset", s.handleChaosReset)
}
