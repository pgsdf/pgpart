package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/pgsdf/pgpart/internal/partition"
)

type DiskInfoDialog struct {
	window   fyne.Window
	diskName string
}

func NewDiskInfoDialog(window fyne.Window, diskName string) *DiskInfoDialog {
	return &DiskInfoDialog{
		window:   window,
		diskName: diskName,
	}
}

func (d *DiskInfoDialog) Show() {
	// Show loading dialog
	progressBar := widget.NewProgressBarInfinite()
	loadingContent := container.NewVBox(
		widget.NewLabel("Loading disk information..."),
		progressBar,
	)
	loadingDialog := dialog.NewCustom("Please Wait", "Cancel", loadingContent, d.window)
	loadingDialog.Show()

	// Fetch disk info in goroutine
	go func() {
		info, err := partition.GetDetailedDiskInfo(d.diskName)
		loadingDialog.Hide()

		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to get disk information: %w", err), d.window)
			return
		}

		d.showDiskInfo(info)
	}()
}

func (d *DiskInfoDialog) showDiskInfo(info *partition.DiskInfo) {
	// Create tabbed interface
	tabs := container.NewAppTabs()

	// General tab
	generalTab := d.createGeneralTab(info)
	tabs.Append(container.NewTabItem("General", generalTab))

	// SMART tab (if available)
	if info.SMARTEnabled {
		smartTab := d.createSMARTTab(info)
		tabs.Append(container.NewTabItem("SMART Status", smartTab))

		attributesTab := d.createAttributesTab(info)
		tabs.Append(container.NewTabItem("SMART Attributes", attributesTab))
	} else {
		noSmartTab := container.NewVBox(
			widget.NewLabel("SMART monitoring is not available for this disk."),
			widget.NewSeparator(),
			widget.NewLabel("To enable SMART monitoring, install smartmontools:"),
			widget.NewLabel("  pkg install smartmontools"),
		)
		tabs.Append(container.NewTabItem("SMART Status", noSmartTab))
	}

	// Capabilities tab
	capsTab := d.createCapabilitiesTab(info)
	tabs.Append(container.NewTabItem("Capabilities", capsTab))

	// Create dialog
	customDialog := dialog.NewCustom("Disk Information - "+info.Device, "Close", tabs, d.window)
	customDialog.Resize(fyne.NewSize(700, 500))
	customDialog.Show()
}

func (d *DiskInfoDialog) createGeneralTab(info *partition.DiskInfo) *fyne.Container {
	// Create info grid
	form := widget.NewForm()

	form.Append("Device", widget.NewLabel(info.Device))
	form.Append("Model", widget.NewLabel(info.Model))
	form.Append("Serial Number", widget.NewLabel(info.Serial))
	form.Append("Capacity", widget.NewLabel(partition.FormatBytes(info.Size)))
	form.Append("Sector Size", widget.NewLabel(fmt.Sprintf("%d bytes", info.SectorSize)))

	if info.Scheme != "" {
		form.Append("Partition Scheme", widget.NewLabel(info.Scheme))
	} else {
		form.Append("Partition Scheme", widget.NewLabel("None (unformatted)"))
	}

	if info.Temperature > 0 {
		tempLabel := widget.NewLabel(fmt.Sprintf("%d°C", info.Temperature))
		if info.Temperature > 60 {
			tempLabel.TextStyle = fyne.TextStyle{Bold: true}
		}
		form.Append("Temperature", tempLabel)
	}

	if info.PowerOnHours > 0 {
		days := info.PowerOnHours / 24
		hours := info.PowerOnHours % 24
		form.Append("Power On Time", widget.NewLabel(fmt.Sprintf("%d hours (%d days, %d hours)", info.PowerOnHours, days, hours)))
	}

	if info.PowerCycles > 0 {
		form.Append("Power Cycle Count", widget.NewLabel(fmt.Sprintf("%d", info.PowerCycles)))
	}

	return container.NewVBox(
		form,
	)
}

func (d *DiskInfoDialog) createSMARTTab(info *partition.DiskInfo) *fyne.Container {
	// SMART status indicator
	var statusLabel *widget.Label
	var statusColor color.Color

	switch info.SMARTStatus {
	case "PASSED":
		statusLabel = widget.NewLabel("✓ PASSED - Disk is healthy")
		statusLabel.TextStyle = fyne.TextStyle{Bold: true}
		statusColor = color.RGBA{R: 50, G: 205, B: 50, A: 255} // Green
	case "FAILED":
		statusLabel = widget.NewLabel("✗ FAILED - Disk may be failing!")
		statusLabel.TextStyle = fyne.TextStyle{Bold: true}
		statusColor = color.RGBA{R: 220, G: 20, B: 60, A: 255} // Red
	default:
		statusLabel = widget.NewLabel("? UNKNOWN - Status unclear")
		statusLabel.TextStyle = fyne.TextStyle{Italic: true}
		statusColor = color.RGBA{R: 255, G: 165, B: 0, A: 255} // Orange
	}

	statusRect := canvas.NewRectangle(statusColor)
	statusRect.SetMinSize(fyne.NewSize(20, 20))

	statusBox := container.NewHBox(statusRect, statusLabel)

	// Summary information
	summaryForm := widget.NewForm()

	if info.Temperature > 0 {
		tempStr := fmt.Sprintf("%d°C", info.Temperature)
		if info.Temperature > 60 {
			tempStr += " ⚠️ HIGH"
		}
		summaryForm.Append("Current Temperature", widget.NewLabel(tempStr))
	}

	if info.PowerOnHours > 0 {
		summaryForm.Append("Power On Hours", widget.NewLabel(fmt.Sprintf("%d hours", info.PowerOnHours)))
	}

	if info.PowerCycles > 0 {
		summaryForm.Append("Power Cycle Count", widget.NewLabel(fmt.Sprintf("%d cycles", info.PowerCycles)))
	}

	// Warning about critical attributes
	criticalCount := 0
	warningCount := 0
	for _, attr := range info.Attributes {
		if attr.Status == "FAILING" {
			criticalCount++
		} else if attr.Status == "WARNING" {
			warningCount++
		}
	}

	if criticalCount > 0 || warningCount > 0 {
		warningText := fmt.Sprintf("⚠️ %d critical, %d warning attributes", criticalCount, warningCount)
		warningLabel := widget.NewLabel(warningText)
		warningLabel.TextStyle = fyne.TextStyle{Bold: true}
		summaryForm.Append("Attribute Status", warningLabel)
	} else if len(info.Attributes) > 0 {
		summaryForm.Append("Attribute Status", widget.NewLabel("✓ All attributes OK"))
	}

	infoLabel := widget.NewLabel("SMART (Self-Monitoring, Analysis and Reporting Technology) monitors disk health and predicts failures.")
	infoLabel.Wrapping = fyne.TextWrapWord
	infoLabel.TextStyle = fyne.TextStyle{Italic: true}

	return container.NewVBox(
		statusBox,
		widget.NewSeparator(),
		summaryForm,
		widget.NewSeparator(),
		infoLabel,
	)
}

func (d *DiskInfoDialog) createAttributesTab(info *partition.DiskInfo) *fyne.Container {
	if len(info.Attributes) == 0 {
		return container.NewVBox(
			widget.NewLabel("No SMART attributes available for this disk."),
		)
	}

	// Create scrollable list of attributes
	attrList := widget.NewList(
		func() int {
			return len(info.Attributes)
		},
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel(""),
				widget.NewLabel(""),
				widget.NewLabel(""),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			cont := item.(*fyne.Container)
			attr := info.Attributes[id]

			// Attribute name and status
			nameLabel := cont.Objects[0].(*widget.Label)
			nameText := fmt.Sprintf("%d: %s", attr.ID, attr.Name)
			if attr.Status == "FAILING" {
				nameText += " ⚠️ FAILING"
				nameLabel.TextStyle = fyne.TextStyle{Bold: true}
			} else if attr.Status == "WARNING" {
				nameText += " ⚠️ WARNING"
			}
			nameLabel.SetText(nameText)

			// Values
			valueLabel := cont.Objects[1].(*widget.Label)
			valueLabel.SetText(fmt.Sprintf("Value: %d (Worst: %d, Threshold: %d)", attr.Value, attr.Worst, attr.Threshold))

			// Description and raw value
			descLabel := cont.Objects[2].(*widget.Label)
			descLabel.SetText(fmt.Sprintf("%s | Raw: %s", attr.Description, attr.RawValue))
			descLabel.TextStyle = fyne.TextStyle{Italic: true}
		},
	)

	legendLabel := widget.NewLabel("Legend: Value should stay above Threshold. Worst is the lowest recorded value.")
	legendLabel.Wrapping = fyne.TextWrapWord
	legendLabel.TextStyle = fyne.TextStyle{Italic: true}

	return container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), legendLabel),
		nil, nil,
		attrList,
	)
}

func (d *DiskInfoDialog) createCapabilitiesTab(info *partition.DiskInfo) *fyne.Container {
	capsList := container.NewVBox()

	if len(info.Capabilities) == 0 {
		capsList.Add(widget.NewLabel("No capabilities detected."))
	} else {
		for _, cap := range info.Capabilities {
			capLabel := widget.NewLabel("✓ " + cap)
			capsList.Add(capLabel)
		}
	}

	return container.NewVBox(
		widget.NewLabel("Disk Capabilities:"),
		widget.NewSeparator(),
		capsList,
	)
}
