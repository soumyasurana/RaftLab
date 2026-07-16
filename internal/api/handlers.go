package api

import (
	"context"
	"encoding/json"
	"time"

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

func (s *Server) handleChaosStatus(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	chaos, err := s.manager.ChaosStatus(userContext(c))
	if err != nil {
		return internalError("failed to load chaos status", err)
	}

	return c.JSON(chaos)
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

func (s *Server) handleChaosLatency(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	var req ChaosLatencyRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return badRequest("invalid latency request body")
	}

	minDelay := time.Duration(req.MinDelayMs) * time.Millisecond
	maxDelay := time.Duration(req.MaxDelayMs) * time.Millisecond

	if err := s.manager.SetChaosLatency(userContext(c), minDelay, maxDelay); err != nil {
		return internalError("failed to update chaos latency", err)
	}

	return c.JSON(ActionResponse{Success: true})
}

func (s *Server) handleChaosPacketLoss(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	var req ChaosPacketLossRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return badRequest("invalid packet loss request body")
	}

	if err := s.manager.SetChaosPacketLoss(userContext(c), req.Probability); err != nil {
		return internalError("failed to update packet loss", err)
	}

	return c.JSON(ActionResponse{Success: true})
}

func (s *Server) handleChaosPartition(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	var req ChaosPartitionRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return badRequest("invalid partition request body")
	}

	if err := s.manager.SetChaosPartition(userContext(c), req.Groups); err != nil {
		return internalError("failed to update partitions", err)
	}

	return c.JSON(ActionResponse{Success: true})
}

func (s *Server) handleChaosNodeFailure(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	var req ChaosNodeRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return badRequest("invalid node failure request body")
	}

	if req.NodeID == "" {
		return badRequest("nodeId is required")
	}

	if err := s.manager.CrashNode(userContext(c), req.NodeID); err != nil {
		return internalError("failed to crash node", err)
	}

	return c.JSON(ActionResponse{Success: true})
}

func (s *Server) handleChaosNodeRestart(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	var req ChaosNodeRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return badRequest("invalid node restart request body")
	}

	if req.NodeID == "" {
		return badRequest("nodeId is required")
	}

	if err := s.manager.RestartNode(userContext(c), req.NodeID); err != nil {
		return internalError("failed to restart node", err)
	}

	return c.JSON(ActionResponse{Success: true})
}

func (s *Server) handleChaosNodeDisconnect(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	var req ChaosNodeRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return badRequest("invalid node disconnect request body")
	}

	if req.NodeID == "" {
		return badRequest("nodeId is required")
	}

	if err := s.manager.DisconnectNode(userContext(c), req.NodeID); err != nil {
		return internalError("failed to disconnect node", err)
	}

	return c.JSON(ActionResponse{Success: true})
}

func (s *Server) handleChaosNodeReconnect(c fiber.Ctx) error {
	if err := s.ensureManager(); err != nil {
		return internalError(err.Error(), err)
	}

	var req ChaosNodeRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return badRequest("invalid node reconnect request body")
	}

	if req.NodeID == "" {
		return badRequest("nodeId is required")
	}

	if err := s.manager.ReconnectNode(userContext(c), req.NodeID); err != nil {
		return internalError("failed to reconnect node", err)
	}

	return c.JSON(ActionResponse{Success: true})
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
