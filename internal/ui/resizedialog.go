package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/pgsdf/pgpart/internal/partition"
)

type ResizeDialog struct {
	window    fyne.Window
	disk      *partition.Disk
	partition *partition.Partition
	onResize  func()
}

func NewResizeDialog(window fyne.Window, disk *partition.Disk, part *partition.Partition, onResize func()) *ResizeDialog {
	return &ResizeDialog{
		window:    window,
		disk:      disk,
		partition: part,
		onResize:  onResize,
	}
}

func (rd *ResizeDialog) Show() {
	currentSizeMB := rd.partition.Size * 512 / (1024 * 1024)
	currentSizeStr := partition.FormatBytes(rd.partition.Size * 512)

	maxSize := rd.calculateMaxSize()
	maxSizeMB := maxSize * 512 / (1024 * 1024)
	minSizeMB := uint64(10)

	currentLabel := widget.NewLabel(fmt.Sprintf("Current Size: %s (%d MB)", currentSizeStr, currentSizeMB))
	currentLabel.Wrapping = fyne.TextWrapWord

	sizeEntry := widget.NewEntry()
	sizeEntry.SetText(fmt.Sprintf("%d", currentSizeMB))
	sizeEntry.SetPlaceHolder(fmt.Sprintf("Size in MB (min: %d, max: %d)", minSizeMB, maxSizeMB))

	slider := widget.NewSlider(float64(minSizeMB), float64(maxSizeMB))
	slider.Value = float64(currentSizeMB)
	slider.Step = 100

	previewLabel := widget.NewLabel("")
	previewLabel.Wrapping = fyne.TextWrapWord

	updatePreview := func(sizeMB uint64) {
		newSizeStr := partition.FormatBytes(sizeMB * 1024 * 1024)
		diff := int64(sizeMB) - int64(currentSizeMB)
		var diffStr string
		if diff > 0 {
			diffStr = fmt.Sprintf("+%d MB", diff)
		} else if diff < 0 {
			diffStr = fmt.Sprintf("%d MB", diff)
		} else {
			diffStr = "No change"
		}
		previewLabel.SetText(fmt.Sprintf("New Size: %s (%s)", newSizeStr, diffStr))
	}

	slider.OnChanged = func(value float64) {
		sizeMB := uint64(value)
		sizeEntry.SetText(fmt.Sprintf("%d", sizeMB))
		updatePreview(sizeMB)
	}

	sizeEntry.OnChanged = func(value string) {
		sizeMB, err := strconv.ParseUint(value, 10, 64)
		if err == nil && sizeMB >= minSizeMB && sizeMB <= maxSizeMB {
			slider.SetValue(float64(sizeMB))
			updatePreview(sizeMB)
		}
	}

	updatePreview(currentSizeMB)

	infoLabel := widget.NewLabel(fmt.Sprintf(
		"Partition: %s\nType: %s\nFilesystem: %s\nMin: %d MB, Max: %d MB",
		rd.partition.Name,
		rd.partition.Type,
		rd.partition.FileSystem,
		minSizeMB,
		maxSizeMB,
	))
	infoLabel.Wrapping = fyne.TextWrapWord

	// Check online resize capability
	isGrow := true // Will be updated based on actual size change
	onlineResizeCheck := widget.NewCheck("Online Resize (keep filesystem mounted)", nil)
	onlineResizeCheck.Disable()

	onlineResizeInfo := widget.NewLabel("")
	onlineResizeInfo.Wrapping = fyne.TextWrapWord

	// Update online resize info based on size change
	updateOnlineResizeInfo := func(newSizeMB uint64) {
		isGrow = newSizeMB > currentSizeMB
		canOnline, reason := partition.CanResizeOnline(rd.partition, isGrow)

		if canOnline {
			onlineResizeCheck.Enable()
			onlineResizeCheck.Checked = true
			recommendation := partition.GetOnlineResizeRecommendation(rd.partition, isGrow)
			onlineResizeInfo.SetText(recommendation)
		} else {
			onlineResizeCheck.Disable()
			onlineResizeCheck.Checked = false
			onlineResizeInfo.SetText(reason)
		}
		onlineResizeCheck.Refresh()
		onlineResizeInfo.Refresh()
	}

	// Update online resize info on slider/entry change
	originalSliderOnChanged := slider.OnChanged
	slider.OnChanged = func(value float64) {
		originalSliderOnChanged(value)
		updateOnlineResizeInfo(uint64(value))
	}

	originalEntryOnChanged := sizeEntry.OnChanged
	sizeEntry.OnChanged = func(value string) {
		originalEntryOnChanged(value)
		sizeMB, err := strconv.ParseUint(value, 10, 64)
		if err == nil {
			updateOnlineResizeInfo(sizeMB)
		}
	}

	// Initial update
	updateOnlineResizeInfo(currentSizeMB)

	warningLabel := widget.NewLabel("⚠️  WARNING: Resizing partitions can cause data loss!\nMake sure you have backups before proceeding.")
	warningLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		infoLabel,
		widget.NewSeparator(),
		currentLabel,
		widget.NewForm(
			widget.NewFormItem("New Size (MB)", sizeEntry),
		),
		slider,
		previewLabel,
		widget.NewSeparator(),
		onlineResizeCheck,
		onlineResizeInfo,
		widget.NewSeparator(),
		warningLabel,
	)

	d := dialog.NewCustomConfirm("Resize Partition", "Resize", "Cancel", content,
		func(confirmed bool) {
			if !confirmed {
				return
			}

			sizeMB, err := strconv.ParseUint(sizeEntry.Text, 10, 64)
			if err != nil {
				dialog.ShowError(fmt.Errorf("invalid size: %w", err), rd.window)
				return
			}

			if sizeMB < minSizeMB || sizeMB > maxSizeMB {
				dialog.ShowError(fmt.Errorf("size must be between %d MB and %d MB", minSizeMB, maxSizeMB), rd.window)
				return
			}

			if sizeMB == currentSizeMB {
				dialog.ShowInformation("No Changes", "Partition size unchanged", rd.window)
				return
			}

			useOnlineResize := onlineResizeCheck.Checked && !onlineResizeCheck.Disabled()
			rd.performResize(sizeMB*1024*1024, useOnlineResize)
		}, rd.window)

	d.Resize(fyne.NewSize(500, 400))
	d.Show()
}

func (rd *ResizeDialog) calculateMaxSize() uint64 {
	maxSize := rd.disk.Size - rd.partition.Start

	for _, p := range rd.disk.Partitions {
		if p.Start > rd.partition.Start && p.Start < rd.partition.Start+maxSize {
			maxSize = p.Start - rd.partition.Start
		}
	}

	return maxSize
}

func (rd *ResizeDialog) performResize(newSizeBytes uint64, useOnlineResize bool) {
	parts := strings.Split(rd.partition.Name, "p")
	if len(parts) < 2 {
		dialog.ShowError(fmt.Errorf("invalid partition name format"), rd.window)
		return
	}

	index := parts[len(parts)-1]

	var err error
	if useOnlineResize {
		// Perform online resize (partition + filesystem together)
		err = partition.PerformOnlineResize(rd.disk.Name, index, newSizeBytes, rd.partition)
		if err != nil {
			dialog.ShowError(fmt.Errorf("online resize failed: %w", err), rd.window)
			return
		}
		dialog.ShowInformation("Success", "Partition and filesystem resized online successfully!\nThe filesystem remained mounted during the operation.", rd.window)
	} else {
		// Perform offline resize (partition only)
		err = partition.ResizePartition(rd.disk.Name, index, newSizeBytes)
		if err != nil {
			dialog.ShowError(fmt.Errorf("resize failed: %w", err), rd.window)
			return
		}
		dialog.ShowInformation("Success", "Partition resized successfully.\nYou may need to resize the filesystem separately if it exists.", rd.window)
	}

	if rd.onResize != nil {
		rd.onResize()
	}
}
