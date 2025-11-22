package partition

import (
	"fmt"
	"os/exec"
	"strings"
)

// OnlineResizeCapability describes the online resize support for a filesystem
type OnlineResizeCapability struct {
	SupportsGrow    bool
	SupportsShrink  bool
	RequiresMounted bool
	Command         string
	Notes           string
}

// GetOnlineResizeCapability returns the online resize capability for a filesystem type
func GetOnlineResizeCapability(fsType string) OnlineResizeCapability {
	switch strings.ToLower(fsType) {
	case "ufs":
		return OnlineResizeCapability{
			SupportsGrow:    true,
			SupportsShrink:  false,
			RequiresMounted: true,
			Command:         "growfs",
			Notes:           "UFS can be grown while mounted using growfs. Cannot shrink online.",
		}
	case "ext3", "ext4":
		return OnlineResizeCapability{
			SupportsGrow:    true,
			SupportsShrink:  true,
			RequiresMounted: false, // Can resize both mounted and unmounted
			Command:         "resize2fs",
			Notes:           "ext3/ext4 support both online grow and shrink with resize2fs.",
		}
	case "ext2":
		return OnlineResizeCapability{
			SupportsGrow:    true,
			SupportsShrink:  true,
			RequiresMounted: false,
			Command:         "resize2fs",
			Notes:           "ext2 can be resized with resize2fs when unmounted.",
		}
	case "xfs":
		return OnlineResizeCapability{
			SupportsGrow:    true,
			SupportsShrink:  false,
			RequiresMounted: true,
			Command:         "xfs_growfs",
			Notes:           "XFS can be grown while mounted. Cannot shrink XFS filesystems.",
		}
	default:
		return OnlineResizeCapability{
			SupportsGrow:    false,
			SupportsShrink:  false,
			RequiresMounted: false,
			Command:         "",
			Notes:           fmt.Sprintf("Online resize not supported for %s", fsType),
		}
	}
}

// CanResizeOnline checks if a partition can be resized online
func CanResizeOnline(part *Partition, grow bool) (bool, string) {
	if part.FileSystem == "" || part.FileSystem == "unknown" {
		return false, "Unknown filesystem type"
	}

	capability := GetOnlineResizeCapability(part.FileSystem)

	if grow && !capability.SupportsGrow {
		return false, fmt.Sprintf("%s does not support online grow", part.FileSystem)
	}

	if !grow && !capability.SupportsShrink {
		return false, fmt.Sprintf("%s does not support online shrink", part.FileSystem)
	}

	// Check if filesystem is mounted when required
	if capability.RequiresMounted && part.MountPoint == "" {
		return false, fmt.Sprintf("Filesystem must be mounted for online resize (using %s)", capability.Command)
	}

	return true, ""
}

// ResizeFilesystemOnline resizes a mounted filesystem
func ResizeFilesystemOnline(part *Partition, newSizeBytes uint64) error {
	if part.FileSystem == "" || part.FileSystem == "unknown" {
		return fmt.Errorf("cannot resize unknown filesystem type")
	}

	capability := GetOnlineResizeCapability(part.FileSystem)

	// Determine if we're growing or shrinking
	currentSizeBytes := part.Size * 512
	isGrow := newSizeBytes > currentSizeBytes

	if isGrow && !capability.SupportsGrow {
		return fmt.Errorf("%s does not support online grow", part.FileSystem)
	}

	if !isGrow && !capability.SupportsShrink {
		return fmt.Errorf("%s does not support online shrink", part.FileSystem)
	}

	// Perform filesystem-specific resize
	switch strings.ToLower(part.FileSystem) {
	case "ufs":
		return resizeUFSOnline(part)
	case "ext2", "ext3", "ext4":
		return resizeExt234Online(part, newSizeBytes)
	case "xfs":
		return resizeXFSOnline(part)
	default:
		return fmt.Errorf("online resize not implemented for %s", part.FileSystem)
	}
}

// resizeUFSOnline grows a UFS filesystem using growfs
func resizeUFSOnline(part *Partition) error {
	if part.MountPoint == "" {
		return fmt.Errorf("UFS filesystem must be mounted for online resize")
	}

	// Run growfs on the mounted filesystem
	// growfs will automatically grow to fill the partition
	cmd := exec.Command("growfs", "-y", part.MountPoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("growfs failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// resizeExt234Online resizes ext2/ext3/ext4 filesystem using resize2fs
func resizeExt234Online(part *Partition, newSizeBytes uint64) error {
	// resize2fs can work on mounted or unmounted filesystems
	// For ext3/ext4, it can resize while mounted
	// Size is specified in K (1024-byte blocks)
	newSizeK := newSizeBytes / 1024

	var cmd *exec.Cmd
	if newSizeK > 0 {
		// Specify target size
		cmd = exec.Command("resize2fs", part.Name, fmt.Sprintf("%dK", newSizeK))
	} else {
		// Grow to fill partition
		cmd = exec.Command("resize2fs", part.Name)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("resize2fs failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// resizeXFSOnline grows an XFS filesystem using xfs_growfs
func resizeXFSOnline(part *Partition) error {
	if part.MountPoint == "" {
		return fmt.Errorf("XFS filesystem must be mounted for online resize")
	}

	// xfs_growfs grows to fill the partition
	cmd := exec.Command("xfs_growfs", part.MountPoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xfs_growfs failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// PerformOnlineResize performs a complete online resize operation
// This includes resizing the partition AND the filesystem
func PerformOnlineResize(diskName, partIndex string, newSizeBytes uint64, part *Partition) error {
	// First, verify online resize is possible
	isGrow := newSizeBytes > (part.Size * 512)
	canResize, reason := CanResizeOnline(part, isGrow)
	if !canResize {
		return fmt.Errorf("cannot perform online resize: %s", reason)
	}

	// Step 1: Resize the partition
	if err := ResizePartition(diskName, partIndex, newSizeBytes); err != nil {
		return fmt.Errorf("failed to resize partition: %v", err)
	}

	// Step 2: Resize the filesystem online
	if err := ResizeFilesystemOnline(part, newSizeBytes); err != nil {
		// Partition was resized but filesystem wasn't
		// Log the error but don't fail completely
		return fmt.Errorf("partition resized but filesystem resize failed: %v\nYou may need to resize the filesystem manually", err)
	}

	return nil
}

// GetOnlineResizeRecommendation returns a user-friendly recommendation for resize
func GetOnlineResizeRecommendation(part *Partition, grow bool) string {
	capability := GetOnlineResizeCapability(part.FileSystem)

	if grow && capability.SupportsGrow {
		if capability.RequiresMounted && part.MountPoint == "" {
			return fmt.Sprintf("✓ Online grow supported with %s (filesystem must be mounted)", capability.Command)
		}
		return fmt.Sprintf("✓ Online grow supported with %s", capability.Command)
	}

	if !grow && capability.SupportsShrink {
		return fmt.Sprintf("✓ Online shrink supported with %s", capability.Command)
	}

	if grow && !capability.SupportsGrow {
		return "⚠ Online grow not supported - filesystem must be unmounted for resize"
	}

	if !grow && !capability.SupportsShrink {
		return "⚠ Online shrink not supported - filesystem must be unmounted for resize"
	}

	return "⚠ Online resize not supported for this filesystem"
}
