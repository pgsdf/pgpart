package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/pgsdf/pgpart/internal/partition"
)

// CLI manages the command-line interface
type CLI struct {
	args []string
}

// NewCLI creates a new CLI instance
func NewCLI(args []string) *CLI {
	return &CLI{args: args}
}

// Run executes the CLI based on arguments
func (c *CLI) Run() int {
	if len(c.args) < 2 {
		c.printUsage()
		return 1
	}

	command := c.args[1]

	switch command {
	case "list":
		return c.listCommand()
	case "create":
		return c.createCommand()
	case "delete":
		return c.deleteCommand()
	case "format":
		return c.formatCommand()
	case "resize":
		return c.resizeCommand()
	case "copy":
		return c.copyCommand()
	case "info":
		return c.infoCommand()
	case "help", "-h", "--help":
		c.printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		c.printUsage()
		return 1
	}
}

// printUsage prints CLI usage information
func (c *CLI) printUsage() {
	fmt.Println("PGPart - Partition Manager for FreeBSD/GhostBSD")
	fmt.Println("\nUsage:")
	fmt.Println("  pgpart [command] [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  list                    List all disks and partitions")
	fmt.Println("  create <disk> <size> <fstype>")
	fmt.Println("                          Create a new partition")
	fmt.Println("  delete <disk> <index>   Delete a partition")
	fmt.Println("  format <partition> <fstype>")
	fmt.Println("                          Format a partition")
	fmt.Println("  resize <disk> <index> <size>")
	fmt.Println("                          Resize a partition")
	fmt.Println("  copy <source> <dest>    Copy partition data")
	fmt.Println("  info <disk>             Show detailed disk information")
	fmt.Println("  help                    Show this help message")
	fmt.Println("\nOptions:")
	fmt.Println("  -gui                    Launch graphical interface (default if no command)")
	fmt.Println("\nExamples:")
	fmt.Println("  pgpart list")
	fmt.Println("  pgpart create ada0 10G ufs")
	fmt.Println("  pgpart delete ada0 3")
	fmt.Println("  pgpart format ada0p3 ext4")
	fmt.Println("  pgpart resize ada0 2 20G")
	fmt.Println("  pgpart copy ada0p1 ada0p2")
	fmt.Println("  pgpart info ada0")
	fmt.Println("\nNote: Most operations require root privileges")
}

// listCommand lists all disks and partitions
func (c *CLI) listCommand() int {
	disks, err := partition.GetDisks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting disks: %v\n", err)
		return 1
	}

	if len(disks) == 0 {
		fmt.Println("No disks found")
		return 0
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DISK\tSIZE\tSCHEME\tPARTITIONS")
	fmt.Fprintln(w, "----\t----\t------\t----------")

	for _, disk := range disks {
		sizeGB := float64(disk.Size) / (1024 * 1024 * 1024)
		fmt.Fprintf(w, "%s\t%.2f GB\t%s\t%d\n", disk.Name, sizeGB, disk.Scheme, len(disk.Partitions))

		if len(disk.Partitions) > 0 {
			fmt.Fprintln(w, "\nPARTITION\tSIZE\tTYPE\tFILESYSTEM\tMOUNT")
			fmt.Fprintln(w, "---------\t----\t----\t----------\t-----")
			for _, part := range disk.Partitions {
				partSizeGB := float64(part.Size) / (1024 * 1024 * 1024)
				mount := part.MountPoint
				if mount == "" {
					mount = "-"
				}
				fmt.Fprintf(w, "%s\t%.2f GB\t%s\t%s\t%s\n",
					part.Name, partSizeGB, part.Type, part.FileSystem, mount)
			}
			fmt.Fprintln(w, "")
		}
	}
	w.Flush()

	return 0
}

// createCommand creates a new partition
func (c *CLI) createCommand() int {
	fs := flag.NewFlagSet("create", flag.ExitOnError)
	if err := fs.Parse(c.args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		return 1
	}

	args := fs.Args()
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: pgpart create <disk> <size> <fstype>")
		fmt.Fprintln(os.Stderr, "Example: pgpart create ada0 10G ufs")
		return 1
	}

	disk := args[0]
	sizeStr := args[1]
	fstype := args[2]

	// Parse size (supports G, M suffixes)
	size, err := parseSize(sizeStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid size: %v\n", err)
		return 1
	}

	fmt.Printf("Creating partition on %s: size=%s, filesystem=%s\n", disk, sizeStr, fstype)

	if err := partition.CreatePartition(disk, size, fstype); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating partition: %v\n", err)
		return 1
	}

	fmt.Println("Partition created successfully")
	return 0
}

// deleteCommand deletes a partition
func (c *CLI) deleteCommand() int {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	force := fs.Bool("f", false, "Force deletion without confirmation")
	if err := fs.Parse(c.args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		return 1
	}

	args := fs.Args()
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: pgpart delete [-f] <disk> <index>")
		fmt.Fprintln(os.Stderr, "Example: pgpart delete ada0 3")
		return 1
	}

	disk := args[0]
	index := args[1]

	if !*force {
		fmt.Printf("Delete partition %s%s? This cannot be undone! (yes/no): ", disk, index)
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Deletion cancelled")
			return 0
		}
	}

	fmt.Printf("Deleting partition %s%s\n", disk, index)

	if err := partition.DeletePartition(disk, index); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting partition: %v\n", err)
		return 1
	}

	fmt.Println("Partition deleted successfully")
	return 0
}

// formatCommand formats a partition
func (c *CLI) formatCommand() int {
	fs := flag.NewFlagSet("format", flag.ExitOnError)
	force := fs.Bool("f", false, "Force format without confirmation")
	if err := fs.Parse(c.args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		return 1
	}

	args := fs.Args()
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: pgpart format [-f] <partition> <fstype>")
		fmt.Fprintln(os.Stderr, "Example: pgpart format ada0p3 ext4")
		fmt.Fprintln(os.Stderr, "Supported filesystems: ufs, fat32, ext2, ext3, ext4, ntfs")
		return 1
	}

	partName := args[0]
	fstype := args[1]

	if !*force {
		fmt.Printf("Format partition %s as %s? This will destroy all data! (yes/no): ", partName, fstype)
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			fmt.Println("Format cancelled")
			return 0
		}
	}

	fmt.Printf("Formatting %s as %s\n", partName, fstype)

	if err := partition.FormatPartition(partName, fstype); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting partition: %v\n", err)
		return 1
	}

	fmt.Println("Partition formatted successfully")
	return 0
}

// resizeCommand resizes a partition
func (c *CLI) resizeCommand() int {
	fs := flag.NewFlagSet("resize", flag.ExitOnError)
	if err := fs.Parse(c.args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		return 1
	}

	args := fs.Args()
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: pgpart resize <disk> <index> <size>")
		fmt.Fprintln(os.Stderr, "Example: pgpart resize ada0 2 20G")
		return 1
	}

	disk := args[0]
	index := args[1]
	sizeStr := args[2]

	size, err := parseSize(sizeStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid size: %v\n", err)
		return 1
	}

	fmt.Printf("Resizing partition %s%s to %s\n", disk, index, sizeStr)

	if err := partition.ResizePartition(disk, index, size); err != nil {
		fmt.Fprintf(os.Stderr, "Error resizing partition: %v\n", err)
		return 1
	}

	fmt.Println("Partition resized successfully")
	return 0
}

// copyCommand copies a partition
func (c *CLI) copyCommand() int {
	fs := flag.NewFlagSet("copy", flag.ExitOnError)
	if err := fs.Parse(c.args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		return 1
	}

	args := fs.Args()
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: pgpart copy <source> <dest>")
		fmt.Fprintln(os.Stderr, "Example: pgpart copy ada0p1 ada0p2")
		return 1
	}

	source := args[0]
	dest := args[1]

	fmt.Printf("Copying %s to %s\n", source, dest)

	progressCallback := func(progress float64) {
		fmt.Printf("\rProgress: %.1f%%", progress)
	}

	if err := partition.CopyPartition(source, dest, progressCallback); err != nil {
		fmt.Fprintf(os.Stderr, "\nError copying partition: %v\n", err)
		return 1
	}

	fmt.Println("\nPartition copied successfully")
	return 0
}

// infoCommand shows detailed disk information
func (c *CLI) infoCommand() int {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	if err := fs.Parse(c.args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		return 1
	}

	args := fs.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: pgpart info <disk>")
		fmt.Fprintln(os.Stderr, "Example: pgpart info ada0")
		return 1
	}

	diskName := args[0]

	info, err := partition.GetDetailedDiskInfo(diskName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting disk info: %v\n", err)
		return 1
	}

	fmt.Printf("Disk Information: %s\n", diskName)
	fmt.Printf("==================%s\n", repeatChar('=', len(diskName)))
	fmt.Printf("Model:        %s\n", info.Model)
	fmt.Printf("Serial:       %s\n", info.Serial)
	fmt.Printf("Temperature:  %d°C\n", info.Temperature)
	fmt.Printf("Power Hours:  %d\n", info.PowerOnHours)
	fmt.Printf("SMART Status: %s\n", info.SMARTStatus)
	fmt.Printf("SMART Enabled: %t\n", info.SMARTEnabled)

	if len(info.Capabilities) > 0 {
		fmt.Println("\nCapabilities:")
		for _, cap := range info.Capabilities {
			fmt.Printf("  - %s\n", cap)
		}
	}

	if len(info.Attributes) > 0 {
		fmt.Println("\nSMART Attributes:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tVALUE\tWORST\tTHRESH\tSTATUS")
		fmt.Fprintln(w, "--\t----\t-----\t-----\t------\t------")
		for _, attr := range info.Attributes {
			fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%d\t%s\n",
				attr.ID, attr.Name, attr.Value, attr.Worst, attr.Threshold, attr.Status)
		}
		w.Flush()
	}

	return 0
}

// parseSize parses size strings like "10G", "512M", "1024"
func parseSize(sizeStr string) (uint64, error) {
	if len(sizeStr) == 0 {
		return 0, fmt.Errorf("empty size string")
	}

	// Check for suffix
	suffix := sizeStr[len(sizeStr)-1]
	var multiplier uint64 = 1

	numStr := sizeStr
	switch suffix {
	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024
		numStr = sizeStr[:len(sizeStr)-1]
	case 'M', 'm':
		multiplier = 1024 * 1024
		numStr = sizeStr[:len(sizeStr)-1]
	case 'K', 'k':
		multiplier = 1024
		numStr = sizeStr[:len(sizeStr)-1]
	}

	// Parse number
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", numStr)
	}

	if num <= 0 {
		return 0, fmt.Errorf("size must be positive")
	}

	return uint64(num * float64(multiplier)), nil
}

// repeatChar repeats a character n times
func repeatChar(char rune, n int) string {
	result := make([]rune, n)
	for i := range result {
		result[i] = char
	}
	return string(result)
}
