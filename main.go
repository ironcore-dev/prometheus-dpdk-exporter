package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	socketPath = "/var/run/dpdk/rte/dpdk_telemetry.v2"

	promMetrics = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dpdk_interface_stat",
			Help: "DPDK interface statistic",
		},
		[]string{"interface", "stat_name"},
	)

	promMetricsCallCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dpdk_graph_stat",
			Help: "Dp-Service graph statistics",
		},
		[]string{"node_name", "graph_node"},
	)
)

type EthdevList struct {
	Value []int `json:"/ethdev/list"`
}

type EthdevInfo struct {
	Value struct {
		Name string `json:"name"`
	} `json:"/ethdev/info"`
}

type EthdevXstats struct {
	Value map[string]float64 `json:"/ethdev/xstats"`
}

type DpServiceNatPort struct {
	Value map[string]int `json:"/dp_service/nat/used_port_count"`
}

type NodeData map[string]float64

type GraphCallCount struct {
	Node_0_to_255 NodeData `json:"Node_0_to_255"`
}

type DpServiceGraphCallCount struct {
	GraphCallCnt GraphCallCount `json:"/dp_service/graph/call_count"`
}

func flushSocket(conn net.Conn) {
	respBytes := make([]byte, 1024)

	_, err := conn.Read(respBytes)
	if err != nil {
		log.Fatalf("Failed to read response from %s: %v", socketPath, err)
	}
}

func queryTelemetry(conn net.Conn, command string, response interface{}) {
	_, err := conn.Write([]byte(command))
	if err != nil {
		log.Fatalf("Failed to send command to %s: %v", socketPath, err)
	}

	respBytes := make([]byte, 1024*4)
	var responseBuffer bytes.Buffer
	for {
		n, err := conn.Read(respBytes)
		if err != nil {
			log.Fatalf("Failed to read response from %s: %v", socketPath, err)
		}
		responseBuffer.Write(respBytes[:n])
		if bytes.Contains(respBytes, []byte(command[:len(command)-2])) {
			break
		}
	}

	log.Printf("Raw reply %s\n", responseBuffer.String())
	err = json.Unmarshal(responseBuffer.Bytes(), response)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON response: %v", err)
	}
}

func updateMetrics(conn net.Conn, hostname string) {
	log.Println("Updating metrics...")
	var ethdevList EthdevList
	queryTelemetry(conn, "/ethdev/list", &ethdevList)

	for _, id := range ethdevList.Value {
		var ethdevInfo EthdevInfo
		log.Printf("Interface info for id %d\n", id)
		queryTelemetry(conn, fmt.Sprintf("/ethdev/info,%d", id), &ethdevInfo)

		var ethdevXstats EthdevXstats
		queryTelemetry(conn, fmt.Sprintf("/ethdev/xstats,%d", id), &ethdevXstats)

		for statName, statValueFloat := range ethdevXstats.Value {
			promMetrics.With(prometheus.Labels{"interface": ethdevInfo.Value.Name, "stat_name": statName}).Set(statValueFloat)
		}
	}
	var dpserviceNatPort DpServiceNatPort
	queryTelemetry(conn, "/dp_service/nat/used_port_count", &dpserviceNatPort)

	for ifName, portCount := range dpserviceNatPort.Value {
		promMetrics.With(prometheus.Labels{"interface": ifName, "stat_name": "nat_used_port_count"}).Set(float64(portCount))
	}

	var dpserviceCallCount DpServiceGraphCallCount
	queryTelemetry(conn, "/dp_service/graph/call_count", &dpserviceCallCount)

	for graphNodeName, callCount := range dpserviceCallCount.GraphCallCnt.Node_0_to_255 {
		promMetricsCallCount.With(prometheus.Labels{"node_name": hostname, "graph_node": graphNodeName}).Set(callCount)
	}

	log.Println("Metrics update finished.")
}

func main() {
	r := prometheus.NewRegistry()
	r.MustRegister(promMetrics)
	r.MustRegister(promMetricsCallCount)

	http.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))

	conn, err := net.Dial("unixpacket", socketPath)
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", socketPath, err)
	}
	defer conn.Close()

	host, err := os.Hostname()
	if err != nil {
		fmt.Printf("Error retrieving hostname: %v\n", err)
	} else {
		fmt.Printf("Hostname: %s\n", host)
	}

	flushSocket(conn)
	go func() {
		for {
			updateMetrics(conn, host)
			time.Sleep(5 * time.Second)
		}
	}()

	log.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
