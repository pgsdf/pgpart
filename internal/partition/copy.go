package partition

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// CopyPartition copies data from source partition to destination partition
func CopyPartition(sourcePart, destPart string, progressCallback func(float64)) error {
	if err := CheckPrivileges(); err != nil {
		return err
	}

	// Validate source and destination
	if sourcePart == destPart {
		return fmt.Errorf("source and destination cannot be the same")
	}

	// Get source partition size
	sourceSize, err := getPartitionSize(sourcePart)
	if err != nil {
		return fmt.Errorf("failed to get source partition size: %w", err)
	}

	// Get destination partition size
	destSize, err := getPartitionSize(destPart)
	if err != nil {
		return fmt.Errorf("failed to get destination partition size: %w", err)
	}

	// Check if destination is large enough
	if destSize < sourceSize {
		return fmt.Errorf("destination partition (%s) is too small - source: %d bytes, dest: %d bytes",
			FormatBytes(destSize), sourceSize, destSize)
	}

	// Use dd with status=progress if available, otherwise use basic dd
	blockSize := uint64(1024 * 1024) // 1MB blocks
	cmd := exec.Command("dd",
		"if=/dev/"+sourcePart,
		"of=/dev/"+destPart,
		fmt.Sprintf("bs=%d", blockSize),
		"conv=sync,noerror",
		"status=progress",
	)

	// Set up pipes to capture output
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start dd command: %w", err)
	}

	// Monitor progress
	if progressCallback != nil {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			// Parse dd progress output
			if strings.Contains(line, "bytes") {
				progress := parseProgress(line, sourceSize)
				progressCallback(progress)
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("partition copy failed: %w", err)
	}

	return nil
}

// MovePartition moves a partition by copying it and then deleting the source
func MovePartition(sourceDisk, sourceIndex, destDisk, destIndex string, progressCallback func(float64)) error {
	if err := CheckPrivileges(); err != nil {
		return err
	}

	// First, copy the partition
	sourcePart := fmt.Sprintf("%sp%s", sourceDisk, sourceIndex)
	destPart := fmt.Sprintf("%sp%s", destDisk, destIndex)

	if err := CopyPartition(sourcePart, destPart, progressCallback); err != nil {
		return fmt.Errorf("failed to copy partition: %w", err)
	}

	// After successful copy, delete the source partition
	if err := DeletePartition(sourceDisk, sourceIndex); err != nil {
		return fmt.Errorf("copy succeeded but failed to delete source partition: %w", err)
	}

	return nil
}

// ClonePartition creates a new partition with the same data
func ClonePartition(sourcePart, destPart string, progressCallback func(float64)) error {
	return CopyPartition(sourcePart, destPart, progressCallback)
}

// getPartitionSize returns the size of a partition in bytes
func getPartitionSize(partName string) (uint64, error) {
	cmd := exec.Command("diskinfo", "/dev/"+partName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get partition info: %w", err)
	}

	// diskinfo output: /dev/ada0p2	512	512000000	1000000	0	0
	// Fields: device, sectorsize, mediasize, sectors, ...
	fields := strings.Fields(string(output))
	if len(fields) < 3 {
		return 0, fmt.Errorf("unexpected diskinfo output format")
	}

	size, err := strconv.ParseUint(fields[2], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse partition size: %w", err)
	}

	return size, nil
}

// parseProgress extracts progress percentage from dd output
func parseProgress(line string, totalSize uint64) float64 {
	// Example dd output: "524288000 bytes (524 MB) copied"
	// Extract the number of bytes copied
	fields := strings.Fields(line)
	if len(fields) > 0 {
		if bytes, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
			if totalSize > 0 {
				return float64(bytes) / float64(totalSize) * 100.0
			}
		}
	}
	return 0.0
}

// VerifyPartitionCopy verifies that the copy was successful by comparing checksums
func VerifyPartitionCopy(sourcePart, destPart string) error {
	if err := CheckPrivileges(); err != nil {
		return err
	}

	// Get source checksum
	sourceChecksum, err := getPartitionChecksum(sourcePart)
	if err != nil {
		return fmt.Errorf("failed to get source checksum: %w", err)
	}

	// Get destination checksum
	destChecksum, err := getPartitionChecksum(destPart)
	if err != nil {
		return fmt.Errorf("failed to get destination checksum: %w", err)
	}

	if sourceChecksum != destChecksum {
		return fmt.Errorf("verification failed: checksums do not match")
	}

	return nil
}

// getPartitionChecksum calculates SHA256 checksum of partition data
func getPartitionChecksum(partName string) (string, error) {
	cmd := exec.Command("sha256", "-q", "/dev/"+partName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
