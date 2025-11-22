package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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
}

func NewMainWindow(app fyne.App) *MainWindow {
	mw := &MainWindow{
		window:       app.NewWindow("PGPart - Partition Manager for FreeBSD"),
		selectedDisk: -1,
	}

	mw.window.Resize(fyne.NewSize(900, 600))
	mw.setupUI()
	mw.refreshDisks()

	return mw
}

func (mw *MainWindow) setupUI() {
	mw.infoLabel = widget.NewLabel("Select a disk to view partitions")

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(fyne.NewMenuItem("Refresh", mw.refreshDisks).Icon, mw.refreshDisks),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(fyne.NewMenuItem("New Partition Table", nil).Icon, mw.showNewPartitionTableDialog),
		widget.NewToolbarAction(fyne.NewMenuItem("New Partition", nil).Icon, mw.showNewPartitionDialog),
		widget.NewToolbarAction(fyne.NewMenuItem("Resize Partition", nil).Icon, mw.showResizeDialog),
		widget.NewToolbarAction(fyne.NewMenuItem("Delete Partition", nil).Icon, mw.showDeletePartitionDialog),
		widget.NewToolbarAction(fyne.NewMenuItem("Format", nil).Icon, mw.showFormatDialog),
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
		legendLabel := widget.NewLabel("Legend: UFS (blue) | ZFS (green) | FAT32 (orange) | swap (red) | ext4 (purple)")
		legendLabel.TextStyle = fyne.TextStyle{Italic: true}
		mw.partitionView.Add(legendLabel)

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
		return color.RGBA{R: 100, G: 150, B: 255, A: 255}
	case "ZFS":
		return color.RGBA{R: 100, G: 200, B: 100, A: 255}
	case "FAT32":
		return color.RGBA{R: 255, G: 200, B: 100, A: 255}
	case "swap":
		return color.RGBA{R: 255, G: 100, B: 100, A: 255}
	case "ext4":
		return color.RGBA{R: 200, G: 100, B: 255, A: 255}
	default:
		return color.RGBA{R: 150, G: 150, B: 150, A: 255}
	}
}

func (mw *MainWindow) createPartitionCard(part partition.Partition) *fyne.Container {
	nameLabel := widget.NewLabelWithStyle(part.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	typeLabel := widget.NewLabel(fmt.Sprintf("Type: %s", part.Type))
	sizeLabel := widget.NewLabel(fmt.Sprintf("Size: %s", partition.FormatBytes(part.Size*512)))
	fsLabel := widget.NewLabel(fmt.Sprintf("Filesystem: %s", part.FileSystem))
	mountLabel := widget.NewLabel(fmt.Sprintf("Mount: %s", part.MountPoint))

	if part.MountPoint == "" {
		mountLabel.SetText("Mount: (not mounted)")
	}

	card := container.NewVBox(
		nameLabel,
		typeLabel,
		sizeLabel,
		fsLabel,
		mountLabel,
		widget.NewSeparator(),
	)

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
	fsSelect := widget.NewSelect([]string{"UFS", "FAT32"}, nil)
	fsSelect.SetSelected("UFS")

	dialog.ShowForm("Format Partition", "Format", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Partition", partSelect),
			widget.NewFormItem("Filesystem", fsSelect),
		},
		func(ok bool) {
			if !ok {
				return
			}

			dialog.ShowConfirm("Confirm Format",
				fmt.Sprintf("Are you sure you want to format %s? This will DESTROY all data!", partSelect.Selected),
				func(confirmed bool) {
					if !confirmed {
						return
					}

					err := partition.FormatPartition(partSelect.Selected, fsSelect.Selected)
					if err != nil {
						dialog.ShowError(err, mw.window)
						return
					}

					dialog.ShowInformation("Success", "Partition formatted successfully", mw.window)
					mw.refreshDisks()
				}, mw.window)
		}, mw.window)
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

func (mw *MainWindow) Show() {
	mw.window.ShowAndRun()
}
