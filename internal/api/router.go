package api

import "github.com/gofiber/fiber/v3"

func registerRoutes(app *fiber.App, s *Server) {
	app.Get("/health", s.handleHealth)
	app.Get("/status", s.handleStatus)
	app.Get("/peers", s.handlePeers)
	app.Get("/state", s.handleState)
	app.Get("/metrics", s.handleMetrics)
	app.Get("/chaos", s.handleChaosStatus)

	app.Post("/snapshot", s.handleSnapshot)

	app.Post("/chaos/enable", s.handleChaosEnable)
	app.Post("/chaos/disable", s.handleChaosDisable)
	app.Post("/chaos/reset", s.handleChaosReset)
	app.Post("/chaos/latency", s.handleChaosLatency)
	app.Post("/chaos/packet-loss", s.handleChaosPacketLoss)
	app.Post("/chaos/partition", s.handleChaosPartition)
	app.Post("/chaos/node-failure", s.handleChaosNodeFailure)
	app.Post("/chaos/node-restart", s.handleChaosNodeRestart)
	app.Post("/chaos/node-disconnect", s.handleChaosNodeDisconnect)
	app.Post("/chaos/node-reconnect", s.handleChaosNodeReconnect)
}
