// Copyright 2018 Ben Kochie <superq@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sparrc/go-ping"
	"os"
	"strconv"
	"strings"
)

const (
	namespace = "smokeping"
)

var (
	labelNames = []string{"host", "ip"}

	summary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "smokeping_response_latency_summary",
		Help:       "Summary for ping response latencies",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	},
		labelNames,
	)
	packetsTx = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "smokeping_packets_sent",
		Help: "counter of all packets being sent out",
	},
		labelNames,
	)
	packetsRx = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "smokeping_packets_received",
		Help: "counter of all responses received (ignoring dups)",
	},
		labelNames,
	)
	histo *prometheus.HistogramVec
)

func init() {
	prometheus.MustRegister(summary, packetsTx, packetsRx)
}

func newHisto(buckets string) {
	bucketstrings := strings.Split(buckets, ",")
	bucketlist := make([]float64, len(bucketstrings))
	for i := range bucketstrings {
		value, err := strconv.ParseFloat(bucketstrings[i], 64)
		if err != nil {
			fmt.Printf("invalid float in bucket: %s", err)
			os.Exit(10)
		}
		bucketlist[i] = value
	}

	histo = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "response_duration_seconds",
			Help:      "A histogram of latencies for ping responses.",
			Buckets:   bucketlist,
		},
		labelNames,
	)
	prometheus.MustRegister(histo)
}
func (pe *pingEntry) Ping() {
	packetsTx.WithLabelValues(pe.Hostname(), pe.Address()).Inc()
	pe.pinger.Run()
}
func (pe *pingEntry) Hostname() string {
	if pe.hostname == "" {
		pe.hostname = pe.pinger.Addr()
	}
	return pe.hostname

}
func (pe *pingEntry) Address() string {
	if pe.address == "" {
		pe.address = pe.pinger.IPAddr().String()
	}
	return pe.address
}

func (pe *pingEntry) OnRecv(pkt *ping.Packet) {
	packetsRx.WithLabelValues(pe.Hostname(), pe.Address()).Inc()
	summary.WithLabelValues(pe.Hostname(), pe.Address()).Observe(pkt.Rtt.Seconds())
	histo.WithLabelValues(pe.Hostname(), pe.Address()).Observe(pkt.Rtt.Seconds())
	if *debug {
		fmt.Printf("OnRecv %s: time=%v\n", pe.Hostname(), pkt.Rtt)
	}
	pe.received = true
}
func (pe *pingEntry) OnFinish(stats *ping.Statistics) {
	if *debug {
		fmt.Printf("OnFinish: %d packets transmitted, %d packets received, %v%% packet loss\n", stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
		fmt.Printf("OnFinish: round-trip min/avg/max/stddev = %v/%v/%v/%v\n", stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
	}
}
