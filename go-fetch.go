package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

func main() {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	osInfo, _ := host.Info()
	memInfo, _ := mem.VirtualMemory()
	cpuInfo, _ := cpu.Info()

	var cpuModel string
	var cpuSpeed float64

	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
		cpuSpeed = cpuInfo[0].Mhz / 1000 // GHz
	} else {
		cpuModel = "Unknown"
		cpuSpeed = 0
	}

	logicalCores := runtime.NumCPU()

	packageCount, err := getPackageCount()
	if err != nil {
		packageCount = -1 // Indicate error
	}

	fmt.Printf("\x1b[1m%s@%s\x1b[0m\n", username, hostname)
	fmt.Println(strings.Repeat("-", len(username)+len(hostname)+1))

	info := []string{
		fmt.Sprintf("Hostname: %s", hostname),
		fmt.Sprintf("OS: %s %s", osInfo.Platform, osInfo.PlatformVersion),
		fmt.Sprintf("Kernel: %s", osInfo.KernelVersion),
		fmt.Sprintf("Uptime: %s", formatUptime(osInfo.Uptime)),
		fmt.Sprintf("Shell: %s", os.Getenv("SHELL")),
		fmt.Sprintf("CPU: %s (%d cores @ %.2f GHz)", cpuModel, logicalCores, cpuSpeed),
		fmt.Sprintf("Memory: %s / %s", formatBytes(memInfo.Used), formatBytes(memInfo.Total)),
	}

	if packageCount >= 0 {
		info = append(info, fmt.Sprintf("Installed Packages: %d", packageCount))
	} else {
		info = append(info, fmt.Sprintf("Installed Packages: Unable to determine (%s)", err))
	}

	maxLength := 0
	for _, line := range info {
		if len(line) > maxLength {
			maxLength = len(line)
		}
	}

	fmt.Println(strings.Repeat("-", maxLength+4))
	for _, line := range info {
		fmt.Printf("| %-*s |\n", maxLength, line)
	}
	fmt.Println(strings.Repeat("-", maxLength+4))
}

func formatUptime(uptime uint64) string {
	duration := time.Duration(uptime) * time.Second
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func getPackageCount() (int, error) {
	var cmd *exec.Cmd
	osInfo, err := host.Info()
	if err != nil {
		return 0, err
	}

	switch runtime.GOOS {
	case "linux":
		switch osInfo.Platform {
		case "debian", "ubuntu":
			cmd = exec.Command("dpkg", "--get-selections")
		case "arch":
			cmd = exec.Command("pacman", "-Q")
		case "fedora", "centos", "rhel":
			cmd = exec.Command("rpm", "-qa")
		default:
			if _, err := exec.LookPath("dpkg"); err == nil {
				cmd = exec.Command("dpkg", "--get-selections")
			} else if _, err := exec.LookPath("pacman"); err == nil {
				cmd = exec.Command("pacman", "-Q")
			} else if _, err := exec.LookPath("rpm"); err == nil {
				cmd = exec.Command("rpm", "-qa")
			} else {
				return 0, fmt.Errorf("unsupported Linux distribution")
			}
		}
	default:
		return 0, fmt.Errorf("Unsupported OS for package counting")
	}

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	return len(strings.Split(string(output), "\n")) - 1, nil
}
