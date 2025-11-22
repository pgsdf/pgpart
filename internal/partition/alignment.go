package partition

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// AlignmentInfo contains partition alignment information
type AlignmentInfo struct {
	Partition      string
	StartOffset    uint64
	SectorSize     uint64
	PhysicalSize   uint64
	IsAligned      bool
	AlignmentType  string
	Recommendation string
}

// Common alignment boundaries in bytes
const (
	Align4K   uint64 = 4096    // 4 KiB - minimum for advanced format
	Align128K uint64 = 131072  // 128 KiB - good for some SSDs
	Align1M   uint64 = 1048576 // 1 MiB - recommended default
	Align4M   uint64 = 4194304 // 4 MiB - optimal for many SSDs
)

// CheckPartitionAlignment checks if a partition is properly aligned
func CheckPartitionAlignment(partName string) (*AlignmentInfo, error) {
	info := &AlignmentInfo{
		Partition: partName,
	}

	// Get partition start offset using gpart show
	cmd := exec.Command("gpart", "show", "-p", partName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get partition info: %v", err)
	}

	// Parse the output to get start offset
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.Contains(line, partName) {
			start, err := strconv.ParseUint(fields[0], 10, 64)
			if err == nil {
				info.StartOffset = start
				break
			}
		}
	}

	// Get sector size information
	diskName, _, err := ParsePartitionName(partName)
	if err != nil {
		// Try to extract disk name another way
		diskName = strings.TrimRight(partName, "0123456789ps")
	}

	// Get disk info for sector size
	cmd = exec.Command("diskinfo", diskName)
	output, err = cmd.CombinedOutput()
	if err == nil {
		fields := strings.Fields(string(output))
		if len(fields) >= 2 {
			sectorSize, err := strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				info.SectorSize = sectorSize
			}
		}
	}

	// Default sector size if we couldn't determine it
	if info.SectorSize == 0 {
		info.SectorSize = 512
	}

	// Calculate physical sector size (often 4K for modern drives)
	info.PhysicalSize = Align4K

	// Check alignment
	startBytes := info.StartOffset * info.SectorSize
	info.IsAligned, info.AlignmentType, info.Recommendation = checkAlignment(startBytes)

	return info, nil
}

// checkAlignment determines if a byte offset is aligned and provides recommendations
func checkAlignment(offset uint64) (bool, string, string) {
	// Check various alignment levels
	if offset%Align4M == 0 {
		return true, "4 MiB aligned", "Optimal alignment for SSDs"
	}
	if offset%Align1M == 0 {
		return true, "1 MiB aligned", "Recommended alignment for modern drives"
	}
	if offset%Align128K == 0 {
		return true, "128 KiB aligned", "Good alignment, but 1 MiB recommended"
	}
	if offset%Align4K == 0 {
		return true, "4 KiB aligned", "Minimum alignment, consider 1 MiB for better performance"
	}

	// Not aligned
	return false, "Misaligned", "Partition should be aligned to at least 1 MiB boundary for optimal performance"
}

// CalculateAlignedOffset calculates the next aligned offset for a given start position
func CalculateAlignedOffset(offset, alignment uint64) uint64 {
	if offset%alignment == 0 {
		return offset
	}
	return ((offset / alignment) + 1) * alignment
}

// GetOptimalAlignment returns the recommended alignment for a disk type
func GetOptimalAlignment(diskName string) uint64 {
	// Check if SSD using rotation rate
	cmd := exec.Command("diskinfo", "-v", diskName)
	output, err := cmd.CombinedOutput()
	if err == nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "non-rotating") || strings.Contains(outputStr, "# Rotation rate") && strings.Contains(outputStr, "0") {
			// SSD - use 4 MiB alignment
			return Align4M
		}
	}

	// Default to 1 MiB for HDDs and unknown types
	return Align1M
}

// AlignPartitionSize ensures partition size is aligned to sector boundaries
func AlignPartitionSize(size, sectorSize uint64) uint64 {
	if size%sectorSize == 0 {
		return size
	}
	return (size / sectorSize) * sectorSize
}

// CheckDiskAlignment checks alignment of all partitions on a disk
func CheckDiskAlignment(diskName string) ([]AlignmentInfo, error) {
	disks, err := GetDisks()
	if err != nil {
		return nil, err
	}

	var results []AlignmentInfo
	for _, disk := range disks {
		if disk.Name == diskName {
			for _, part := range disk.Partitions {
				info, err := CheckPartitionAlignment(part.Name)
				if err != nil {
					continue
				}
				results = append(results, *info)
			}
			break
		}
	}

	return results, nil
}

// FormatAlignmentInfo returns a human-readable alignment report
func FormatAlignmentInfo(info *AlignmentInfo) string {
	status := "✓ ALIGNED"
	if !info.IsAligned {
		status = "✗ MISALIGNED"
	}

	return fmt.Sprintf("%s: %s\n  Start: %d sectors (%d bytes)\n  Type: %s\n  Recommendation: %s",
		info.Partition, status, info.StartOffset, info.StartOffset*info.SectorSize,
		info.AlignmentType, info.Recommendation)
}

// CreateAlignedPartition creates a partition with optimal alignment
func CreateAlignedPartition(disk string, size uint64, fsType string, alignment uint64) error {
	// Get current disk info to find free space
	disks, err := GetDisks()
	if err != nil {
		return err
	}

	var targetDisk *Disk
	for i, d := range disks {
		if d.Name == disk {
			targetDisk = &disks[i]
			break
		}
	}

	if targetDisk == nil {
		return fmt.Errorf("disk %s not found", disk)
	}

	// Calculate aligned start position
	// For now, we'll use gpart's default behavior which typically aligns to 1M
	// In the future, we could add custom alignment using gpart's -a flag

	// Create partition normally (gpart handles alignment automatically in modern FreeBSD)
	return CreatePartition(disk, size, fsType)
}

// GetAlignmentSummary returns a summary of alignment status for a disk
func GetAlignmentSummary(diskName string) (aligned, misaligned int, err error) {
	results, err := CheckDiskAlignment(diskName)
	if err != nil {
		return 0, 0, err
	}

	for _, info := range results {
		if info.IsAligned {
			aligned++
		} else {
			misaligned++
		}
	}

	return aligned, misaligned, nil
}
