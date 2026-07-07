package main

import (
	"fmt"
	"log"

	"github.com/raftlab/raftlab/internal/config"
)

func main() {
	cfg, err := config.Load("deployments/configs/node1.yaml")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg.Node.ID)
	fmt.Println(cfg.Node.Address)
	fmt.Println(cfg.Node.Peers)
}
