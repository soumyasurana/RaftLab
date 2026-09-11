package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func main() {
	targetURL := flag.String("target", envOrDefault("RAFTLAB_API_URL", "http://localhost:8081"), "Target node management API URL")
	action := flag.String("action", "status", "Action to perform: status|enable|disable|reset|latency|packet-loss|partition|crash|restart|disconnect|reconnect|snapshot")
	nodeID := flag.String("node", "", "Target node ID for node-level chaos actions")
	minDelayMs := flag.Int64("min-delay", 10, "Minimum latency delay in milliseconds")
	maxDelayMs := flag.Int64("max-delay", 50, "Maximum latency delay in milliseconds")
	prob := flag.Float64("prob", 0.1, "Packet loss probability (0.0 - 1.0)")
	rawGroups := flag.String("groups", "", "Semicolon-separated partition groups, e.g. 'node1,node2;node3,node4,node5'")
	flag.Parse()

	url := strings.TrimRight(*targetURL, "/")

	switch *action {
	case "status":
		getAndPrint(url + "/chaos")
	case "enable":
		postAndPrint(url+"/chaos/enable", nil)
	case "disable":
		postAndPrint(url+"/chaos/disable", nil)
	case "reset":
		postAndPrint(url+"/chaos/reset", nil)
	case "latency":
		payload := map[string]int64{"minDelayMs": *minDelayMs, "maxDelayMs": *maxDelayMs}
		postAndPrint(url+"/chaos/latency", payload)
	case "packet-loss":
		payload := map[string]float64{"probability": *prob}
		postAndPrint(url+"/chaos/packet-loss", payload)
	case "partition":
		if *rawGroups == "" {
			fmt.Println("Error: -groups is required for partition action (e.g. -groups='node1,node2;node3,node4,node5')")
			os.Exit(1)
		}
		var groups [][]string
		for _, part := range strings.Split(*rawGroups, ";") {
			var group []string
			for _, item := range strings.Split(part, ",") {
				if trimmed := strings.TrimSpace(item); trimmed != "" {
					group = append(group, trimmed)
				}
			}
			if len(group) > 0 {
				groups = append(groups, group)
			}
		}
		payload := map[string]any{"groups": groups}
		postAndPrint(url+"/chaos/partition", payload)
	case "crash":
		requireNodeID(*nodeID)
		postAndPrint(url+"/chaos/node-failure", map[string]string{"nodeId": *nodeID})
	case "restart":
		requireNodeID(*nodeID)
		postAndPrint(url+"/chaos/node-restart", map[string]string{"nodeId": *nodeID})
	case "disconnect":
		requireNodeID(*nodeID)
		postAndPrint(url+"/chaos/node-disconnect", map[string]string{"nodeId": *nodeID})
	case "reconnect":
		requireNodeID(*nodeID)
		postAndPrint(url+"/chaos/node-reconnect", map[string]string{"nodeId": *nodeID})
	case "snapshot":
		postAndPrint(url+"/snapshot", nil)
	default:
		fmt.Printf("Unknown action: %s\n", *action)
		os.Exit(1)
	}
}

func requireNodeID(nodeID string) {
	if nodeID == "" {
		fmt.Println("Error: -node is required for this action")
		os.Exit(1)
	}
}

func getAndPrint(url string) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("HTTP GET failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, body, "", "  "); err == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(body))
	}
}

func postAndPrint(url string, payload any) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("Marshal request payload: %v\n", err)
			os.Exit(1)
		}
		body = bytes.NewReader(data)
	} else {
		body = bytes.NewReader([]byte("{}"))
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		fmt.Printf("Create HTTP request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("HTTP POST failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	resBody, _ := io.ReadAll(resp.Body)
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, resBody, "", "  "); err == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(resBody))
	}
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
