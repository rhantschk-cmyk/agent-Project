package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemStats struct {
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	RAMUsagePercent float64 `json:"ram_usage_percent"`
	RAMTotalMB      uint64  `json:"ram_total_mb"`
	RAMUsedMB       uint64  `json:"ram_used_mb"`
	DiskFreeGB      uint64  `json:"disk_free_gb"`
	DiskTotalGB     uint64  `json:"disk_total_gb"`
	UptimeSeconds   uint64  `json:"uptime_seconds"`
	UptimeFormated  string  `json:"uptime_formated"`
	GPUInfo         string  `json:"gpu_info"`
}

func getSystemStats() (SystemStats, error) {
	var stats SystemStats
	logf("-> [Monitoring] Fetching System Stats")

	cpuPercents, err := cpu.Percent(200*time.Millisecond, false)
	if err == nil && len(cpuPercents) > 0 {
		stats.CPUUsagePercent = cpuPercents[0]
	}
	logf("-> [Monitoring] Got CPU Percents")

	vMem, err := mem.VirtualMemory()
	if err == nil {
		stats.RAMUsagePercent = vMem.UsedPercent
		stats.RAMTotalMB = vMem.Total / 1024 / 1024
		stats.RAMUsedMB = vMem.Used / 1024 / 1024
	}
	logf("-> [Monitoring] Got RAM Info")

	diskUsage, err := disk.Usage("/")
	if err == nil {
		stats.DiskFreeGB = diskUsage.Free / 1024 / 1024 / 1024
		stats.DiskTotalGB = diskUsage.Total / 1024 / 1024 / 1024
	}
	logf("-> [Monitoring] Got Disk Info")

	hostInfo, err := host.Info()
	if err == nil {
		stats.UptimeSeconds = hostInfo.Uptime
		d := time.Duration(hostInfo.Uptime) * time.Second
		stats.UptimeFormated = fmt.Sprintf("%dd %dh %dm", int(d.Hours())/24, int(d.Hours())%24, int(d.Minutes())%60)
	}
	logf("-> [Monitoring] Got Host Info")

	stats.GPUInfo = "Derzeit noch nicht verfügbar"
	logf("-> [Monitoring] Done")

	return stats, nil
}

func startMonitorServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		stats, err := getSystemStats()
		if err != nil {
			http.Error(w, "Fehler beim Auslesen der Stats", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	server := &http.Server{
		Addr:    ":9000",
		Handler: mux,
	}

	logf("-> [Monitoring] Starte Server auf :9000")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logf("-> [Monitoring] Server error: %v", err)
	}
}
