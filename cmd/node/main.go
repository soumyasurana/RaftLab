package main

import (
	"fmt"
	"github.com/raftlab/raftlab/internal/api"
	"github.com/raftlab/raftlab/internal/config"
	"github.com/raftlab/raftlab/internal/logger"
	"github.com/raftlab/raftlab/internal/raft"
	"github.com/raftlab/raftlab/internal/rpc"
)

func main() {
	fmt.Println("Starting Raft Node...")
	
	// TODO: load config
	cfg := config.LoadConfig()
	
	// TODO: initialize logger
	log := logger.NewLogger()
	_ = log
	
	// TODO: create raft node
	_ = raft.NewNode(cfg)
	
	// TODO: start grpc server
	go rpc.StartServer(cfg.RPCPort)
	
	// TODO: start http server
	go api.StartServer(cfg.HTTPPort)
	
	// TODO: block forever
	select {}
}
