package partition

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// DiskInfo contains detailed information about a disk
type DiskInfo struct {
	Device       string
	Model        string
	Serial       string
	Size         uint64
	SectorSize   uint64
	Scheme       string
	Temperature  int
	PowerOnHours uint64
	PowerCycles  uint64
	SMARTStatus  string
	SMARTEnabled bool
	Attributes   []SMARTAttribute
	Capabilities []string
}

// SMARTAttribute represents a SMART attribute
type SMARTAttribute struct {
	ID          int
	Name        string
	Value       int
	Worst       int
	Threshold   int
	RawValue    string
	Status      string
	Description string
}

// GetDetailedDiskInfo retrieves comprehensive disk information including SMART data
func GetDetailedDiskInfo(diskName string) (*DiskInfo, error) {
	info := &DiskInfo{
		Device: diskName,
	}

	// Get basic disk info from geom
	if err := getGeomInfo(info); err != nil {
		return nil, fmt.Errorf("failed to get geom info: %w", err)
	}

	// Get SMART data if available
	if err := getSMARTInfo(info); err != nil {
		// SMART may not be available, but don't fail entirely
		info.SMARTEnabled = false
	}

	// Get additional capabilities
	getCapabilities(info)

	return info, nil
}

// getGeomInfo gets basic disk information from geom
func getGeomInfo(info *DiskInfo) error {
	cmd := exec.Command("geom", "disk", "list", info.Device)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Mediasize:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				size, _ := strconv.ParseUint(fields[1], 10, 64)
				info.Size = size
			}
		} else if strings.HasPrefix(line, "Sectorsize:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				size, _ := strconv.ParseUint(fields[1], 10, 64)
				info.SectorSize = size
			}
		} else if strings.HasPrefix(line, "descr:") {
			info.Model = strings.TrimSpace(strings.TrimPrefix(line, "descr:"))
		} else if strings.HasPrefix(line, "ident:") {
			info.Serial = strings.TrimSpace(strings.TrimPrefix(line, "ident:"))
		}
	}

	// Get partition scheme
	cmd = exec.Command("gpart", "show", info.Device)
	output, _ = cmd.CombinedOutput()
	lines = strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "=>") {
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				info.Scheme = strings.ToUpper(fields[5])
			}
		}
	}

	return nil
}

// getSMARTInfo retrieves SMART data from the disk
func getSMARTInfo(info *DiskInfo) error {
	// Check if smartctl is available
	if _, err := exec.LookPath("smartctl"); err != nil {
		return fmt.Errorf("smartctl not found - install smartmontools: pkg install smartmontools")
	}

	// Get SMART overall health
	cmd := exec.Command("smartctl", "-H", "/dev/"+info.Device)
	output, err := cmd.CombinedOutput()
	outStr := string(output)

	if err == nil {
		info.SMARTEnabled = true
		if strings.Contains(outStr, "PASSED") {
			info.SMARTStatus = "PASSED"
		} else if strings.Contains(outStr, "FAILED") {
			info.SMARTStatus = "FAILED"
		} else {
			info.SMARTStatus = "UNKNOWN"
		}
	}

	// Get detailed SMART attributes
	cmd = exec.Command("smartctl", "-A", "/dev/"+info.Device)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil // Don't fail if attributes aren't available
	}

	parseSMARTAttributes(info, string(output))

	// Get SMART information (temperature, power on hours, etc.)
	cmd = exec.Command("smartctl", "-a", "/dev/"+info.Device)
	output, _ = cmd.CombinedOutput()
	parseSMARTDetails(info, string(output))

	return nil
}

// parseSMARTAttributes parses SMART attribute table
func parseSMARTAttributes(info *DiskInfo, output string) {
	lines := strings.Split(output, "\n")
	inTable := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "ID#") && strings.Contains(line, "ATTRIBUTE_NAME") {
			inTable = true
			continue
		}

		if !inTable || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		id, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		value, _ := strconv.Atoi(fields[3])
		worst, _ := strconv.Atoi(fields[4])
		threshold, _ := strconv.Atoi(fields[5])

		attr := SMARTAttribute{
			ID:        id,
			Name:      fields[1],
			Value:     value,
			Worst:     worst,
			Threshold: threshold,
			RawValue:  fields[9],
		}

		// Determine status
		if value <= threshold {
			attr.Status = "FAILING"
		} else if value < threshold+10 {
			attr.Status = "WARNING"
		} else {
			attr.Status = "OK"
		}

		// Add human-readable description
		attr.Description = getSMARTAttributeDescription(attr.Name, attr.ID)

		info.Attributes = append(info.Attributes, attr)
	}
}

// parseSMARTDetails extracts temperature, power on hours, etc.
func parseSMARTDetails(info *DiskInfo, output string) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Temperature_Celsius") || strings.Contains(line, "Airflow_Temperature") {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				if temp, err := strconv.Atoi(fields[9]); err == nil {
					info.Temperature = temp
				}
			}
		} else if strings.Contains(line, "Power_On_Hours") {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				if hours, err := strconv.ParseUint(fields[9], 10, 64); err == nil {
					info.PowerOnHours = hours
				}
			}
		} else if strings.Contains(line, "Power_Cycle_Count") || strings.Contains(line, "Start_Stop_Count") {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				if cycles, err := strconv.ParseUint(fields[9], 10, 64); err == nil {
					info.PowerCycles = cycles
				}
			}
		}
	}
}

// getCapabilities determines disk capabilities
func getCapabilities(info *DiskInfo) {
	info.Capabilities = []string{}

	// Check for TRIM support
	cmd := exec.Command("camcontrol", "identify", info.Device)
	output, err := cmd.CombinedOutput()
	if err == nil {
		outStr := strings.ToLower(string(output))
		if strings.Contains(outStr, "trim") || strings.Contains(outStr, "data set management") {
			info.Capabilities = append(info.Capabilities, "TRIM/UNMAP support")
		}
		if strings.Contains(outStr, "naa") || strings.Contains(outStr, "sata") {
			info.Capabilities = append(info.Capabilities, "SATA")
		}
		if strings.Contains(outStr, "nvme") {
			info.Capabilities = append(info.Capabilities, "NVMe")
		}
	}

	// Check if it's an SSD
	if info.Model != "" {
		modelLower := strings.ToLower(info.Model)
		if strings.Contains(modelLower, "ssd") || strings.Contains(modelLower, "solid state") {
			info.Capabilities = append(info.Capabilities, "Solid State Drive (SSD)")
		}
	}

	// Add rotation rate if available (for HDDs)
	if len(info.Capabilities) == 0 || !strings.Contains(strings.Join(info.Capabilities, " "), "SSD") {
		info.Capabilities = append(info.Capabilities, "Hard Disk Drive (HDD)")
	}
}

// getSMARTAttributeDescription returns a human-readable description
func getSMARTAttributeDescription(name string, id int) string {
	descriptions := map[string]string{
		"Raw_Read_Error_Rate":     "Rate of hardware read errors",
		"Throughput_Performance":  "Overall throughput performance",
		"Spin_Up_Time":            "Time to spin up to operating speed",
		"Start_Stop_Count":        "Number of spindle start/stop cycles",
		"Reallocated_Sector_Ct":   "Count of reallocated sectors",
		"Seek_Error_Rate":         "Rate of seek errors",
		"Seek_Time_Performance":   "Seek time performance",
		"Power_On_Hours":          "Total hours powered on",
		"Spin_Retry_Count":        "Number of retry attempts to spin up",
		"Power_Cycle_Count":       "Number of power-on events",
		"End-to-End_Error":        "Errors in data transfer",
		"Reported_Uncorrect":      "Uncorrectable sector count",
		"Command_Timeout":         "Count of command timeouts",
		"Temperature_Celsius":     "Current drive temperature",
		"Hardware_ECC_Recovered":  "ECC errors corrected by hardware",
		"Current_Pending_Sector":  "Sectors waiting to be remapped",
		"Offline_Uncorrectable":   "Uncorrectable offline errors",
		"UDMA_CRC_Error_Count":    "CRC errors during UDMA transfers",
		"Multi_Zone_Error_Rate":   "Write error rate across zones",
		"Wear_Leveling_Count":     "SSD wear leveling count",
		"Total_LBAs_Written":      "Total logical blocks written",
		"Total_LBAs_Read":         "Total logical blocks read",
		"Available_Reservd_Space": "Available reserved space (SSD)",
		"Runtime_Bad_Block":       "Runtime bad block count",
		"Airflow_Temperature_Cel": "Airflow temperature",
	}

	if desc, ok := descriptions[name]; ok {
		return desc
	}

	return "SMART attribute ID " + strconv.Itoa(id)
}
