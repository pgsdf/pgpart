package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/pgsdf/pgpart/internal/partition"
)

type PartitionBlock struct {
	widget.BaseWidget
	partition   *partition.Partition
	disk        *partition.Disk
	rect        *canvas.Rectangle
	label       *canvas.Text
	leftHandle  *ResizeHandle
	rightHandle *ResizeHandle
	width       float32
	onResize    func(part *partition.Partition, newSize uint64)
	partIndex   int
}

type ResizeHandle struct {
	widget.BaseWidget
	rect      *canvas.Rectangle
	dragging  bool
	startX    float32
	onDrag    func(deltaX float32)
	direction string
}

func NewResizeHandle(direction string, onDrag func(deltaX float32)) *ResizeHandle {
	h := &ResizeHandle{
		direction: direction,
		onDrag:    onDrag,
	}
	h.ExtendBaseWidget(h)
	return h
}

func (h *ResizeHandle) CreateRenderer() fyne.WidgetRenderer {
	h.rect = canvas.NewRectangle(color.RGBA{R: 80, G: 80, B: 80, A: 255})
	h.rect.StrokeColor = color.RGBA{R: 200, G: 200, B: 200, A: 255}
	h.rect.StrokeWidth = 2

	return &resizeHandleRenderer{
		handle:  h,
		objects: []fyne.CanvasObject{h.rect},
	}
}

func (h *ResizeHandle) Dragged(e *fyne.DragEvent) {
	if !h.dragging {
		h.dragging = true
		h.startX = e.Position.X
	}
	deltaX := e.Position.X - h.startX
	if h.onDrag != nil {
		h.onDrag(deltaX)
	}
}

func (h *ResizeHandle) DragEnd() {
	h.dragging = false
}

func (h *ResizeHandle) Cursor() desktop.Cursor {
	return desktop.HResizeCursor
}

type resizeHandleRenderer struct {
	handle  *ResizeHandle
	objects []fyne.CanvasObject
}

func (r *resizeHandleRenderer) Layout(size fyne.Size) {
	r.handle.rect.Resize(size)
}

func (r *resizeHandleRenderer) MinSize() fyne.Size {
	return fyne.NewSize(8, 40)
}

func (r *resizeHandleRenderer) Refresh() {
	canvas.Refresh(r.handle.rect)
}

func (r *resizeHandleRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *resizeHandleRenderer) Destroy() {}

type InteractivePartitionView struct {
	widget.BaseWidget
	disk      *partition.Disk
	blocks    []*PartitionBlock
	container *fyne.Container
	window    fyne.Window
	onRefresh func()
}

func NewInteractivePartitionView(disk *partition.Disk, window fyne.Window, onRefresh func()) *InteractivePartitionView {
	view := &InteractivePartitionView{
		disk:      disk,
		window:    window,
		onRefresh: onRefresh,
	}
	view.ExtendBaseWidget(view)
	view.buildBlocks()
	return view
}

func (v *InteractivePartitionView) buildBlocks() {
	v.blocks = []*PartitionBlock{}

	if v.disk == nil || len(v.disk.Partitions) == 0 {
		return
	}

	for i := range v.disk.Partitions {
		block := v.createPartitionBlock(&v.disk.Partitions[i], i)
		v.blocks = append(v.blocks, block)
	}
}

func (v *InteractivePartitionView) createPartitionBlock(part *partition.Partition, index int) *PartitionBlock {
	block := &PartitionBlock{
		partition: part,
		disk:      v.disk,
		partIndex: index,
		onResize:  v.handleResize,
	}

	partColor := getPartitionColor(part.FileSystem)
	block.rect = canvas.NewRectangle(partColor)
	block.rect.StrokeColor = color.RGBA{R: 50, G: 50, B: 50, A: 255}
	block.rect.StrokeWidth = 1

	sizeStr := partition.FormatBytes(part.Size * 512)
	block.label = canvas.NewText(sizeStr, color.White)
	block.label.TextSize = 10
	block.label.Alignment = fyne.TextAlignCenter

	return block
}

func (v *InteractivePartitionView) handleResize(part *partition.Partition, newSize uint64) {
	sizeStr := partition.FormatBytes(newSize * 512)

	dialog.ShowConfirm("Resize Partition",
		fmt.Sprintf("Resize partition %s to %s?\n\nWARNING: This operation may result in data loss!\nMake sure you have backups before proceeding.", part.Name, sizeStr),
		func(confirmed bool) {
			if !confirmed {
				v.onRefresh()
				return
			}

			parts := []string{}
			for _, p := range part.Name {
				if p >= '0' && p <= '9' {
					parts = append(parts, string(p))
				}
			}

			index := ""
			if len(parts) > 0 {
				index = parts[len(parts)-1]
			}

			err := partition.ResizePartition(v.disk.Name, index, newSize*512)
			if err != nil {
				dialog.ShowError(fmt.Errorf("resize failed: %w", err), v.window)
			} else {
				dialog.ShowInformation("Success", "Partition resized successfully", v.window)
			}

			v.onRefresh()
		}, v.window)
}

func (v *InteractivePartitionView) CreateRenderer() fyne.WidgetRenderer {
	v.container = container.NewHBox()

	if len(v.blocks) == 0 {
		emptyRect := canvas.NewRectangle(color.RGBA{R: 200, G: 200, B: 200, A: 255})
		emptyRect.SetMinSize(fyne.NewSize(600, 60))
		v.container.Add(emptyRect)
	} else {
		for _, block := range v.blocks {
			width := float32(600) * float32(block.partition.Size) / float32(v.disk.Size)
			if width < 40 {
				width = 40
			}
			block.width = width

			blockContainer := v.createBlockWithHandles(block, width)
			v.container.Add(blockContainer)
		}
	}

	return widget.NewSimpleRenderer(v.container)
}

func (v *InteractivePartitionView) createBlockWithHandles(block *PartitionBlock, width float32) *fyne.Container {
	block.rect.SetMinSize(fyne.NewSize(width, 60))

	partContainer := container.NewStack(block.rect, container.NewCenter(block.label))

	leftHandle := NewResizeHandle("left", func(deltaX float32) {
		v.handleDrag(block, deltaX, true)
	})

	rightHandle := NewResizeHandle("right", func(deltaX float32) {
		v.handleDrag(block, deltaX, false)
	})

	block.leftHandle = leftHandle
	block.rightHandle = rightHandle

	return container.NewBorder(nil, nil, leftHandle, rightHandle, partContainer)
}

func (v *InteractivePartitionView) handleDrag(block *PartitionBlock, deltaX float32, isLeft bool) {
	pixelsPerSector := float32(600) / float32(v.disk.Size)
	sectorDelta := uint64(deltaX / pixelsPerSector)

	var newSize uint64
	if isLeft {
		if sectorDelta > block.partition.Size {
			newSize = block.partition.Size / 2
		} else {
			newSize = block.partition.Size - sectorDelta
		}
	} else {
		newSize = block.partition.Size + sectorDelta
	}

	minSize := uint64(1024 * 1024 * 10 / 512)
	if newSize < minSize {
		newSize = minSize
	}

	maxSize := v.calculateMaxSize(block)
	if newSize > maxSize {
		newSize = maxSize
	}

	newWidth := float32(600) * float32(newSize) / float32(v.disk.Size)
	if newWidth < 40 {
		newWidth = 40
	}

	block.rect.SetMinSize(fyne.NewSize(newWidth, 60))
	block.label.Text = partition.FormatBytes(newSize * 512)
	block.label.Refresh()

	block.partition.Size = newSize
}

func (v *InteractivePartitionView) calculateMaxSize(block *PartitionBlock) uint64 {
	maxSize := v.disk.Size - block.partition.Start

	for _, p := range v.disk.Partitions {
		if p.Start > block.partition.Start && p.Start < block.partition.Start+maxSize {
			maxSize = p.Start - block.partition.Start
		}
	}

	return maxSize
}
