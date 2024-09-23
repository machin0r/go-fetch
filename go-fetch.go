// Package main implements a system information fetching tool.
// It provides detailed hardware and software information about the host system.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/jaypipes/ghw"
	"github.com/muesli/termenv"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// ANSI color codes for output formatting
const (
	text     = "\033[34m"
	category = "\033[95m"
	hostcol  = "\033[34m"
	reset    = "\033[0m"
)

// colourise wraps the given text with the specified colour code and reset code.
// This function is used to apply consistent colouring throughout the output.
func colourise(text string, color string) string {
	return color + text + reset
}

// main is the entry point of the application. It controls the gathering
// and display of system information, handling any errors that occur during
// the process.
func main() {
	// Redirect stderr to a log file
	logFile, err := os.OpenFile("go-fetch.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err == nil {
		defer logFile.Close()
		os.Stderr = logFile
	}

	// Disable ghw warnings
	ghw.WithDisableWarnings()

	// Gather system information
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	osInfo, _ := host.Info()
	memInfo, _ := mem.VirtualMemory()
	cpuModel, logicalCores, cpuSpeed := getCPUInfo()

	// Attempt to get package count, defaulting to -1 if unsuccessful
	packageCount, err := getPackageCount()
	if err != nil {
		packageCount = -1 // Indicate error
	}

	// Attempt to get GPU information, defaulting to "None" if unsuccessful
	gpu, err := getGPUInfo()
	if err != nil {
		gpu = "None"
	}

	// Print the username and hostname
	fmt.Printf("\x1b[1m%s@%s\x1b[0m\n", colourise(username, hostcol), colourise(hostname, hostcol))

	// Prepare the information to be displayed
	info := []struct {
		label string
		value string
	}{
		{"Hostname", hostname},
		{"OS", fmt.Sprintf("%s %s", osInfo.Platform, osInfo.PlatformVersion)},
		{"Kernel", osInfo.KernelVersion},
		{"Uptime", formatUptime(osInfo.Uptime)},
		{"Shell", filepath.Base(os.Getenv("SHELL"))},
		{"CPU", fmt.Sprintf("%s (%d cores @ %.2f GHz)", cpuModel, logicalCores, cpuSpeed)},
		{"GPU", fmt.Sprintf("%s", gpu)},
		{"Memory", fmt.Sprintf("%s / %s", formatBytes(memInfo.Used), formatBytes(memInfo.Total))},
	}

	// Add package count information if available
	if packageCount >= 0 {
		info = append(info, struct{ label, value string }{"Packages", fmt.Sprintf("%d", packageCount)})
	} else {
		info = append(info, struct{ label, value string }{"Packages", fmt.Sprintf("Unable to determine (%s)", err)})
	}

	// Find the longest label for alignment
	maxLabelLength := 0
	for _, item := range info {
		if len(item.label) > maxLabelLength {
			maxLabelLength = len(item.label)
		}
	}

	// Print aligned and colored information
	for _, item := range info {
		fmt.Printf("%s%-*s %s\n",
			colourise(item.label, category),
			maxLabelLength-len(item.label),
			"",
			colourise(item.value, text))
	}
	// Display color blocks at the end of the output
	printColorBlocks()
}

// formatUptime converts the uptime in seconds to a human-readable string
// in the format of "Xd Yh Zm" (days, hours, minutes).
func formatUptime(uptime uint64) string {
	duration := time.Duration(uptime) * time.Second
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
}

// formatBytes converts bytes to a human-readable string with appropriate unit suffix.
// It uses binary units (KiB, MiB, GiB, etc.)
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

// getPackageCount attempts to count the number of installed packages on the system.
// It supports multiple package managers (dpkg, pacman, rpm) and tries to detect
// the appropriate one based on the OS and available commands.
func getPackageCount() (int, error) {
	var cmd *exec.Cmd
	osInfo, err := host.Info()
	if err != nil {
		return 0, err
	}

	// Determine the appropriate package manager command based on the OS
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
			// Fallback detection for unknown distributions
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

	// Execute the command and count the lines of output
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	return len(strings.Split(string(output), "\n")) - 1, nil
}

// getCPUInfo retrieves CPU information including model name, core count, and clock speed.
// It falls back to runtime.NumCPU() for core count if gopsutil fails to provide the information.
func getCPUInfo() (model string, cores int, speed float64) {
	cpuInfo, err := cpu.Info()
	if err != nil || len(cpuInfo) == 0 {
		// Fallback to runtime package for core count
		cores = runtime.NumCPU()
		model = "Unknown"
		speed = 0.0
		return
	}

	model = cpuInfo[0].ModelName
	cores = runtime.NumCPU()      // Use runtime.NumCPU() for consistent logical core count
	speed = cpuInfo[0].Mhz / 1000 // Convert to GHz

	// If cores is 0, fallback to runtime package
	if cores == 0 {
		cores = runtime.NumCPU()
	}

	return
}

// cleanGPUName removes model number prefixes from GPU names to provide a cleaner output.
// It uses a regular expression to extract the main part of the GPU name.
func cleanGPUName(name string) string {
	re := regexp.MustCompile(`^[A-Z0-9]+\s*\[(.+)\]$`)
	matches := re.FindStringSubmatch(name)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return name
}

// getGPUInfo attempts to retrieve information about the system's GPU.
// It returns the name of the first GPU found or an error if no GPU is detected.
func getGPUInfo() (string, error) {
	gpu, err := ghw.GPU()
	if err != nil {
		return "", err
	}
	if len(gpu.GraphicsCards) > 0 {
		card := gpu.GraphicsCards[0]
		if card.DeviceInfo != nil && card.DeviceInfo.Product != nil {
			return cleanGPUName(card.DeviceInfo.Product.Name), nil
		}
		return fmt.Sprintf("Unknown GPU (Vendor: %s)", card.DeviceInfo.Vendor.Name), nil
	}
	return "None", nil
}

// printColorBlocks displays a row of colored blocks at the end of the output.
func printColorBlocks() {
	p := termenv.ColorProfile()

	// Use the 16 ANSI colors
	colors := []termenv.Color{
		p.Color("0"), // Black
		p.Color("1"), // Red
		p.Color("2"), // Green
		p.Color("3"), // Yellow
		p.Color("4"), // Blue
		p.Color("5"), // Magenta
		p.Color("6"), // Cyan
	}

	for _, color := range colors {
		fmt.Print(termenv.String("   ").Background(color))
	}
	fmt.Println()
}
