// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"time"

	"github.com/ironcore-dev/prometheus-dpdk-exporter/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	maxRetries = 5
	sleepTime  = 10 * time.Second
)

func main() {
	var conn net.Conn
	var err error
	var host string
	var hostnameFlag string
	var pollIntervalFlag int
	var exporterPort uint64
	var exporterAddr netip.AddrPort

	r := prometheus.NewRegistry()
	r.MustRegister(metrics.InterfaceStat)
	r.MustRegister(metrics.CallCount)

	http.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))

	for i := 0; i < maxRetries; i++ {
		conn, err = net.Dial("unixpacket", metrics.SocketPath)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to %s: %v. Retry %d of %d", metrics.SocketPath, err, i+1, maxRetries)
		if i < maxRetries-1 {
			time.Sleep(sleepTime)
		}
	}
	defer conn.Close()

	flag.StringVar(&hostnameFlag, "hostname", "", "Hostname to use")
	flag.IntVar(&pollIntervalFlag, "poll-interval", 20, "Polling interval in seconds")
	flag.Uint64Var(&exporterPort, "port", 9064, "Port on which exporter will be running.")
	flag.Parse()

	if exporterPort < 1024 || exporterPort > 65535 {
		log.Fatal("port must be in range 1024 - 65535")
	}
	exporterAddr = netip.AddrPortFrom(netip.IPv4Unspecified(), uint16(exporterPort))

	host, err = getHostname(hostnameFlag)
	if err != nil {
		log.Fatal("could not get hostname")
	}
	fmt.Printf("Hostname: %s\n", host)

	flushSocket(conn)
	go func() {
		for {
			metrics.Update(conn, host)
			time.Sleep(time.Duration(pollIntervalFlag) * time.Second)
		}
	}()

	log.Printf("Server starting on :%v...\n", exporterPort)
	log.Fatal(http.ListenAndServe(exporterAddr.String(), nil))
}

func flushSocket(conn net.Conn) {
	respBytes := make([]byte, 1024)

	_, err := conn.Read(respBytes)
	if err != nil {
		log.Fatalf("Failed to read response from %s: %v", metrics.SocketPath, err)
	}
}

func getHostname(hostnameFlag string) (string, error) {
	if hostnameFlag == "" {
		// Try to get hostname from environment variable
		envHostName := os.Getenv("NODE_NAME")
		if envHostName != "" {
			return envHostName, nil
		} else {
			// If environment variable not set, get hostname from os.Hostname
			hostname, err := os.Hostname()
			if err != nil {
				return "unknown", err
			} else {
				return hostname, nil
			}
		}
	} else {
		return hostnameFlag, nil
	}
}
