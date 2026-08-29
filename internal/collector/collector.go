package collector

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// cmdTimeout bounds every external command the collector spawns, so a hung
// NFS mount (df), stale systemd (systemctl) or similar cannot block collection.
const cmdTimeout = 10 * time.Second

// runCmd executes a command with a timeout and returns its stdout.
func runCmd(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

// Metrics holds a snapshot of system metrics.
type Metrics struct {
	Timestamp    string  `json:"timestamp"`
	CPUPercent   float64 `json:"cpu_percent"`
	MemoryTotal  uint64  `json:"memory_total"`
	MemoryUsed   uint64  `json:"memory_used"`
	MemoryAvail  uint64  `json:"memory_available"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskTotal    uint64  `json:"disk_total"`
	DiskUsed     uint64  `json:"disk_used"`
	DiskFree     uint64  `json:"disk_free"`
	DiskPercent  float64 `json:"disk_percent"`
	Load1        float64 `json:"load_1"`
	Load5        float64 `json:"load_5"`
	Load15       float64 `json:"load_15"`
	CPUCount     int     `json:"cpu_count"`
	Uptime       int     `json:"uptime"`
	Hostname     string  `json:"hostname"`
	Network      string  `json:"network"` // reachable/unreachable/disabled
	ProcessCount int     `json:"process_count"`
}

// Instance represents one SCP:SL service instance.
type Instance struct {
	Service   string `json:"service"`
	Port      int    `json:"port"`
	State     string `json:"state"`     // active/inactive/failed/unknown
	UDP       string `json:"udp"`       // listening/not-listening/unknown
	Processes int    `json:"processes"` // process count
}

// Collector periodically gathers system metrics and instance info.
type Collector struct {
	mu       sync.RWMutex
	metrics  Metrics
	instances []Instance

	prevCPUIdle  uint64
	prevCPUTotal uint64
	firstCPU     bool

	connHost    string
	connPort    int
	connTimeout int
}

// NewCollector creates a new Collector.
func NewCollector(connHost string, connPort, connTimeout int) *Collector {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return &Collector{
		firstCPU:    true,
		connHost:    connHost,
		connPort:    connPort,
		connTimeout: connTimeout,
		metrics: Metrics{
			Hostname: hostname,
			Network:  "disabled",
		},
	}
}

// Get returns the current metrics snapshot and instance list.
func (c *Collector) Get() (Metrics, []Instance) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics, append([]Instance(nil), c.instances...)
}

// Collect performs one round of metric collection.
func (c *Collector) Collect() {
	m := Metrics{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Hostname:  c.metrics.Hostname,
		Network:   "disabled",
	}

	m.CPUCount = runtime.NumCPU()
	m.CPUPercent = c.readCPU()
	m.MemoryTotal, m.MemoryUsed, m.MemoryAvail, m.MemoryPercent = readMemory()
	m.DiskTotal, m.DiskUsed, m.DiskFree, m.DiskPercent = readDisk("/")
	m.Load1, m.Load5, m.Load15 = readLoad()
	m.Uptime = readUptime()
	m.ProcessCount = countProcesses()

	if c.connHost != "" {
		m.Network = checkConnectivity(c.connHost, c.connPort, c.connTimeout)
	}

	instances := discoverInstances()

	c.mu.Lock()
	c.metrics = m
	c.instances = instances
	c.mu.Unlock()
}

// --- Linux /proc helpers ---

func (c *Collector) readCPU() float64 {
	if runtime.GOOS != "linux" {
		return 0
	}
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 8 {
		return 0
	}

	var total, idle uint64
	for i := 1; i < len(fields); i++ {
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			continue
		}
		total += v
		if i == 4 { // idle
			idle = v
		}
	}

	if c.firstCPU {
		c.prevCPUIdle = idle
		c.prevCPUTotal = total
		c.firstCPU = false
		return 0
	}

	dt := total - c.prevCPUTotal
	di := idle - c.prevCPUIdle
	c.prevCPUIdle = idle
	c.prevCPUTotal = total

	if dt == 0 {
		return 0
	}
	usage := float64(dt-di) * 100 / float64(dt)
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	return round(usage, 1)
}

func readMemory() (total, used, avail uint64, percent float64) {
	if runtime.GOOS != "linux" {
		return 0, 0, 0, 0
	}
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, 0
	}
	defer f.Close()

	info := map[string]uint64{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		v, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		info[key] = v * 1024 // kB -> bytes
	}

	total = info["MemTotal"]
	avail = info["MemAvailable"]
	if avail == 0 {
		// fallback: free + buffers + cached
		avail = info["MemFree"] + info["Buffers"] + info["Cached"]
	}
	if total == 0 {
		return 0, 0, 0, 0
	}
	used = total - avail
	percent = round(float64(used)*100/float64(total), 1)
	return
}

func readDisk(path string) (total, used, free uint64, percent float64) {
	// Cross-platform disk usage via statfs
	if runtime.GOOS == "linux" {
		return readDiskLinux(path)
	}
	return 0, 0, 0, 0
}

func readDiskLinux(path string) (total, used, free uint64, percent float64) {
	out, err := runCmd("df", "-B1", path)
	if err != nil {
		return 0, 0, 0, 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0, 0, 0, 0
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return 0, 0, 0, 0
	}
	total, _ = strconv.ParseUint(fields[1], 10, 64)
	used, _ = strconv.ParseUint(fields[2], 10, 64)
	free, _ = strconv.ParseUint(fields[3], 10, 64)
	if total > 0 {
		percent = round(float64(used)*100/float64(total), 1)
	}
	return
}

func readLoad() (l1, l5, l15 float64) {
	if runtime.GOOS != "linux" {
		return 0, 0, 0
	}
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0
	}
	l1, _ = strconv.ParseFloat(fields[0], 64)
	l5, _ = strconv.ParseFloat(fields[1], 64)
	l15, _ = strconv.ParseFloat(fields[2], 64)
	return
}

func readUptime() int {
	if runtime.GOOS != "linux" {
		return 0
	}
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0
	}
	v, _ := strconv.ParseFloat(fields[0], 64)
	return int(v)
}

func countProcesses() int {
	if runtime.GOOS != "linux" {
		return 0
	}
	out, err := runCmd("ps", "-e", "--no-headers")
	if err != nil {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(string(out)), "\n"))
}

func checkConnectivity(host string, port, timeoutSec int) string {
	if timeoutSec <= 0 {
		timeoutSec = 5
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, time.Duration(timeoutSec)*time.Second)
	if err != nil {
		return "unreachable"
	}
	conn.Close()
	return "reachable"
}

// --- SCP:SL instance discovery ---

var svcRe = regexp.MustCompile(`(scpsl-\d+\.service)`)

// discoverInstances returns per-instance state. The ss snapshot is taken once
// per discovery round and reused for every instance.
func discoverInstances() []Instance {
	if runtime.GOOS != "linux" {
		return nil
	}

	out, err := runCmd("systemctl", "list-units", "--type=service", "--all",
		"--no-legend", "--no-pager")
	if err != nil {
		return nil
	}

	// One ss snapshot for all instances.
	udpOut, _ := runCmd("ss", "-H", "-lun")

	var instances []Instance
	seen := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		m := svcRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		svc := m[1]
		if seen[svc] {
			continue
		}
		seen[svc] = true

		port := 0
		pm := regexp.MustCompile(`(\d+)`).FindStringSubmatch(svc)
		if pm != nil {
			port, _ = strconv.Atoi(pm[1])
		}

		state := checkService(svc)
		udp := "not-listening"
		if UDPPortListening(string(udpOut), port) {
			udp = "listening"
		}
		procs := countPortProcesses(port)

		instances = append(instances, Instance{
			Service:   svc,
			Port:      port,
			State:     state,
			UDP:       udp,
			Processes: procs,
		})
	}
	return instances
}

func checkService(svc string) string {
	out, err := runCmd("systemctl", "is-active", svc)
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// UDPPortListening reports whether the given ss -lun output shows any local
// address bound to port. Parsing the local-address column avoids substring
// false positives (e.g. port 777 matching 7777).
func UDPPortListening(ssOutput string, port int) bool {
	if port <= 0 {
		return false
	}
	portStr := strconv.Itoa(port)
	for _, line := range strings.Split(ssOutput, "\n") {
		// ss -H -lun columns: Netid State Recv-Q Send-Q Local Peer
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		local := fields[4]
		if localAddrPort(local) == portStr {
			return true
		}
	}
	return false
}

// localAddrPort extracts the port from a socket local address of the form
// "0.0.0.0:7777", "[::]:7777" or "127.0.0.1:7777".
func localAddrPort(addr string) string {
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[i+1:]
	}
	return ""
}

func countPortProcesses(port int) int {
	if port == 0 {
		return 0
	}
	// Match either "-port 7777" or "port=7777" in the process command line.
	out, err := runCmd("pgrep", "-c", "-f", fmt.Sprintf("port[ =]%d", port))
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n
}

func round(v float64, decimals int) float64 {
	mult := 1.0
	for i := 0; i < decimals; i++ {
		mult *= 10
	}
	return float64(int(v*mult+0.5)) / mult
}
