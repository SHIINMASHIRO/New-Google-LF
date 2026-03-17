package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// nicSampler measures real network interface RX throughput.
type nicSampler struct {
	mu       sync.Mutex
	iface    string
	lastRx   int64
	lastTime time.Time
	lastRate float64
}

func newNICSampler() *nicSampler {
	iface := detectPrimaryIface()
	rx, _ := readIfaceRxBytes(iface)
	return &nicSampler{
		iface:    iface,
		lastRx:   rx,
		lastTime: time.Now(),
	}
}

// Rate returns the RX rate in Mbps since the last call.
func (s *nicSampler) Rate() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	rx, err := readIfaceRxBytes(s.iface)
	if err != nil {
		return s.lastRate
	}

	elapsed := now.Sub(s.lastTime).Seconds()
	if elapsed < 0.5 {
		return s.lastRate
	}

	delta := rx - s.lastRx
	if delta < 0 {
		delta = 0
	}

	rate := float64(delta) / elapsed / 1e6 * 8 // Mbps
	s.lastRx = rx
	s.lastTime = now
	s.lastRate = rate
	return rate
}

// readIfaceRxBytes reads cumulative RX bytes for iface from /proc/net/dev.
func readIfaceRxBytes(iface string) (int64, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		prefix := iface + ":"
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		fields := strings.Fields(line[len(prefix):])
		if len(fields) < 1 {
			continue
		}
		return strconv.ParseInt(fields[0], 10, 64)
	}
	return 0, fmt.Errorf("interface %s not found in /proc/net/dev", iface)
}

// detectPrimaryIface returns the non-loopback interface with the most RX traffic.
func detectPrimaryIface() string {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return "eth0"
	}
	var best string
	var bestBytes int64
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:colonIdx])
		if name == "lo" {
			continue
		}
		fields := strings.Fields(line[colonIdx+1:])
		if len(fields) < 1 {
			continue
		}
		rx, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		if rx > bestBytes {
			bestBytes = rx
			best = name
		}
	}
	if best == "" {
		return "eth0"
	}
	return best
}
