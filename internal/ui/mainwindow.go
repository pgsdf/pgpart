package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/pgsdf/pgpart/internal/partition"
)

type MainWindow struct {
	window        fyne.Window
	diskList      *widget.List
	disks         []partition.Disk
	selectedDisk  int
	partitionView *fyne.Container
	infoLabel     *widget.Label
	history       *partition.OperationHistory
	undoBtn       *widget.Button
	redoBtn       *widget.Button
}

func NewMainWindow(app fyne.App) *MainWindow {
	mw := &MainWindow{
		window:       app.NewWindow("PGPart - Partition Manager"),
		selectedDisk: -1,
		history:      partition.NewOperationHistory(),
	}

	mw.window.Resize(fyne.NewSize(900, 600))
	mw.setupUI()
	mw.refreshDisks()

	return mw
}

// createToolbarButton creates a toolbar button with an icon and text
func (mw *MainWindow) createToolbarButton(icon fyne.Resource, text string, tapped func()) *widget.Button {
	btn := widget.NewButtonWithIcon(text, icon, tapped)
	btn.Importance = widget.LowImportance
	return btn
}

func (mw *MainWindow) setupUI() {
	mw.infoLabel = widget.NewLabel("Select a disk to view partitions")

	// Create toolbar buttons with labels
	undoBtn := mw.createToolbarButton(theme.NavigateBackIcon(), "Undo", mw.performUndo)
	redoBtn := mw.createToolbarButton(theme.NavigateNextIcon(), "Redo", mw.performRedo)
	refreshBtn := mw.createToolbarButton(theme.ViewRefreshIcon(), "Refresh", mw.refreshDisks)
	infoBtn := mw.createToolbarButton(theme.InfoIcon(), "Disk Info", mw.showDiskInfo)
	newTableBtn := mw.createToolbarButton(theme.StorageIcon(), "New Table", mw.showNewPartitionTableDialog)
	newPartBtn := mw.createToolbarButton(theme.ContentAddIcon(), "New Partition", mw.showNewPartitionDialog)
	copyBtn := mw.createToolbarButton(theme.ContentCopyIcon(), "Copy", mw.showCopyDialog)
	moveBtn := mw.createToolbarButton(theme.NavigateNextIcon(), "Move", mw.showMoveDialog)
	resizeBtn := mw.createToolbarButton(theme.ZoomInIcon(), "Resize", mw.showResizeDialog)
	deleteBtn := mw.createToolbarButton(theme.DeleteIcon(), "Delete", mw.showDeletePartitionDialog)
	formatBtn := mw.createToolbarButton(theme.DocumentCreateIcon(), "Format", mw.showFormatDialog)
	bootableBtn := mw.createToolbarButton(theme.ConfirmIcon(), "Toggle Boot", mw.toggleBootableDialog)
	attrBtn := mw.createToolbarButton(theme.SettingsIcon(), "Attributes", mw.showAttributesDialog)
	batchBtn := mw.createToolbarButton(theme.ListIcon(), "Batch", mw.showBatchDialog)

	// Create toolbar with buttons
	toolbar := container.NewHBox(
		undoBtn,
		redoBtn,
		widget.NewSeparator(),
		refreshBtn,
		infoBtn,
		widget.NewSeparator(),
		newTableBtn,
		newPartBtn,
		widget.NewSeparator(),
		copyBtn,
		moveBtn,
		widget.NewSeparator(),
		resizeBtn,
		deleteBtn,
		formatBtn,
		widget.NewSeparator(),
		bootableBtn,
		attrBtn,
		widget.NewSeparator(),
		batchBtn,
	)

	mw.diskList = widget.NewList(
		func() int {
			return len(mw.disks)
		},
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel(""),
				widget.NewLabel(""),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			cont := item.(*fyne.Container)
			disk := mw.disks[id]

			nameLabel := cont.Objects[0].(*widget.Label)
			sizeLabel := cont.Objects[1].(*widget.Label)

			nameLabel.SetText(fmt.Sprintf("%s - %s", disk.Name, disk.Model))
			sizeLabel.SetText(fmt.Sprintf("Size: %s, Scheme: %s", partition.FormatBytes(disk.Size), disk.Scheme))
		},
	)

	mw.diskList.OnSelected = func(id widget.ListItemID) {
		mw.selectedDisk = id
		mw.updatePartitionView()
	}

	mw.partitionView = container.NewVBox()

	leftPanel := container.NewBorder(
		widget.NewLabel("Disks:"),
		nil, nil, nil,
		mw.diskList,
	)

	rightPanel := container.NewBorder(
		mw.infoLabel,
		nil, nil, nil,
		container.NewScroll(mw.partitionView),
	)

	split := container.NewHSplit(leftPanel, rightPanel)
	split.Offset = 0.3

	content := container.NewBorder(
		toolbar,
		nil, nil, nil,
		split,
	)

	mw.window.SetContent(content)
}

func (mw *MainWindow) refreshDisks() {
	disks, err := partition.GetDisks()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to get disks: %w", err), mw.window)
		return
	}

	mw.disks = disks
	mw.diskList.Refresh()

	if mw.selectedDisk >= 0 && mw.selectedDisk < len(mw.disks) {
		mw.updatePartitionView()
	}
}

func (mw *MainWindow) updatePartitionView() {
	if mw.selectedDisk < 0 || mw.selectedDisk >= len(mw.disks) {
		return
	}

	disk := mw.disks[mw.selectedDisk]
	mw.infoLabel.SetText(fmt.Sprintf("Disk: %s (%s) - %s", disk.Name, disk.Model, partition.FormatBytes(disk.Size)))

	mw.partitionView.Objects = nil

	interactiveView := NewInteractivePartitionView(&disk, mw.window, mw.refreshDisks)
	mw.partitionView.Add(container.NewVBox(
		widget.NewLabel("Partition Layout (drag edges to resize):"),
		interactiveView,
	))

	if len(disk.Partitions) == 0 {
		mw.partitionView.Add(widget.NewLabel("No partitions found"))
	} else {
		legend := mw.createColorLegend()
		mw.partitionView.Add(legend)

		for _, part := range disk.Partitions {
			partCard := mw.createPartitionCard(part)
			mw.partitionView.Add(partCard)
		}
	}

	mw.partitionView.Refresh()
}

func (mw *MainWindow) createPartitionVisual(disk partition.Disk) *fyne.Container {
	visual := container.NewHBox()

	if len(disk.Partitions) == 0 {
		rect := canvas.NewRectangle(color.RGBA{R: 200, G: 200, B: 200, A: 255})
		rect.SetMinSize(fyne.NewSize(600, 40))
		visual.Add(rect)
		return container.NewVBox(
			widget.NewLabel("Partition Layout:"),
			visual,
		)
	}

	for _, part := range disk.Partitions {
		partColor := getPartitionColor(part.FileSystem)
		rect := canvas.NewRectangle(partColor)

		width := float32(600) * float32(part.Size) / float32(disk.Size)
		if width < 20 {
			width = 20
		}
		rect.SetMinSize(fyne.NewSize(width, 40))

		visual.Add(rect)
	}

	return container.NewVBox(
		widget.NewLabel("Partition Layout:"),
		visual,
	)
}

func getPartitionColor(fsType string) color.Color {
	switch fsType {
	case "UFS":
		return color.RGBA{R: 70, G: 130, B: 230, A: 255} // Steel Blue
	case "ZFS":
		return color.RGBA{R: 50, G: 205, B: 50, A: 255} // Lime Green
	case "FAT32":
		return color.RGBA{R: 255, G: 165, B: 0, A: 255} // Orange
	case "swap":
		return color.RGBA{R: 220, G: 20, B: 60, A: 255} // Crimson Red
	case "ext2", "ext3", "ext4":
		return color.RGBA{R: 147, G: 51, B: 234, A: 255} // Purple (Linux ext family)
	case "NTFS":
		return color.RGBA{R: 0, G: 123, B: 255, A: 255} // Bright Blue (Windows)
	case "unknown":
		return color.RGBA{R: 169, G: 169, B: 169, A: 255} // Dark Gray
	default:
		// For any other filesystem types, use a neutral color
		return color.RGBA{R: 120, G: 120, B: 120, A: 255} // Medium Gray
	}
}

func (mw *MainWindow) createPartitionCard(part partition.Partition) *fyne.Container {
	nameLabel := widget.NewLabelWithStyle(part.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	typeLabel := widget.NewLabel(fmt.Sprintf("Type: %s", part.Type))
	sizeLabel := widget.NewLabel(fmt.Sprintf("Size: %s", partition.FormatBytes(part.Size*512)))
	fsLabel := widget.NewLabel(fmt.Sprintf("Filesystem: %s", part.FileSystem))

	var mountLabel *widget.Label
	if part.MountPoint != "" {
		mountLabel = widget.NewLabel(fmt.Sprintf("Mount: %s", part.MountPoint))
		mountLabel.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		mountLabel = widget.NewLabel("Mount: (not mounted)")
		mountLabel.TextStyle = fyne.TextStyle{Italic: true}
	}

	// Check for GPT attributes
	attrSummary := partition.GetAttributeSummary(part.Name)
	var attrLabel *widget.Label
	if attrSummary != "" {
		attrLabel = widget.NewLabel(fmt.Sprintf("Attributes: %s", attrSummary))
		attrLabel.TextStyle = fyne.TextStyle{Bold: true}
		// Use a color to highlight bootable partitions
		if strings.Contains(attrSummary, "Bootable") {
			attrLabel.Importance = widget.HighImportance
		}
	}

	cardItems := []fyne.CanvasObject{
		nameLabel,
		typeLabel,
		sizeLabel,
		fsLabel,
		mountLabel,
	}

	// Add attribute label if present
	if attrLabel != nil {
		cardItems = append(cardItems, attrLabel)
	}

	cardItems = append(cardItems, widget.NewSeparator())

	card := container.NewVBox(cardItems...)

	return card
}

func (mw *MainWindow) showNewPartitionTableDialog() {
	if mw.selectedDisk < 0 {
		dialog.ShowInformation("No Disk Selected", "Please select a disk first", mw.window)
		return
	}

	disk := mw.disks[mw.selectedDisk]

	schemeSelect := widget.NewSelect([]string{"GPT", "MBR", "BSD"}, nil)
	schemeSelect.SetSelected("GPT")

	dialog.ShowForm("Create New Partition Table", "Create", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Scheme", schemeSelect),
		},
		func(ok bool) {
			if !ok {
				return
			}

			err := partition.CreatePartitionTable(disk.Name, strings.ToLower(schemeSelect.Selected))
			if err != nil {
				dialog.ShowError(err, mw.window)
				return
			}

			dialog.ShowInformation("Success", "Partition table created successfully", mw.window)
			mw.refreshDisks()
		}, mw.window)
}

func (mw *MainWindow) showNewPartitionDialog() {
	if mw.selectedDisk < 0 {
		dialog.ShowInformation("No Disk Selected", "Please select a disk first", mw.window)
		return
	}

	disk := mw.disks[mw.selectedDisk]

	sizeEntry := widget.NewEntry()
	sizeEntry.SetPlaceHolder("1024")

	typeSelect := widget.NewSelect([]string{"freebsd-ufs", "freebsd-swap", "freebsd-zfs", "ms-basic-data"}, nil)
	typeSelect.SetSelected("freebsd-ufs")

	dialog.ShowForm("Create New Partition", "Create", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Size (MB)", sizeEntry),
			widget.NewFormItem("Type", typeSelect),
		},
		func(ok bool) {
			if !ok {
				return
			}

			var size uint64
			fmt.Sscanf(sizeEntry.Text, "%d", &size)
			if size == 0 {
				dialog.ShowError(fmt.Errorf("invalid size"), mw.window)
				return
			}

			err := partition.CreatePartition(disk.Name, size*1024*1024, typeSelect.Selected)
			if err != nil {
				dialog.ShowError(err, mw.window)
				return
			}

			dialog.ShowInformation("Success", "Partition created successfully", mw.window)
			mw.refreshDisks()
		}, mw.window)
}

func (mw *MainWindow) showDeletePartitionDialog() {
	if mw.selectedDisk < 0 {
		dialog.ShowInformation("No Disk Selected", "Please select a disk first", mw.window)
		return
	}

	disk := mw.disks[mw.selectedDisk]

	if len(disk.Partitions) == 0 {
		dialog.ShowInformation("No Partitions", "This disk has no partitions", mw.window)
		return
	}

	partNames := make([]string, len(disk.Partitions))
	for i, part := range disk.Partitions {
		partNames[i] = fmt.Sprintf("%s (%s)", part.Name, partition.FormatBytes(part.Size*512))
	}

	partSelect := widget.NewSelect(partNames, nil)

	dialog.ShowForm("Delete Partition", "Delete", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Partition", partSelect),
		},
		func(ok bool) {
			if !ok {
				return
			}

			selectedIdx := -1
			for i, name := range partNames {
				if name == partSelect.Selected {
					selectedIdx = i
					break
				}
			}

			if selectedIdx < 0 {
				return
			}

			parts := strings.Split(disk.Partitions[selectedIdx].Name, "p")
			if len(parts) < 2 {
				dialog.ShowError(fmt.Errorf("invalid partition name"), mw.window)
				return
			}
			index := parts[len(parts)-1]

			dialog.ShowConfirm("Confirm Delete",
				fmt.Sprintf("Are you sure you want to delete partition %s?", disk.Partitions[selectedIdx].Name),
				func(confirmed bool) {
					if !confirmed {
						return
					}

					err := partition.DeletePartition(disk.Name, index)
					if err != nil {
						dialog.ShowError(err, mw.window)
						return
					}

					dialog.ShowInformation("Success", "Partition deleted successfully", mw.window)
					mw.refreshDisks()
				}, mw.window)
		}, mw.window)
}

func (mw *MainWindow) showFormatDialog() {
	if mw.selectedDisk < 0 {
		dialog.ShowInformation("No Disk Selected", "Please select a disk first", mw.window)
		return
	}

	disk := mw.disks[mw.selectedDisk]

	if len(disk.Partitions) == 0 {
		dialog.ShowInformation("No Partitions", "This disk has no partitions", mw.window)
		return
	}

	partNames := make([]string, len(disk.Partitions))
	for i, part := range disk.Partitions {
		partNames[i] = part.Name
	}

	partSelect := widget.NewSelect(partNames, nil)
	fsSelect := widget.NewSelect([]string{"UFS", "FAT32", "ext2", "ext3", "ext4", "NTFS"}, nil)
	fsSelect.SetSelected("UFS")

	infoLabel := widget.NewLabel("Note: ext2/3/4 requires e2fsprogs package\nNTFS requires fusefs-ntfs package")
	infoLabel.Wrapping = fyne.TextWrapWord
	infoLabel.TextStyle = fyne.TextStyle{Italic: true}

	formContent := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Partition", partSelect),
			widget.NewFormItem("Filesystem", fsSelect),
		),
		widget.NewSeparator(),
		infoLabel,
	)

	customDialog := dialog.NewCustomConfirm("Format Partition", "Format", "Cancel", formContent,
		func(ok bool) {
			if !ok {
				return
			}

			if partSelect.Selected == "" {
				dialog.ShowError(fmt.Errorf("please select a partition"), mw.window)
				return
			}

			dialog.ShowConfirm("Confirm Format",
				fmt.Sprintf("Are you sure you want to format %s as %s?\n\nThis will DESTROY all data!", partSelect.Selected, fsSelect.Selected),
				func(confirmed bool) {
					if !confirmed {
						return
					}

					err := partition.FormatPartition(partSelect.Selected, fsSelect.Selected)
					if err != nil {
						dialog.ShowError(err, mw.window)
						return
					}

					dialog.ShowInformation("Success", fmt.Sprintf("Partition formatted successfully as %s", fsSelect.Selected), mw.window)
					mw.refreshDisks()
				}, mw.window)
		}, mw.window)

	customDialog.Resize(fyne.NewSize(450, 250))
	customDialog.Show()
}

func (mw *MainWindow) showResizeDialog() {
	if mw.selectedDisk < 0 {
		dialog.ShowInformation("No Disk Selected", "Please select a disk first", mw.window)
		return
	}

	disk := mw.disks[mw.selectedDisk]

	if len(disk.Partitions) == 0 {
		dialog.ShowInformation("No Partitions", "This disk has no partitions", mw.window)
		return
	}

	partNames := make([]string, len(disk.Partitions))
	for i, part := range disk.Partitions {
		partNames[i] = fmt.Sprintf("%s (%s)", part.Name, partition.FormatBytes(part.Size*512))
	}

	partSelect := widget.NewSelect(partNames, nil)

	dialog.ShowForm("Resize Partition", "Next", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Partition", partSelect),
		},
		func(ok bool) {
			if !ok {
				return
			}

			selectedIdx := -1
			for i, name := range partNames {
				if name == partSelect.Selected {
					selectedIdx = i
					break
				}
			}

			if selectedIdx < 0 {
				return
			}

			resizeDialog := NewResizeDialog(mw.window, &disk, &disk.Partitions[selectedIdx], mw.refreshDisks)
			resizeDialog.Show()
		}, mw.window)
}

func (mw *MainWindow) createColorLegend() *fyne.Container {
	createLegendItem := func(label string, fsType string) *fyne.Container {
		colorBox := canvas.NewRectangle(getPartitionColor(fsType))
		colorBox.SetMinSize(fyne.NewSize(20, 20))
		colorBox.StrokeColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
		colorBox.StrokeWidth = 1

		text := widget.NewLabel(label)
		return container.NewHBox(colorBox, text)
	}

	legendLabel := widget.NewLabel("Color Legend:")
	legendLabel.TextStyle = fyne.TextStyle{Bold: true}

	items := container.NewHBox(
		createLegendItem("UFS", "UFS"),
		createLegendItem("ZFS", "ZFS"),
		createLegendItem("FAT32", "FAT32"),
		createLegendItem("swap", "swap"),
		createLegendItem("ext2/3/4", "ext4"),
		createLegendItem("NTFS", "NTFS"),
		createLegendItem("Unknown", "unknown"),
	)

	return container.NewVBox(
		legendLabel,
		items,
		widget.NewSeparator(),
	)
}

func (mw *MainWindow) showCopyDialog() {
	copyDialog := NewCopyDialog(mw.window, mw.disks, "copy", mw.refreshDisks)
	copyDialog.Show()
}

func (mw *MainWindow) showMoveDialog() {
	moveDialog := NewCopyDialog(mw.window, mw.disks, "move", mw.refreshDisks)
	moveDialog.Show()
}

func (mw *MainWindow) showDiskInfo() {
	if mw.selectedDisk < 0 {
		dialog.ShowInformation("No Disk Selected", "Please select a disk first to view detailed information", mw.window)
		return
	}

	disk := mw.disks[mw.selectedDisk]
	infoDialog := NewDiskInfoDialog(mw.window, disk.Name)
	infoDialog.Show()
}

func (mw *MainWindow) showBatchDialog() {
	batchDialog := NewBatchDialog(mw.window, mw.disks)
	batchDialog.Show()
}

func (mw *MainWindow) performUndo() {
	if !mw.history.CanUndo() {
		dialog.ShowInformation("Cannot Undo", "No reversible operations to undo", mw.window)
		return
	}

	entry, err := mw.history.GetUndoOperation()
	if err != nil {
		dialog.ShowError(err, mw.window)
		return
	}

	// Confirm undo
	entryID := entry.ID
	oldPos := mw.history.GetCurrentPosition()
	dialog.ShowConfirm("Undo Operation",
		fmt.Sprintf("Undo: %s\n\nThis will reverse the operation.", entry.Description),
		func(ok bool) {
			if ok {
				mw.executeUndo(entry)
			} else {
				// Restore the operation state if user cancels
				mw.history.RestoreReversedState(entryID, false)
				mw.history.RestorePosition(oldPos)
			}
		}, mw.window)
}

func (mw *MainWindow) executeUndo(entry *partition.HistoryEntry) {
	var err error

	switch entry.UndoOperation {
	case "delete":
		// Undo create by deleting the partition
		err = partition.DeletePartition(entry.UndoDisk, entry.UndoIndex)

	case "resize":
		// Undo resize by resizing back
		err = partition.ResizePartition(entry.UndoDisk, entry.UndoIndex, entry.UndoSize)

	case "attribute":
		// Undo attribute change by toggling back
		if entry.AttributeSet {
			err = partition.UnsetPartitionAttribute(entry.Partition, entry.AttributeName)
		} else {
			err = partition.SetPartitionAttribute(entry.Partition, entry.AttributeName)
		}

	default:
		err = fmt.Errorf("unknown undo operation: %s", entry.UndoOperation)
	}

	if err != nil {
		dialog.ShowError(fmt.Errorf("undo failed: %v", err), mw.window)
		// Restore the operation state
		mw.history.RestoreReversedState(entry.ID, false)
		mw.history.RestorePosition(mw.history.GetCurrentPosition() + 1)
	} else {
		dialog.ShowInformation("Undo Complete", fmt.Sprintf("Successfully undid: %s", entry.Description), mw.window)
		mw.refreshDisks()
	}
}

func (mw *MainWindow) performRedo() {
	if !mw.history.CanRedo() {
		dialog.ShowInformation("Cannot Redo", "No operations to redo", mw.window)
		return
	}

	entry, err := mw.history.GetRedoOperation()
	if err != nil {
		dialog.ShowError(err, mw.window)
		return
	}

	// Confirm redo
	entryID := entry.ID
	oldPos := mw.history.GetCurrentPosition()
	dialog.ShowConfirm("Redo Operation",
		fmt.Sprintf("Redo: %s\n\nThis will re-apply the operation.", entry.Description),
		func(ok bool) {
			if ok {
				mw.executeRedo(entry)
			} else {
				// Restore the operation state if user cancels
				mw.history.RestoreReversedState(entryID, true)
				mw.history.RestorePosition(oldPos)
			}
		}, mw.window)
}

func (mw *MainWindow) executeRedo(entry *partition.HistoryEntry) {
	var err error

	switch entry.Operation {
	case "create":
		// Redo create
		err = partition.CreatePartition(entry.Disk, entry.Size, entry.FSType)

	case "resize":
		// Redo resize
		err = partition.ResizePartition(entry.Disk, entry.Index, entry.Size)

	case "attribute":
		// Redo attribute change
		if entry.AttributeSet {
			err = partition.SetPartitionAttribute(entry.Partition, entry.AttributeName)
		} else {
			err = partition.UnsetPartitionAttribute(entry.Partition, entry.AttributeName)
		}

	default:
		err = fmt.Errorf("unknown redo operation: %s", entry.Operation)
	}

	if err != nil {
		dialog.ShowError(fmt.Errorf("redo failed: %v", err), mw.window)
		// Restore the operation state
		mw.history.RestoreReversedState(entry.ID, true)
		mw.history.RestorePosition(mw.history.GetCurrentPosition() - 1)
	} else {
		dialog.ShowInformation("Redo Complete", fmt.Sprintf("Successfully redid: %s", entry.Description), mw.window)
		mw.refreshDisks()
	}
}

func (mw *MainWindow) toggleBootableDialog() {
	if mw.selectedDisk < 0 {
		dialog.ShowInformation("No Disk Selected", "Please select a disk first", mw.window)
		return
	}

	disk := mw.disks[mw.selectedDisk]

	if len(disk.Partitions) == 0 {
		dialog.ShowInformation("No Partitions", "This disk has no partitions", mw.window)
		return
	}

	// Validate disk is GPT
	if err := partition.ValidatePartitionForAttributes(disk.Partitions[0].Name); err != nil {
		dialog.ShowError(fmt.Errorf("This disk does not support GPT attributes.\n\nOnly GPT-partitioned disks support bootable flags. This disk appears to be using %s partitioning.", disk.Scheme), mw.window)
		return
	}

	// Create partition selection with current bootable status
	partOptions := make([]string, len(disk.Partitions))
	for i, part := range disk.Partitions {
		bootable, _ := partition.IsBootable(part.Name)
		if bootable {
			partOptions[i] = fmt.Sprintf("%s [BOOTABLE]", part.Name)
		} else {
			partOptions[i] = part.Name
		}
	}

	partSelect := widget.NewSelect(partOptions, nil)
	helpLabel := widget.NewLabel("Toggle the 'bootme' attribute to mark a partition as bootable.\nThis is commonly used for EFI system partitions.")
	helpLabel.Wrapping = fyne.TextWrapWord

	formContent := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Partition", partSelect),
		),
		widget.NewSeparator(),
		helpLabel,
	)

	customDialog := dialog.NewCustomConfirm("Toggle Bootable Flag", "Toggle", "Cancel", formContent,
		func(ok bool) {
			if !ok {
				return
			}

			if partSelect.Selected == "" {
				dialog.ShowError(fmt.Errorf("Please select a partition"), mw.window)
				return
			}

			// Extract partition name (remove [BOOTABLE] suffix if present)
			selectedPartName := partSelect.Selected
			if strings.Contains(selectedPartName, " [BOOTABLE]") {
				selectedPartName = strings.TrimSuffix(selectedPartName, " [BOOTABLE]")
			}

			// Find the selected partition
			var selectedPart *partition.Partition
			for i := range disk.Partitions {
				if disk.Partitions[i].Name == selectedPartName {
					selectedPart = &disk.Partitions[i]
					break
				}
			}

			if selectedPart == nil {
				dialog.ShowError(fmt.Errorf("Partition not found"), mw.window)
				return
			}

			// Check old status before toggling
			wasBootable, _ := partition.IsBootable(selectedPart.Name)

			// Toggle the bootable attribute
			err := partition.TogglePartitionAttribute(selectedPart.Name, partition.AttrBootme)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to toggle bootable flag: %v", err), mw.window)
				return
			}

			// Check new status
			isBootable, _ := partition.IsBootable(selectedPart.Name)

			// Record in history
			mw.history.RecordAttributeChange(selectedPart.Name, partition.AttrBootme, wasBootable, isBootable)

			if isBootable {
				dialog.ShowInformation("Success", fmt.Sprintf("Partition %s is now marked as BOOTABLE", selectedPart.Name), mw.window)
			} else {
				dialog.ShowInformation("Success", fmt.Sprintf("Removed bootable flag from partition %s", selectedPart.Name), mw.window)
			}

			mw.refreshDisks()
		}, mw.window)

	customDialog.Show()
}

func (mw *MainWindow) showAttributesDialog() {
	if mw.selectedDisk < 0 {
		dialog.ShowInformation("No Disk Selected", "Please select a disk first", mw.window)
		return
	}

	disk := mw.disks[mw.selectedDisk]

	if len(disk.Partitions) == 0 {
		dialog.ShowInformation("No Partitions", "This disk has no partitions", mw.window)
		return
	}

	// Create partition selection
	partNames := make([]string, len(disk.Partitions))
	for i, part := range disk.Partitions {
		partNames[i] = part.Name
	}

	partSelect := widget.NewSelect(partNames, nil)

	formContent := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Partition", partSelect),
		),
		widget.NewLabel("Select a partition to edit its GPT attributes"),
	)

	customDialog := dialog.NewCustomConfirm("Edit GPT Attributes", "Edit", "Cancel", formContent,
		func(ok bool) {
			if !ok {
				return
			}

			if partSelect.Selected == "" {
				dialog.ShowError(fmt.Errorf("Please select a partition"), mw.window)
				return
			}

			// Find the selected partition
			var selectedPart *partition.Partition
			for i := range disk.Partitions {
				if disk.Partitions[i].Name == partSelect.Selected {
					selectedPart = &disk.Partitions[i]
					break
				}
			}

			if selectedPart != nil {
				// Show the attributes dialog
				attrDialog := NewAttributesDialog(mw.window, selectedPart, mw.history, mw.refreshDisks)
				attrDialog.Show()
			}
		}, mw.window)

	customDialog.Show()
}

func (mw *MainWindow) Show() {
	mw.window.ShowAndRun()
}
