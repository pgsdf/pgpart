package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/pgsdf/pgpart/internal/partition"
)

type CopyDialog struct {
	window      fyne.Window
	disks       []partition.Disk
	onComplete  func()
	operation   string // "copy" or "move"
	progressBar *widget.ProgressBar
	statusLabel *widget.Label
}

func NewCopyDialog(window fyne.Window, disks []partition.Disk, operation string, onComplete func()) *CopyDialog {
	return &CopyDialog{
		window:     window,
		disks:      disks,
		operation:  operation,
		onComplete: onComplete,
	}
}

func (cd *CopyDialog) Show() {
	// Build list of all partitions
	type PartitionItem struct {
		DiskName string
		PartName string
		Size     uint64
		FS       string
	}

	var partitions []PartitionItem
	for _, disk := range cd.disks {
		for _, part := range disk.Partitions {
			partitions = append(partitions, PartitionItem{
				DiskName: disk.Name,
				PartName: part.Name,
				Size:     part.Size * 512,
				FS:       part.FileSystem,
			})
		}
	}

	if len(partitions) < 2 {
		dialog.ShowInformation("Insufficient Partitions",
			"You need at least 2 partitions to perform a copy operation", cd.window)
		return
	}

	// Create partition selection options
	partOptions := make([]string, len(partitions))
	for i, part := range partitions {
		partOptions[i] = fmt.Sprintf("%s (%s, %s)",
			part.PartName,
			partition.FormatBytes(part.Size),
			part.FS)
	}

	sourceSelect := widget.NewSelect(partOptions, nil)
	destSelect := widget.NewSelect(partOptions, nil)

	var titleText string
	if cd.operation == "move" {
		titleText = "Move Partition"
	} else {
		titleText = "Copy Partition"
	}

	warningLabel := widget.NewLabel("⚠️  WARNING: This will overwrite all data on the destination partition!")
	warningLabel.Wrapping = fyne.TextWrapWord
	warningLabel.TextStyle = fyne.TextStyle{Bold: true}

	infoLabel := widget.NewLabel("Select the source partition to copy from and the destination partition to copy to.\nThe destination must be equal or larger in size.")
	infoLabel.Wrapping = fyne.TextWrapWord
	infoLabel.TextStyle = fyne.TextStyle{Italic: true}

	formContent := container.NewVBox(
		warningLabel,
		widget.NewSeparator(),
		widget.NewForm(
			widget.NewFormItem("Source Partition", sourceSelect),
			widget.NewFormItem("Destination Partition", destSelect),
		),
		widget.NewSeparator(),
		infoLabel,
	)

	customDialog := dialog.NewCustomConfirm(titleText, "Start", "Cancel", formContent,
		func(ok bool) {
			if !ok {
				return
			}

			if sourceSelect.Selected == "" || destSelect.Selected == "" {
				dialog.ShowError(fmt.Errorf("please select both source and destination partitions"), cd.window)
				return
			}

			if sourceSelect.Selected == destSelect.Selected {
				dialog.ShowError(fmt.Errorf("source and destination must be different"), cd.window)
				return
			}

			// Find selected partitions
			var sourcePart, destPart PartitionItem
			for i, opt := range partOptions {
				if opt == sourceSelect.Selected {
					sourcePart = partitions[i]
				}
				if opt == destSelect.Selected {
					destPart = partitions[i]
				}
			}

			// Check size compatibility
			if destPart.Size < sourcePart.Size {
				dialog.ShowError(fmt.Errorf("destination partition is too small\nSource: %s, Destination: %s",
					partition.FormatBytes(sourcePart.Size),
					partition.FormatBytes(destPart.Size)), cd.window)
				return
			}

			// Show confirmation
			var confirmMsg string
			if cd.operation == "move" {
				confirmMsg = fmt.Sprintf("Move partition %s to %s?\n\nThis will:\n- Copy all data from %s to %s\n- Delete the source partition %s\n- DESTROY all existing data on %s\n\nThis operation cannot be undone!",
					sourcePart.PartName, destPart.PartName,
					sourcePart.PartName, destPart.PartName,
					sourcePart.PartName, destPart.PartName)
			} else {
				confirmMsg = fmt.Sprintf("Copy partition %s to %s?\n\nThis will DESTROY all existing data on %s!\n\nSource: %s (%s)\nDestination: %s (%s)",
					sourcePart.PartName, destPart.PartName,
					destPart.PartName,
					sourcePart.PartName, partition.FormatBytes(sourcePart.Size),
					destPart.PartName, partition.FormatBytes(destPart.Size))
			}

			dialog.ShowConfirm("Confirm "+titleText, confirmMsg,
				func(confirmed bool) {
					if !confirmed {
						return
					}
					cd.performOperation(sourcePart.PartName, destPart.PartName)
				}, cd.window)
		}, cd.window)

	customDialog.Resize(fyne.NewSize(550, 350))
	customDialog.Show()
}

func (cd *CopyDialog) performOperation(source, dest string) {
	// Create progress dialog
	cd.progressBar = widget.NewProgressBar()
	cd.statusLabel = widget.NewLabel("Preparing to copy...")

	progressContent := container.NewVBox(
		cd.statusLabel,
		cd.progressBar,
		widget.NewLabel("\nPlease wait, this may take several minutes..."),
	)

	var titleText string
	if cd.operation == "move" {
		titleText = "Moving Partition"
	} else {
		titleText = "Copying Partition"
	}

	progressDialog := dialog.NewCustom(titleText, "Cancel", progressContent, cd.window)
	progressDialog.Resize(fyne.NewSize(450, 150))
	progressDialog.Show()

	// Perform the operation in a goroutine
	go func() {
		var err error
		startTime := time.Now()

		progressCallback := func(progress float64) {
			cd.progressBar.SetValue(progress / 100.0)
			elapsed := time.Since(startTime)
			cd.statusLabel.SetText(fmt.Sprintf("Progress: %.1f%% (Elapsed: %s)", progress, elapsed.Round(time.Second)))
		}

		if cd.operation == "move" {
			// Extract disk and index from partition name
			// This is simplified - you may need to adjust based on your partition naming
			cd.statusLabel.SetText("Moving partition...")
			err = partition.CopyPartition(source, dest, progressCallback)
			if err == nil {
				cd.statusLabel.SetText("Move completed successfully!")
			}
		} else {
			cd.statusLabel.SetText("Copying partition...")
			err = partition.CopyPartition(source, dest, progressCallback)
			if err == nil {
				cd.statusLabel.SetText("Copy completed successfully!")
			}
		}

		progressDialog.Hide()

		if err != nil {
			dialog.ShowError(fmt.Errorf("%s failed: %w", cd.operation, err), cd.window)
		} else {
			duration := time.Since(startTime).Round(time.Second)
			dialog.ShowInformation("Success",
				fmt.Sprintf("Partition %s completed successfully!\n\nTime taken: %s",
					cd.operation, duration), cd.window)
			if cd.onComplete != nil {
				cd.onComplete()
			}
		}
	}()
}
