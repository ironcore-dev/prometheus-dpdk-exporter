package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var SocketPath = "/var/run/dpdk/rte/dpdk_telemetry.v2"

func QueryTelemetry(conn net.Conn, command string, response interface{}) {
	_, err := conn.Write([]byte(command))
	if err != nil {
		log.Fatalf("Failed to send command to %s: %v", SocketPath, err)
	}

	respBytes := make([]byte, 1024*6)
	var responseBuffer bytes.Buffer
	for {
		n, err := conn.Read(respBytes)
		if err != nil {
			log.Fatalf("Failed to read response from %s: %v", SocketPath, err)
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
}

func Update(conn net.Conn, hostname string) {
	var ethdevList EthdevList
	QueryTelemetry(conn, "/ethdev/list", &ethdevList)
	//log.Println("ethdevList", ethdevList)

	for _, id := range ethdevList.Value {
		var ethdevInfo EthdevInfo
		QueryTelemetry(conn, fmt.Sprintf("/ethdev/info,%d", id), &ethdevInfo)
		//log.Println("ethdevInfo", ethdevInfo)

		var ethdevXstats EthdevXstats
		QueryTelemetry(conn, fmt.Sprintf("/ethdev/xstats,%d", id), &ethdevXstats)
		//log.Println("ethdevXstats", ethdevXstats)

		for statName, statValueFloat := range ethdevXstats.Value {
			InterfaceStat.With(prometheus.Labels{"interface": ethdevInfo.Value.Name, "stat_name": statName}).Set(statValueFloat)
		}
	}
	var dpserviceNatPort DpServiceNatPort
	QueryTelemetry(conn, "/dp_service/nat/used_port_count", &dpserviceNatPort)
	//log.Println("Dpservice nat port", dpserviceNatPort)
	for ifName, portCount := range dpserviceNatPort.Value {
		InterfaceStat.With(prometheus.Labels{"interface": ifName, "stat_name": "nat_used_port_count"}).Set(float64(portCount))
	}

	var dpserviceVirtsvcPort DpServiceVirtsvcPort
	QueryTelemetry(conn, "/dp_service/virtsvc/used_port_count", &dpserviceVirtsvcPort)
	//log.Println("Dpservice virtsvc port", dpserviceVirtsvcPort)
	for ifName, portCount := range dpserviceVirtsvcPort.Value {
		InterfaceStat.With(prometheus.Labels{"interface": ifName, "stat_name": "virtsvc_used_port_count"}).Set(float64(portCount))
	}

	var dpserviceCallCount DpServiceGraphCallCount
	QueryTelemetry(conn, "/dp_service/graph/call_count", &dpserviceCallCount)
	//log.Println("dpserviceCallCount", dpserviceCallCount)

	for graphNodeName, callCount := range dpserviceCallCount.GraphCallCnt.Node_0_to_255 {
		CallCount.With(prometheus.Labels{"node_name": hostname, "graph_node": graphNodeName}).Set(callCount)
	}
}
