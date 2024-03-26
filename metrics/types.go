package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	InterfaceStat = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dpdk_interface_stat",
			Help: "DPDK interface statistic",
		},
		[]string{"interface", "stat_name"},
	)

	CallCount = prometheus.NewGaugeVec(
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

type NodeData map[string]float64

type GraphCallCount struct {
	Node_0_to_255 NodeData `json:"Node_0_to_255"`
}

type DpServiceGraphCallCount struct {
	GraphCallCnt GraphCallCount `json:"/dp_service/graph/call_count"`
}
