package ui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/pgsdf/pgpart/internal/partition"
)

// BatchDialog manages the batch operations dialog
type BatchDialog struct {
	window        fyne.Window
	disks         []partition.Disk
	queue         *partition.BatchQueue
	operationList *widget.List
	statusLabel   *widget.Label
	progressBar   *widget.ProgressBar
	executeBtn    *widget.Button
	stopOnError   *widget.Check
	selectedOp    int
}

// NewBatchDialog creates a new batch operations dialog
func NewBatchDialog(window fyne.Window, disks []partition.Disk) *BatchDialog {
	return &BatchDialog{
		window:     window,
		disks:      disks,
		queue:      partition.NewBatchQueue(),
		selectedOp: -1,
	}
}

// Show displays the batch operations dialog
func (bd *BatchDialog) Show() {
	// Status label
	bd.statusLabel = widget.NewLabel("No operations queued")

	// Progress bar
	bd.progressBar = widget.NewProgressBar()
	bd.progressBar.Hide()

	// Operation list
	bd.operationList = widget.NewList(
		func() int {
			return bd.queue.Count()
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			ops := bd.queue.GetOperations()
			if id < len(ops) {
				op := ops[id]
				status := ""
				switch op.Status {
				case "pending":
					status = "⏸ "
				case "running":
					status = "▶ "
				case "completed":
					status = "✓ "
				case "failed":
					status = "✗ "
				}
				label.SetText(fmt.Sprintf("%s%d. %s - %s", status, op.ID, op.Type, op.Description))
			}
		},
	)

	bd.operationList.OnSelected = func(id widget.ListItemID) {
		bd.selectedOp = id
	}

	// Stop on error checkbox
	bd.stopOnError = widget.NewCheck("Stop on error", nil)
	bd.stopOnError.SetChecked(true)

	// Add operation buttons
	addFormatBtn := widget.NewButton("Add Format", bd.showAddFormatDialog)
	addDeleteBtn := widget.NewButton("Add Delete", bd.showAddDeleteDialog)
	addResizeBtn := widget.NewButton("Add Resize", bd.showAddResizeDialog)
	addCopyBtn := widget.NewButton("Add Copy", bd.showAddCopyDialog)

	addButtons := container.NewGridWithColumns(2,
		addFormatBtn,
		addDeleteBtn,
		addResizeBtn,
		addCopyBtn,
	)

	// Control buttons
	removeBtn := widget.NewButton("Remove Selected", func() {
		if bd.selectedOp >= 0 {
			ops := bd.queue.GetOperations()
			if bd.selectedOp < len(ops) {
				bd.queue.RemoveOperation(ops[bd.selectedOp].ID)
				bd.selectedOp = -1
				bd.updateStatus()
				bd.operationList.Refresh()
			}
		}
	})

	moveUpBtn := widget.NewButton("Move Up", func() {
		if bd.selectedOp > 0 {
			ops := bd.queue.GetOperations()
			bd.queue.MoveOperation(ops[bd.selectedOp].ID, bd.selectedOp-1)
			bd.selectedOp--
			bd.operationList.Refresh()
		}
	})

	moveDownBtn := widget.NewButton("Move Down", func() {
		if bd.selectedOp >= 0 && bd.selectedOp < bd.queue.Count()-1 {
			ops := bd.queue.GetOperations()
			bd.queue.MoveOperation(ops[bd.selectedOp].ID, bd.selectedOp+1)
			bd.selectedOp++
			bd.operationList.Refresh()
		}
	})

	clearBtn := widget.NewButton("Clear All", func() {
		bd.queue.Clear()
		bd.selectedOp = -1
		bd.updateStatus()
		bd.operationList.Refresh()
	})

	controlButtons := container.NewGridWithColumns(2,
		removeBtn,
		clearBtn,
		moveUpBtn,
		moveDownBtn,
	)

	// Execute button
	bd.executeBtn = widget.NewButton("Execute All", bd.executeAll)

	// Close button
	closeBtn := widget.NewButton("Close", func() {
		// Dialog will be closed by the caller
	})

	// Layout
	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Batch Operations Queue"),
			widget.NewSeparator(),
			bd.statusLabel,
			bd.progressBar,
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			widget.NewLabel("Add Operations:"),
			addButtons,
			widget.NewSeparator(),
			widget.NewLabel("Manage Queue:"),
			controlButtons,
			widget.NewSeparator(),
			bd.stopOnError,
			container.NewGridWithColumns(2, bd.executeBtn, closeBtn),
		),
		nil,
		nil,
		bd.operationList,
	)

	// Create and show dialog
	d := dialog.NewCustom("Batch Operations", "Close", content, bd.window)
	d.Resize(fyne.NewSize(600, 500))
	d.Show()
}

// showAddFormatDialog shows dialog to add a format operation
func (bd *BatchDialog) showAddFormatDialog() {
	// Get all partitions
	partitions := bd.getAllPartitions()
	if len(partitions) == 0 {
		dialog.ShowInformation("No Partitions", "No partitions available", bd.window)
		return
	}

	// Partition selector
	partSelect := widget.NewSelect(partitions, nil)
	if len(partitions) > 0 {
		partSelect.SetSelected(partitions[0])
	}

	// Filesystem type selector
	fsTypes := []string{"UFS", "FAT32", "ext2", "ext3", "ext4", "NTFS"}
	fsSelect := widget.NewSelect(fsTypes, nil)
	fsSelect.SetSelected("UFS")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Partition", Widget: partSelect},
			{Text: "Filesystem", Widget: fsSelect},
		},
	}

	dialog.ShowForm("Add Format Operation", "Add", "Cancel", form.Items, func(ok bool) {
		if ok && partSelect.Selected != "" && fsSelect.Selected != "" {
			op := &partition.BatchOperation{
				Type:           partition.OpFormat,
				Partition:      partSelect.Selected,
				FilesystemType: fsSelect.Selected,
				Description:    fmt.Sprintf("Format %s as %s", partSelect.Selected, fsSelect.Selected),
			}
			bd.queue.AddOperation(op)
			bd.updateStatus()
			bd.operationList.Refresh()
		}
	}, bd.window)
}

// showAddDeleteDialog shows dialog to add a delete operation
func (bd *BatchDialog) showAddDeleteDialog() {
	partitions := bd.getAllPartitions()
	if len(partitions) == 0 {
		dialog.ShowInformation("No Partitions", "No partitions available", bd.window)
		return
	}

	partSelect := widget.NewSelect(partitions, nil)
	if len(partitions) > 0 {
		partSelect.SetSelected(partitions[0])
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Partition", Widget: partSelect},
		},
	}

	dialog.ShowForm("Add Delete Operation", "Add", "Cancel", form.Items, func(ok bool) {
		if ok && partSelect.Selected != "" {
			op := &partition.BatchOperation{
				Type:        partition.OpDelete,
				Partition:   partSelect.Selected,
				Description: fmt.Sprintf("Delete partition %s", partSelect.Selected),
			}
			bd.queue.AddOperation(op)
			bd.updateStatus()
			bd.operationList.Refresh()
		}
	}, bd.window)
}

// showAddResizeDialog shows dialog to add a resize operation
func (bd *BatchDialog) showAddResizeDialog() {
	partitions := bd.getAllPartitions()
	if len(partitions) == 0 {
		dialog.ShowInformation("No Partitions", "No partitions available", bd.window)
		return
	}

	partSelect := widget.NewSelect(partitions, nil)
	if len(partitions) > 0 {
		partSelect.SetSelected(partitions[0])
	}

	sizeEntry := widget.NewEntry()
	sizeEntry.SetPlaceHolder("Size in GB")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Partition", Widget: partSelect},
			{Text: "New Size (GB)", Widget: sizeEntry},
		},
	}

	dialog.ShowForm("Add Resize Operation", "Add", "Cancel", form.Items, func(ok bool) {
		if ok && partSelect.Selected != "" && sizeEntry.Text != "" {
			sizeGB, err := strconv.ParseFloat(sizeEntry.Text, 64)
			if err != nil || sizeGB <= 0 {
				dialog.ShowError(fmt.Errorf("invalid size"), bd.window)
				return
			}
			sizeBytes := uint64(sizeGB * 1024 * 1024 * 1024)
			op := &partition.BatchOperation{
				Type:        partition.OpResize,
				Partition:   partSelect.Selected,
				Size:        sizeBytes,
				Description: fmt.Sprintf("Resize %s to %.2f GB", partSelect.Selected, sizeGB),
			}
			bd.queue.AddOperation(op)
			bd.updateStatus()
			bd.operationList.Refresh()
		}
	}, bd.window)
}

// showAddCopyDialog shows dialog to add a copy operation
func (bd *BatchDialog) showAddCopyDialog() {
	partitions := bd.getAllPartitions()
	if len(partitions) < 2 {
		dialog.ShowInformation("Insufficient Partitions", "Need at least 2 partitions for copy operation", bd.window)
		return
	}

	sourceSelect := widget.NewSelect(partitions, nil)
	destSelect := widget.NewSelect(partitions, nil)
	if len(partitions) > 0 {
		sourceSelect.SetSelected(partitions[0])
		if len(partitions) > 1 {
			destSelect.SetSelected(partitions[1])
		}
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Source Partition", Widget: sourceSelect},
			{Text: "Destination Partition", Widget: destSelect},
		},
	}

	dialog.ShowForm("Add Copy Operation", "Add", "Cancel", form.Items, func(ok bool) {
		if ok && sourceSelect.Selected != "" && destSelect.Selected != "" {
			if sourceSelect.Selected == destSelect.Selected {
				dialog.ShowError(fmt.Errorf("source and destination cannot be the same"), bd.window)
				return
			}
			op := &partition.BatchOperation{
				Type:        partition.OpCopy,
				SourcePart:  sourceSelect.Selected,
				DestPart:    destSelect.Selected,
				Description: fmt.Sprintf("Copy %s to %s", sourceSelect.Selected, destSelect.Selected),
			}
			bd.queue.AddOperation(op)
			bd.updateStatus()
			bd.operationList.Refresh()
		}
	}, bd.window)
}

// getAllPartitions returns a list of all partitions from all disks
func (bd *BatchDialog) getAllPartitions() []string {
	var partitions []string
	for _, disk := range bd.disks {
		for _, part := range disk.Partitions {
			partitions = append(partitions, part.Name)
		}
	}
	return partitions
}

// updateStatus updates the status label
func (bd *BatchDialog) updateStatus() {
	count := bd.queue.Count()
	completed := bd.queue.GetCompletedCount()
	failed := bd.queue.GetFailedCount()

	if count == 0 {
		bd.statusLabel.SetText("No operations queued")
	} else {
		bd.statusLabel.SetText(fmt.Sprintf("Total: %d | Completed: %d | Failed: %d | Pending: %d",
			count, completed, failed, count-completed-failed))
	}
}

// executeAll executes all operations in the queue
func (bd *BatchDialog) executeAll() {
	if !bd.queue.HasPendingOperations() {
		dialog.ShowInformation("No Operations", "No pending operations to execute", bd.window)
		return
	}

	// Confirm execution
	dialog.ShowConfirm("Execute Batch Operations",
		fmt.Sprintf("Execute %d operations?\n\nThis will modify your disk partitions!", bd.queue.Count()),
		func(ok bool) {
			if ok {
				bd.performExecution()
			}
		}, bd.window)
}

// performExecution executes the batch operations
func (bd *BatchDialog) performExecution() {
	bd.executeBtn.Disable()
	bd.progressBar.Show()
	bd.progressBar.SetValue(0)

	go func() {
		err := bd.queue.ExecuteAll(bd.stopOnError.Checked, func(current, total int, desc string) {
			bd.statusLabel.SetText(fmt.Sprintf("Executing %d/%d: %s", current, total, desc))
			bd.progressBar.SetValue(float64(current) / float64(total))
			bd.operationList.Refresh()
		})

		// Update UI on main thread
		bd.progressBar.SetValue(1.0)
		bd.executeBtn.Enable()
		bd.updateStatus()
		bd.operationList.Refresh()

		if err != nil {
			dialog.ShowError(err, bd.window)
		} else {
			completed := bd.queue.GetCompletedCount()
			failed := bd.queue.GetFailedCount()
			msg := fmt.Sprintf("Batch execution complete!\n\nCompleted: %d\nFailed: %d", completed, failed)
			dialog.ShowInformation("Execution Complete", msg, bd.window)
		}
	}()
}
