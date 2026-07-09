package rpc

import (
	"context"
	"fmt"
	"sync"

	pb "github.com/soumyasurana/RaftLab/internal/pb/raft"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client manages persistent gRPC connections to Raft peers.
type Client struct {
	mu sync.RWMutex

	connections map[string]*grpc.ClientConn
	clients     map[string]pb.RaftServiceClient
}

// NewClient creates an empty Raft RPC client.
func NewClient() *Client {
	return &Client{
		connections: make(map[string]*grpc.ClientConn),
		clients:     make(map[string]pb.RaftServiceClient),
	}
}

/*
Connect creates and stores a gRPC connection for a peer.

Calling Connect more than once for the same peer replaces the previous
connection.
*/
func (c *Client) Connect(peerID string, address string) error {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf(
			"create gRPC client for peer %s at %s: %w",
			peerID,
			address,
			err,
		)
	}

	client := pb.NewRaftServiceClient(conn)

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.connections[peerID]; ok {
		if err := existing.Close(); err != nil {
			_ = conn.Close()

			return fmt.Errorf(
				"close existing connection for peer %s: %w",
				peerID,
				err,
			)
		}
	}

	c.connections[peerID] = conn
	c.clients[peerID] = client

	return nil
}

// RequestVote sends a RequestVote RPC to a peer.
func (c *Client) RequestVote(
	ctx context.Context,
	peerID string,
	req *pb.RequestVoteRequest,
) (*pb.RequestVoteResponse, error) {
	client, err := c.peerClient(peerID)
	if err != nil {
		return nil, err
	}

	response, err := client.RequestVote(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(
			"request vote from peer %s: %w",
			peerID,
			err,
		)
	}

	return response, nil
}

// AppendEntries sends an AppendEntries RPC to a peer.
func (c *Client) AppendEntries(
	ctx context.Context,
	peerID string,
	req *pb.AppendEntriesRequest,
) (*pb.AppendEntriesResponse, error) {
	client, err := c.peerClient(peerID)
	if err != nil {
		return nil, err
	}

	response, err := client.AppendEntries(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(
			"append entries to peer %s: %w",
			peerID,
			err,
		)
	}

	return response, nil
}

// Close closes every peer connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error

	for peerID, conn := range c.connections {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf(
				"close connection for peer %s: %w",
				peerID,
				err,
			)
		}
	}

	c.connections = make(map[string]*grpc.ClientConn)
	c.clients = make(map[string]pb.RaftServiceClient)

	return firstErr
}

// peerClient returns the generated gRPC client for a connected peer.
func (c *Client) peerClient(
	peerID string,
) (pb.RaftServiceClient, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	client, ok := c.clients[peerID]
	if !ok {
		return nil, fmt.Errorf(
			"peer %s is not connected",
			peerID,
		)
	}

	return client, nil
}
