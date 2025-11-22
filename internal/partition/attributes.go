package partition

import (
	"fmt"
	"os/exec"
	"strings"
)

// GPTAttribute represents a GPT partition attribute
type GPTAttribute struct {
	Name        string
	Value       bool
	Description string
}

// Common GPT attributes in FreeBSD
const (
	AttrBootme     = "bootme"     // Platform required (system partition)
	AttrBootonce   = "bootonce"   // Boot from this partition once
	AttrBootfailed = "bootfailed" // Partition failed to boot
	AttrNoBlockIO  = "noblockio"  // No block I/O protocol
)

// AttributeInfo contains information about partition attributes
type AttributeInfo struct {
	Partition  string
	Attributes map[string]bool
	RawValue   string
}

// GetAvailableAttributes returns a list of supported GPT attributes
func GetAvailableAttributes() []GPTAttribute {
	return []GPTAttribute{
		{
			Name:        AttrBootme,
			Value:       false,
			Description: "Platform required - marks partition as bootable/system partition",
		},
		{
			Name:        AttrBootonce,
			Value:       false,
			Description: "Boot once - boot from this partition once then clear flag",
		},
		{
			Name:        AttrBootfailed,
			Value:       false,
			Description: "Boot failed - indicates partition failed to boot",
		},
		{
			Name:        AttrNoBlockIO,
			Value:       false,
			Description: "No block I/O - disable block I/O protocol",
		},
	}
}

// GetPartitionAttributes retrieves the current attributes of a partition
func GetPartitionAttributes(partName string) (*AttributeInfo, error) {
	// Extract disk name from partition name
	diskName, _, err := ParsePartitionName(partName)
	if err != nil {
		// Try to extract disk name another way
		diskName = strings.TrimRight(partName, "0123456789ps")
	}

	// Get partition information using gpart show
	cmd := exec.Command("gpart", "show", "-l", "-p", diskName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get partition info: %v", err)
	}

	info := &AttributeInfo{
		Partition:  partName,
		Attributes: make(map[string]bool),
		RawValue:   "",
	}

	// Parse the output to find attributes
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, partName) {
			fields := strings.Fields(line)
			// The attributes field varies by output format
			// Look for attribute indicators
			lineUpper := strings.ToUpper(line)

			// Check for common attribute markers
			if strings.Contains(lineUpper, "BOOTME") {
				info.Attributes[AttrBootme] = true
			}
			if strings.Contains(lineUpper, "BOOTONCE") {
				info.Attributes[AttrBootonce] = true
			}
			if strings.Contains(lineUpper, "BOOTFAILED") {
				info.Attributes[AttrBootfailed] = true
			}
			if strings.Contains(lineUpper, "NOBLOCKIO") {
				info.Attributes[AttrNoBlockIO] = true
			}

			// Store the raw line for reference
			if len(fields) > 0 {
				info.RawValue = line
			}
			break
		}
	}

	return info, nil
}

// SetPartitionAttribute sets a GPT attribute on a partition
func SetPartitionAttribute(partName, attribute string) error {
	// Validate attribute name
	valid := false
	for _, attr := range GetAvailableAttributes() {
		if attr.Name == attribute {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid attribute: %s", attribute)
	}

	// Set the attribute using gpart
	cmd := exec.Command("gpart", "set", "-a", attribute, partName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set attribute %s: %v\nOutput: %s", attribute, err, string(output))
	}

	return nil
}

// UnsetPartitionAttribute unsets a GPT attribute on a partition
func UnsetPartitionAttribute(partName, attribute string) error {
	// Validate attribute name
	valid := false
	for _, attr := range GetAvailableAttributes() {
		if attr.Name == attribute {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid attribute: %s", attribute)
	}

	// Unset the attribute using gpart
	cmd := exec.Command("gpart", "unset", "-a", attribute, partName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unset attribute %s: %v\nOutput: %s", attribute, err, string(output))
	}

	return nil
}

// TogglePartitionAttribute toggles a GPT attribute on a partition
func TogglePartitionAttribute(partName, attribute string) error {
	// Get current attributes
	info, err := GetPartitionAttributes(partName)
	if err != nil {
		return err
	}

	// Toggle the attribute
	if info.Attributes[attribute] {
		return UnsetPartitionAttribute(partName, attribute)
	}
	return SetPartitionAttribute(partName, attribute)
}

// SetBootable marks a partition as bootable (convenience function)
func SetBootable(partName string) error {
	return SetPartitionAttribute(partName, AttrBootme)
}

// UnsetBootable removes the bootable flag from a partition
func UnsetBootable(partName string) error {
	return UnsetPartitionAttribute(partName, AttrBootme)
}

// IsBootable checks if a partition is marked as bootable
func IsBootable(partName string) (bool, error) {
	info, err := GetPartitionAttributes(partName)
	if err != nil {
		return false, err
	}
	return info.Attributes[AttrBootme], nil
}

// FormatAttributeInfo returns a human-readable attribute report
func FormatAttributeInfo(info *AttributeInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Partition: %s\n", info.Partition))
	sb.WriteString("Attributes:\n")

	hasAttributes := false
	availableAttrs := GetAvailableAttributes()

	for _, attr := range availableAttrs {
		status := "[ ]"
		if info.Attributes[attr.Name] {
			status = "[✓]"
			hasAttributes = true
		}
		sb.WriteString(fmt.Sprintf("  %s %s - %s\n", status, attr.Name, attr.Description))
	}

	if !hasAttributes {
		sb.WriteString("\nNo attributes are currently set.\n")
	}

	return sb.String()
}

// ValidatePartitionForAttributes checks if a partition supports GPT attributes
func ValidatePartitionForAttributes(partName string) error {
	// Extract disk name
	diskName, _, err := ParsePartitionName(partName)
	if err != nil {
		return fmt.Errorf("invalid partition name: %v", err)
	}

	// Check if disk uses GPT
	cmd := exec.Command("gpart", "show", diskName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check partition scheme: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "GPT") {
		return fmt.Errorf("partition %s is not on a GPT disk (attributes only available for GPT)", partName)
	}

	return nil
}
