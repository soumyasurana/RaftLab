package api

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

func (s *Server) handleHealth(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	health, err := s.manager.Health(userContext(c))
	if err != nil {
		return internalError("failed to load health", err)
	}

	health.BuildVersion = s.buildVersion

	return c.JSON(health)
}

func (s *Server) handleStatus(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	status, err := s.manager.Status(userContext(c))
	if err != nil {
		return internalError("failed to load status", err)
	}

	return c.JSON(status)
}

func (s *Server) handlePeers(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	peers, err := s.manager.Peers(userContext(c))
	if err != nil {
		return internalError("failed to load peers", err)
	}

	return c.JSON(peers)
}

func (s *Server) handleState(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	state, err := s.manager.StateMachineSnapshot(userContext(c))
	if err != nil {
		return internalError("failed to load state machine snapshot", err)
	}

	if state == nil {
		state = map[string]string{}
	}

	return c.JSON(state)
}

func (s *Server) handleMetrics(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	metrics, err := s.manager.Metrics(userContext(c))
	if err != nil {
		return internalError("failed to load metrics", err)
	}

	return c.JSON(metrics)
}

func (s *Server) handleSnapshot(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	ctx := userContext(c)

	select {
	case <-ctx.Done():
		return badRequest("request canceled")
	default:
	}

	snapshot, err := s.manager.TriggerSnapshot(ctx)
	if err != nil {
		return snapshotUnavailable("snapshot trigger failed")
	}

	return c.JSON(snapshot)
}

func (s *Server) handleChaosEnable(c fiber.Ctx) error {
	return s.handleChaosAction(c, func(ctx context.Context) error { return s.manager.EnableChaos(ctx) })
}

func (s *Server) handleChaosDisable(c fiber.Ctx) error {
	return s.handleChaosAction(c, func(ctx context.Context) error { return s.manager.DisableChaos(ctx) })
}

func (s *Server) handleChaosReset(c fiber.Ctx) error {
	return s.handleChaosAction(c, func(ctx context.Context) error { return s.manager.ResetChaos(ctx) })
}

func (s *Server) handleChaosAction(c fiber.Ctx, fn func(context.Context) error) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	if err := fn(userContext(c)); err != nil {
		return internalError("chaos operation failed", err)
	}

	return c.JSON(ActionResponse{Success: true})
}

func userContext(c fiber.Ctx) context.Context {
	ctx := c.Context()
	if ctx == nil {
		return context.Background()
	}

	return ctx
}
