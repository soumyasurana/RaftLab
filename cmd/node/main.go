package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/soumyasurana/RaftLab/internal/api"
	"github.com/soumyasurana/RaftLab/internal/config"
	"github.com/soumyasurana/RaftLab/internal/raft"
	"github.com/soumyasurana/RaftLab/internal/rpc"
)

func main() {
	defaultConfig := os.Getenv("CONFIG_PATH")
	if defaultConfig == "" {
		defaultConfig = "deployments/configs/local/node1.yaml"
	}

	configPath := flag.String(
		"config",
		defaultConfig,
		"path to the node configuration file",
	)

	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	node, err := raft.New(cfg)
	if err != nil {
		log.Fatalf("create Raft node: %v", err)
	}

	apiAddr := os.Getenv("API_ADDRESS")
	if apiAddr == "" {
		apiAddr = ":8080"
	}

	buildVersion := os.Getenv("BUILD_VERSION")
	if buildVersion == "" {
		buildVersion = api.DefaultBuildVersion
	}

	managementAPI := api.NewServer(node, api.WithBuildVersion(buildVersion))

	rpcServer := rpc.NewServer(
		cfg.Node.Address,
		node,
	)

	serverErrCh := make(chan error, 1)
	apiErrCh := make(chan error, 1)

	go func() {
		log.Printf(
			"starting Raft gRPC server: node=%s address=%s",
			cfg.Node.ID,
			cfg.Node.Address,
		)

		serverErrCh <- rpcServer.Start()
	}()

	go func() {
		log.Printf(
			"starting management API: node=%s address=%s",
			cfg.Node.ID,
			apiAddr,
		)

		apiErrCh <- managementAPI.Listen(apiAddr)
	}()

	node.Start()

	log.Printf(
		"Raft node started: id=%s peers=%d",
		cfg.Node.ID,
		len(cfg.Node.Peers),
	)

	shutdownCh := make(chan os.Signal, 1)

	signal.Notify(
		shutdownCh,
		os.Interrupt,
		syscall.SIGTERM,
	)

	select {
	case signal := <-shutdownCh:
		log.Printf(
			"received shutdown signal: %s",
			signal,
		)

	case err := <-serverErrCh:
		if err != nil {
			log.Printf(
				"gRPC server stopped with error: %v",
				err,
			)
		}

	case err := <-apiErrCh:
		if err != nil {
			log.Printf(
				"management API stopped with error: %v",
				err,
			)
		}
	}

	log.Printf(
		"stopping Raft node: id=%s",
		cfg.Node.ID,
	)

	rpcServer.Stop()
	if err := managementAPI.Shutdown(); err != nil {
		log.Printf(
			"stop management API: %v",
			err,
		)
	}

	if err := node.Stop(); err != nil {
		log.Printf(
			"stop Raft node: %v",
			err,
		)
	}

	fmt.Println("Raft node stopped")
}
