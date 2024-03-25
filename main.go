// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	maxRetries = 5
	sleepTime  = 10 * time.Second
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

type DpServiceVirtsvcPort struct {
	Value map[string]int `json:"/dp_service/virtsvc/used_port_count"`
}

//type Commands struct {
//	Value []string `json:"/"`
//}

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

	respBytes := make([]byte, 1024*6)
	var responseBuffer bytes.Buffer
	for {
		n, err := conn.Read(respBytes)
		if err != nil {
			log.Fatalf("Failed to read response from %s: %v", socketPath, err)
		}
		responseBuffer.Write(respBytes[:n])
		parts := strings.SplitN(command, ",", 2)
		command = parts[0]
		if bytes.Contains(respBytes, []byte(command)) {
			break
		}
	}

	err = json.Unmarshal(responseBuffer.Bytes(), response)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON response: %v", err)
	}
	//log.Println(response)
}

func updateMetrics(conn net.Conn, hostname string) {
	//var commands Commands
	//queryTelemetry(conn, "/", &commands)
	//log.Println(commands)

	var ethdevList EthdevList
	queryTelemetry(conn, "/ethdev/list", &ethdevList)
	//log.Println("ethdevList", ethdevList)

	for _, id := range ethdevList.Value {
		var ethdevInfo EthdevInfo
		queryTelemetry(conn, fmt.Sprintf("/ethdev/info,%d", id), &ethdevInfo)
		//log.Println("ethdevInfo", ethdevInfo)

		var ethdevXstats EthdevXstats
		queryTelemetry(conn, fmt.Sprintf("/ethdev/xstats,%d", id), &ethdevXstats)
		//log.Println("ethdevXstats", ethdevXstats)

		for statName, statValueFloat := range ethdevXstats.Value {
			promMetrics.With(prometheus.Labels{"interface": ethdevInfo.Value.Name, "stat_name": statName}).Set(statValueFloat)
		}
	}
	var dpserviceNatPort DpServiceNatPort
	queryTelemetry(conn, "/dp_service/nat/used_port_count", &dpserviceNatPort)
	//log.Println("Dpservice nat port", dpserviceNatPort)
	for ifName, portCount := range dpserviceNatPort.Value {
		promMetrics.With(prometheus.Labels{"interface": ifName, "stat_name": "nat_used_port_count"}).Set(float64(portCount))
	}

	var dpserviceVirtsvcPort DpServiceVirtsvcPort
	queryTelemetry(conn, "/dp_service/virtsvc/used_port_count", &dpserviceVirtsvcPort)
	//log.Println("Dpservice virtsvc port", dpserviceVirtsvcPort)
	for ifName, portCount := range dpserviceVirtsvcPort.Value {
		promMetrics.With(prometheus.Labels{"interface": ifName, "stat_name": "virtsvc_used_port_count"}).Set(float64(portCount))
	}

	var dpserviceCallCount DpServiceGraphCallCount
	queryTelemetry(conn, "/dp_service/graph/call_count", &dpserviceCallCount)
	//log.Println("dpserviceCallCount", dpserviceCallCount)

	for graphNodeName, callCount := range dpserviceCallCount.GraphCallCnt.Node_0_to_255 {
		promMetricsCallCount.With(prometheus.Labels{"node_name": hostname, "graph_node": graphNodeName}).Set(callCount)
	}
}

func main() {
	var conn net.Conn
	var err error

	r := prometheus.NewRegistry()
	r.MustRegister(promMetrics)
	r.MustRegister(promMetricsCallCount)

	http.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))

	for i := 0; i < maxRetries; i++ {
		conn, err = net.Dial("unixpacket", socketPath)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to %s: %v. Retry %d of %d", socketPath, err, i+1, maxRetries)
		if i < maxRetries-1 {
			time.Sleep(sleepTime)
		}
	}
	defer conn.Close()
	var host string
	hostnameFlag := flag.String("hostname", "", "Hostname to use")
	pollIntervalFlag := flag.Int("poll-interval", 20, "Polling interval in seconds")
	flag.Parse()

	if *hostnameFlag == "" {
		// Try to get hostname from environment variable
		envHostName := os.Getenv("NODE_NAME")
		if envHostName != "" {
			host = envHostName
		} else {
			// If environment variable not set, get hostname from os.Hostname
			host, err = os.Hostname()
			if err != nil {
				fmt.Printf("Error retrieving hostname: %v\n", err)
			}
		}
	} else {
		host = *hostnameFlag
	}
	fmt.Printf("Hostname: %s\n", host)

	flushSocket(conn)
	go func() {
		for {
			updateMetrics(conn, host)
			time.Sleep(time.Duration(*pollIntervalFlag) * time.Second)
		}
	}()

	log.Println("Server starting on :9064...")
	log.Fatal(http.ListenAndServe(":9064", nil))
}
