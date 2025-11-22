package partition

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Operation struct {
	Type        string
	Description string
}

func CheckPrivileges() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("this application requires root privileges to manage partitions")
	}
	return nil
}

func CreatePartition(disk string, size uint64, fsType string) error {
	if err := CheckPrivileges(); err != nil {
		return err
	}

	sizeStr := fmt.Sprintf("%dM", size/(1024*1024))

	cmd := exec.Command("gpart", "add", "-t", fsType, "-s", sizeStr, disk)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create partition: %w (output: %s)", err, string(output))
	}

	return nil
}

func DeletePartition(disk string, index string) error {
	if err := CheckPrivileges(); err != nil {
		return err
	}

	cmd := exec.Command("gpart", "delete", "-i", index, disk)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete partition: %w (output: %s)", err, string(output))
	}

	return nil
}

func FormatPartition(partition string, fsType string) error {
	if err := CheckPrivileges(); err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch strings.ToLower(fsType) {
	case "ufs":
		cmd = exec.Command("newfs", "-U", "/dev/"+partition)
	case "fat32":
		cmd = exec.Command("newfs_msdos", "-F", "32", "/dev/"+partition)
	case "ext2":
		// Check if mke2fs is available
		if _, err := exec.LookPath("mke2fs"); err != nil {
			return fmt.Errorf("mke2fs not found - install e2fsprogs package: pkg install e2fsprogs")
		}
		cmd = exec.Command("mke2fs", "-t", "ext2", "/dev/"+partition)
	case "ext3":
		if _, err := exec.LookPath("mke2fs"); err != nil {
			return fmt.Errorf("mke2fs not found - install e2fsprogs package: pkg install e2fsprogs")
		}
		cmd = exec.Command("mke2fs", "-t", "ext3", "/dev/"+partition)
	case "ext4":
		if _, err := exec.LookPath("mke2fs"); err != nil {
			return fmt.Errorf("mke2fs not found - install e2fsprogs package: pkg install e2fsprogs")
		}
		cmd = exec.Command("mke2fs", "-t", "ext4", "/dev/"+partition)
	case "ntfs":
		// Check if mkntfs is available
		if _, err := exec.LookPath("mkntfs"); err != nil {
			return fmt.Errorf("mkntfs not found - install ntfsprogs or ntfs-3g package: pkg install fusefs-ntfs")
		}
		cmd = exec.Command("mkntfs", "-f", "/dev/"+partition)
	case "zfs":
		return fmt.Errorf("ZFS pools must be created using 'zpool create' command directly")
	default:
		return fmt.Errorf("unsupported filesystem type: %s", fsType)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to format partition: %w (output: %s)", err, string(output))
	}

	return nil
}

func CreatePartitionTable(disk string, scheme string) error {
	if err := CheckPrivileges(); err != nil {
		return err
	}

	cmd := exec.Command("gpart", "create", "-s", scheme, disk)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create partition table: %w (output: %s)", err, string(output))
	}

	return nil
}

func DestroyPartitionTable(disk string) error {
	if err := CheckPrivileges(); err != nil {
		return err
	}

	cmd := exec.Command("gpart", "destroy", "-F", disk)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to destroy partition table: %w (output: %s)", err, string(output))
	}

	return nil
}

func ResizePartition(disk string, index string, newSize uint64) error {
	if err := CheckPrivileges(); err != nil {
		return err
	}

	sizeStr := fmt.Sprintf("%dM", newSize/(1024*1024))

	cmd := exec.Command("gpart", "resize", "-i", index, "-s", sizeStr, disk)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to resize partition: %w (output: %s)", err, string(output))
	}

	return nil
}
