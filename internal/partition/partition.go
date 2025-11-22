package partition

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Partition struct {
	Name       string
	Type       string
	Size       uint64
	Start      uint64
	End        uint64
	FileSystem string
	Label      string
	MountPoint string
}

type Disk struct {
	Name        string
	Model       string
	Size        uint64
	SectorSize  uint64
	Scheme      string
	Partitions  []Partition
	Device      string
}

func GetDisks() ([]Disk, error) {
	cmd := exec.Command("geom", "disk", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute geom disk list: %w (output: %s)", err, string(output))
	}

	disks := parseGeomDiskList(string(output))

	for i := range disks {
		parts, err := getPartitions(disks[i].Name)
		if err != nil {
			continue
		}
		disks[i].Partitions = parts
	}

	return disks, nil
}

func parseGeomDiskList(output string) []Disk {
	var disks []Disk
	lines := strings.Split(output, "\n")

	var currentDisk *Disk

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Geom name:") {
			if currentDisk != nil {
				disks = append(disks, *currentDisk)
			}
			name := strings.TrimSpace(strings.TrimPrefix(line, "Geom name:"))
			currentDisk = &Disk{
				Name:   name,
				Device: "/dev/" + name,
			}
		} else if currentDisk != nil {
			if strings.HasPrefix(line, "Mediasize:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					size, _ := strconv.ParseUint(parts[1], 10, 64)
					currentDisk.Size = size
				}
			} else if strings.HasPrefix(line, "Sectorsize:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					size, _ := strconv.ParseUint(parts[1], 10, 64)
					currentDisk.SectorSize = size
				}
			} else if strings.HasPrefix(line, "descr:") {
				currentDisk.Model = strings.TrimSpace(strings.TrimPrefix(line, "descr:"))
			}
		}
	}

	if currentDisk != nil {
		disks = append(disks, *currentDisk)
	}

	return disks
}

func getPartitions(diskName string) ([]Partition, error) {
	cmd := exec.Command("gpart", "show", "-p", diskName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get partitions: %w", err)
	}

	return parseGpartShow(string(output))
}

func parseGpartShow(output string) ([]Partition, error) {
	var partitions []Partition
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "=>") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 4 {
			start, err1 := strconv.ParseUint(fields[0], 10, 64)
			size, err2 := strconv.ParseUint(fields[1], 10, 64)

			if err1 == nil && err2 == nil {
				part := Partition{
					Start: start,
					Size:  size,
					End:   start + size,
				}

				if len(fields) >= 3 {
					part.Type = fields[2]
				}
				if len(fields) >= 4 {
					part.Name = fields[3]
				}

				if part.Name != "" && !strings.HasPrefix(part.Name, "-") {
					fs, _ := getFileSystem(part.Name)
					part.FileSystem = fs

					mp, _ := getMountPoint(part.Name)
					part.MountPoint = mp

					partitions = append(partitions, part)
				}
			}
		}
	}

	return partitions, nil
}

func getFileSystem(partName string) (string, error) {
	cmd := exec.Command("file", "-s", "/dev/"+partName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	outStr := string(output)
	if strings.Contains(outStr, "UFS") {
		return "UFS", nil
	} else if strings.Contains(outStr, "ZFS") {
		return "ZFS", nil
	} else if strings.Contains(outStr, "FAT") {
		return "FAT32", nil
	} else if strings.Contains(outStr, "ext2") || strings.Contains(outStr, "ext3") || strings.Contains(outStr, "ext4") {
		return "ext4", nil
	} else if strings.Contains(outStr, "swap") {
		return "swap", nil
	}

	return "unknown", nil
}

func getMountPoint(partName string) (string, error) {
	cmd := exec.Command("mount")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/dev/"+partName) {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2], nil
			}
		}
	}

	return "", nil
}

func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}

	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
}
