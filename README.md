# PGPart - Partition Manager for FreeBSD/GhostBSD

A modern, graphical partition manager for FreeBSD and GhostBSD, similar to GParted but designed specifically for BSD systems. Built with Go and the Fyne UI framework.

## Features

- **Disk Detection**: Automatically detects all available disks using `geom`
- **Interactive Partition Visualization**:
  - Visual representation of partition layout with color-coded filesystems
  - Drag handles for intuitive partition resizing
  - Real-time size preview during drag operations
- **Partition Operations**:
  - Create new partition tables (GPT, MBR, BSD)
  - Create new partitions
  - Delete partitions
  - **Format partitions**: UFS, FAT32, ext2, ext3, ext4, NTFS
  - **Resize partitions with visual drag handles or slider interface**
  - Interactive resize dialog with min/max validation
- **Filesystem Support**:
  - **Detection**: UFS, ZFS, FAT32, swap, ext2, ext3, ext4, NTFS
  - **Formatting**: UFS (native), FAT32 (native), ext2/3/4 (requires e2fsprogs), NTFS (requires fusefs-ntfs)
- **Mount Point Display**: Shows current mount points for partitions
- **Modern GUI**: Clean, intuitive interface using Fyne

## Screenshots

The application provides a split-pane interface with:
- Left panel: List of available disks
- Right panel: Partition visualization and details
- Toolbar: Quick access to common operations

## Requirements

### System Requirements
- FreeBSD 12.0 or later, or GhostBSD
- Root privileges (for partition operations)
- X11 or Wayland display server

### Build Requirements
- Go 1.18 or later
- GCC or Clang (for Fyne CGO dependencies)
- Required system packages:
  ```bash
  pkg install go gcc git pkgconf mesa-libs libglvnd
  ```

### Optional Packages (for extended filesystem support)
- **e2fsprogs**: For ext2/ext3/ext4 filesystem formatting
  ```bash
  pkg install e2fsprogs
  ```
- **fusefs-ntfs**: For NTFS filesystem formatting
  ```bash
  pkg install fusefs-ntfs
  ```

## Installation

### From Source

1. Clone the repository:
   ```bash
   git clone https://github.com/pgsdf/pgpart.git
   cd pgpart
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the application:
   ```bash
   go build -o pgpart
   ```

4. Install (optional):
   ```bash
   sudo install -m 755 pgpart /usr/local/bin/
   ```

## Usage

### Running the Application

For full functionality, run with root privileges:

```bash
sudo ./pgpart
```

Or if installed system-wide:

```bash
sudo pgpart
```

### Basic Operations

#### Viewing Disks and Partitions
1. Launch the application
2. Select a disk from the left panel
3. View partition layout and details in the right panel

#### Creating a New Partition Table
1. Select a disk
2. Click the "New Partition Table" button in the toolbar
3. Choose the partition scheme (GPT, MBR, or BSD)
4. Confirm the operation

**Warning**: This will destroy all existing data on the disk!

#### Creating a New Partition
1. Select a disk with an existing partition table
2. Click the "New Partition" button
3. Enter the size in MB
4. Select the partition type:
   - `freebsd-ufs`: FreeBSD UFS filesystem
   - `freebsd-swap`: Swap partition
   - `freebsd-zfs`: ZFS partition
   - `ms-basic-data`: FAT32/NTFS compatible
5. Click "Create"

#### Deleting a Partition
1. Select a disk
2. Click the "Delete Partition" button
3. Select the partition to delete
4. Confirm the operation

#### Resizing a Partition

**Method 1: Visual Drag Handles**
1. Select a disk with partitions
2. View the partition layout visualization
3. Drag the resize handles on the left or right edge of a partition
4. Release to see the resize confirmation dialog
5. Confirm to apply the resize operation

**Method 2: Resize Dialog**
1. Select a disk
2. Click the "Resize Partition" button in the toolbar
3. Select the partition to resize
4. Use the slider or enter the new size in MB
5. Review the preview showing current size, new size, and difference
6. Confirm the operation

**Important Notes:**
- The dialog shows minimum and maximum allowed sizes
- You cannot resize a partition to overlap with adjacent partitions
- Minimum size is 10 MB
- Maximum size extends to the next partition or end of disk
- **Warning**: Resizing may result in data loss. Always backup first!

#### Formatting a Partition
1. Select a disk
2. Click the "Format" button
3. Select the partition
4. Choose the filesystem type:
   - **UFS** (native FreeBSD filesystem)
   - **FAT32** (compatible with Windows/Linux)
   - **ext2/ext3/ext4** (Linux filesystems - requires e2fsprogs package)
   - **NTFS** (Windows filesystem - requires fusefs-ntfs package)
5. Confirm the operation

**Important Notes:**
- **Warning**: Formatting will destroy all data on the partition!
- ext2/ext3/ext4 formatting requires: `pkg install e2fsprogs`
- NTFS formatting requires: `pkg install fusefs-ntfs`
- If required packages are missing, you'll see an error message with installation instructions
- ZFS pools must be created using the `zpool create` command directly

#### Refreshing the Disk List
Click the "Refresh" button in the toolbar to rescan all disks.

## Architecture

The application is organized into the following packages:

- `main`: Application entry point and theme configuration
- `internal/partition`: Core partition detection and management
  - `partition.go`: Disk and partition detection using geom/gpart
  - `operations.go`: Partition operations (create, delete, format, resize)
- `internal/ui`: User interface components
  - `mainwindow.go`: Main application window and UI logic
  - `partitionview.go`: Interactive partition visualization with drag handles
  - `resizedialog.go`: Advanced resize dialog with slider and validation

## BSD Tools Used

PGPart uses the following FreeBSD system utilities:

- `geom`: Disk geometry and information
- `gpart`: Partition table manipulation
- `newfs`: UFS filesystem creation
- `newfs_msdos`: FAT filesystem creation
- `mount`: Mount point detection
- `file`: Filesystem type detection

## Limitations

- Requires root privileges for most operations
- Some operations may require the disk to be unmounted
- Advanced features like partition alignment optimization are not yet implemented
- Online resizing is limited by the capabilities of the underlying filesystem

## Safety Features

- Privilege checking before destructive operations
- Confirmation dialogs for delete and format operations
- Error reporting with detailed messages
- Read-only mode when not running as root

## Development

### Project Structure
```
pgpart/
├── main.go                    # Application entry point
├── theme.go                   # UI theme configuration
├── internal/
│   ├── partition/
│   │   ├── partition.go       # Disk detection
│   │   └── operations.go      # Partition operations
│   └── ui/
│       └── mainwindow.go      # Main UI
├── go.mod                     # Go module definition
└── README.md                  # This file
```

### Building for Development

```bash
go run .
```

### Running Tests

```bash
go test ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the BSD 2-Clause License.

Copyright (c) 2025, Pacific Grove Software Distribution Foundation
Author: Vester (Vic) Thacker

See the [LICENSE](LICENSE) file for full license details.

## Acknowledgments

- Inspired by GParted
- Built with [Fyne](https://fyne.io/) UI toolkit
- Uses FreeBSD's powerful geom and gpart utilities

## Disclaimer

**USE AT YOUR OWN RISK!** Partition operations can result in data loss if used incorrectly. Always backup important data before performing partition operations.

## Support

For issues, questions, or contributions, please visit:
https://github.com/pgsdf/pgpart

## Future Enhancements

Planned features for future releases:

- [x] Partition resizing with visual drag handles ✅ **IMPLEMENTED**
- [x] Support for more filesystems (ext2/3/4, NTFS) ✅ **IMPLEMENTED**
- [ ] Partition copying and moving
- [ ] Detailed disk information (SMART status)
- [ ] Batch operations
- [ ] Undo/redo functionality
- [ ] Command-line interface for scripting
- [ ] Partition alignment optimization
- [ ] GPT attribute editing
- [ ] Online filesystem resize (grow/shrink while mounted)