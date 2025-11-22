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
  - **Copy partitions**: Clone partition data to another partition
  - **Move partitions**: Copy partition and delete source
  - Interactive resize dialog with min/max validation
  - Progress monitoring for long operations
- **Filesystem Support**:
  - **Detection**: UFS, ZFS, FAT32, swap, ext2, ext3, ext4, NTFS
  - **Formatting**: UFS (native), FAT32 (native), ext2/3/4 (requires e2fsprogs), NTFS (requires fusefs-ntfs)
- **Mount Point Display**: Shows current mount points for partitions
- **Detailed Disk Information**:
  - SMART status monitoring and health assessment
  - Disk temperature, power-on hours, power cycle count
  - Individual SMART attribute details with status indicators
  - Capability detection (TRIM support, SSD/HDD identification)
- **Batch Operations**:
  - Queue multiple partition operations for sequential execution
  - Supports format, delete, resize, copy operations in batch mode
  - Reorder operations with move up/down controls
  - Progress tracking across all operations
  - Stop on error or continue options
  - Review and manage operation queue before execution
- **Undo/Redo Functionality**:
  - Operation history tracking for all partition changes
  - Undo reversible operations (create, resize)
  - Redo previously undone operations
  - Clear indication of which operations can be reversed
  - Confirmation dialogs before undo/redo execution
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
- **smartmontools**: For detailed disk information and SMART status monitoring
  ```bash
  pkg install smartmontools
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

#### Copying a Partition
1. Click the "Copy Partition" button in the toolbar
2. Select the source partition (partition to copy from)
3. Select the destination partition (where to copy to)
4. Review the warning - destination data will be overwritten
5. Confirm the operation
6. Monitor the progress bar during the copy operation

**Important Notes:**
- Destination partition must be equal or larger than source
- All data on the destination partition will be destroyed
- The operation may take several minutes depending on partition size
- Progress is shown with percentage and elapsed time
- Source partition remains unchanged (read-only operation)

#### Moving a Partition
1. Click the "Move Partition" button in the toolbar
2. Select the source partition (partition to move)
3. Select the destination partition (where to move to)
4. Review the warning - this will copy data and delete source
5. Confirm the operation
6. Monitor the progress during the move operation

**Important Notes:**
- Move = Copy + Delete source partition
- Destination must be equal or larger than source
- Source partition will be deleted after successful copy
- All data on destination will be destroyed
- **Cannot be undone** - ensure you have backups!
- Operation may take several minutes

#### Viewing Detailed Disk Information
1. Select a disk from the left panel
2. Click the "Disk Info" button in the toolbar
3. View comprehensive disk information in the tabbed dialog:
   - **General**: Model, serial number, firmware version, capacity, temperature, power-on hours
   - **SMART Status**: Overall health status with color-coded indicators (green=PASSED, red=FAILED, orange=UNKNOWN)
   - **SMART Attributes**: Detailed list of all SMART attributes with current values, worst values, thresholds, and status
   - **Capabilities**: Disk type (SSD/HDD), TRIM support, and other features

**Important Notes:**
- Requires smartmontools package: `pkg install smartmontools`
- Temperature warnings appear if disk temperature exceeds 60°C
- SMART data requires the disk to support SMART monitoring
- Some attributes may not be available on all disk models

#### Using Batch Operations
Batch operations allow you to queue multiple partition operations and execute them sequentially:

1. Click the "Batch Operations" button in the toolbar
2. Add operations to the queue using the operation buttons:
   - **Add Format**: Queue a partition format operation
   - **Add Delete**: Queue a partition deletion
   - **Add Resize**: Queue a partition resize operation
   - **Add Copy**: Queue a partition copy operation
3. Manage your queue:
   - **Remove Selected**: Remove an operation from the queue
   - **Clear All**: Remove all operations
   - **Move Up/Down**: Reorder operations in the queue
4. Configure execution options:
   - **Stop on error**: Check to halt execution if any operation fails
   - Uncheck to continue executing remaining operations after failures
5. Click **Execute All** to run all queued operations

**Operation Status Indicators:**
- ⏸ Pending - Operation queued but not started
- ▶ Running - Operation currently executing
- ✓ Completed - Operation finished successfully
- ✗ Failed - Operation failed with error

**Important Notes:**
- Operations execute in queue order (top to bottom)
- All operations are destructive and **cannot be undone**
- Review your queue carefully before executing
- Progress bar shows overall completion across all operations
- Failed operations show error details in the status
- You can reorder operations before execution to optimize efficiency

**Best Practices:**
- Group similar operations together (e.g., all deletions, then all formats)
- Delete operations should typically come before create operations
- Format operations should come after partition creation
- Always verify source/destination for copy operations
- Use "Stop on error" for critical sequences where order matters

#### Using Undo/Redo
PGPart tracks partition operations and allows you to undo reversible changes:

**Reversible Operations:**
- **Create Partition** - Can be undone by deleting the created partition
- **Resize Partition** - Can be undone by resizing back to original size

**Non-Reversible Operations (data destructive):**
- **Delete Partition** - Cannot restore deleted data
- **Format Partition** - Cannot restore previous filesystem or data
- **Copy Partition** - Cannot "uncopy" data
- **Move Partition** - Cannot restore (source was deleted)

**How to Use:**
1. Click the **Undo** button (◀) in the toolbar to reverse the last reversible operation
2. Click the **Redo** button (▶) to re-apply an undone operation
3. Confirm the undo/redo action in the dialog that appears

**Important Limitations:**
- Undo only reverses structural changes, not data
- Undoing a partition resize requires sufficient free space
- Operation history is lost when you close the application
- Some operations cannot be undone and are marked as such in history
- Always backup important data before performing partition operations

#### Refreshing the Disk List
Click the "Refresh" button in the toolbar to rescan all disks.

## Architecture

The application is organized into the following packages:

- `main`: Application entry point and theme configuration
- `internal/partition`: Core partition detection and management
  - `partition.go`: Disk and partition detection using geom/gpart
  - `operations.go`: Partition operations (create, delete, format, resize)
  - `copy.go`: Partition copying and moving with progress tracking
  - `diskinfo.go`: Detailed disk information and SMART status retrieval
  - `batch.go`: Batch operation queue management and execution
  - `history.go`: Operation history tracking and undo/redo management
- `internal/ui`: User interface components
  - `mainwindow.go`: Main application window and UI logic
  - `partitionview.go`: Interactive partition visualization with drag handles
  - `resizedialog.go`: Advanced resize dialog with slider and validation
  - `copydialog.go`: Copy and move partition dialogs with progress bars
  - `diskinfodialog.go`: Detailed disk information display with SMART data
  - `batchdialog.go`: Batch operations queue manager with execution controls

## BSD Tools Used

PGPart uses the following FreeBSD system utilities:

- `geom`: Disk geometry and information
- `gpart`: Partition table manipulation
- `newfs`: UFS filesystem creation
- `newfs_msdos`: FAT filesystem creation
- `mount`: Mount point detection
- `file`: Filesystem type detection
- `fstyp`: FreeBSD native filesystem detection
- `diskinfo`: Partition size information
- `dd`: Disk data copying (with progress monitoring)
- `sha256`: Partition data verification
- `smartctl`: SMART status monitoring and disk health assessment

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
│   │   ├── operations.go      # Partition operations
│   │   ├── copy.go            # Partition copying and moving
│   │   ├── diskinfo.go        # SMART status and disk info
│   │   ├── batch.go           # Batch operation queue
│   │   └── history.go         # Undo/redo history tracking
│   └── ui/
│       ├── mainwindow.go      # Main UI
│       ├── partitionview.go   # Partition visualization
│       ├── resizedialog.go    # Resize dialog
│       ├── copydialog.go      # Copy/move dialogs
│       ├── diskinfodialog.go  # Disk information dialog
│       └── batchdialog.go     # Batch operations manager
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
- [x] Partition copying and moving ✅ **IMPLEMENTED**
- [x] Detailed disk information (SMART status) ✅ **IMPLEMENTED**
- [x] Batch operations ✅ **IMPLEMENTED**
- [x] Undo/redo functionality ✅ **IMPLEMENTED**
- [ ] Command-line interface for scripting
- [ ] Partition alignment optimization
- [ ] GPT attribute editing
- [ ] Online filesystem resize (grow/shrink while mounted)