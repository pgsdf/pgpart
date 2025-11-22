package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/pgsdf/pgpart/internal/partition"
)

// AttributesDialog manages the GPT attributes editing dialog
type AttributesDialog struct {
	window    fyne.Window
	partition *partition.Partition
	history   *partition.OperationHistory
	onUpdate  func()
}

// NewAttributesDialog creates a new attributes dialog
func NewAttributesDialog(window fyne.Window, part *partition.Partition, history *partition.OperationHistory, onUpdate func()) *AttributesDialog {
	return &AttributesDialog{
		window:    window,
		partition: part,
		history:   history,
		onUpdate:  onUpdate,
	}
}

// Show displays the attributes dialog
func (ad *AttributesDialog) Show() {
	// Check if partition supports GPT attributes
	err := partition.ValidatePartitionForAttributes(ad.partition.Name)
	if err != nil {
		dialog.ShowError(fmt.Errorf("This partition does not support GPT attributes.\n%v", err), ad.window)
		return
	}

	// Get current attributes
	attrInfo, err := partition.GetPartitionAttributes(ad.partition.Name)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to get partition attributes: %v", err), ad.window)
		return
	}

	// Create checkboxes for each attribute
	availableAttrs := partition.GetAvailableAttributes()
	checkboxes := make(map[string]*widget.Check)
	attrDescriptions := make(map[string]string)

	var attrWidgets []fyne.CanvasObject

	// Header
	header := widget.NewLabel(fmt.Sprintf("GPT Attributes for %s", ad.partition.Name))
	header.TextStyle = fyne.TextStyle{Bold: true}
	attrWidgets = append(attrWidgets, header)

	// Add separator
	attrWidgets = append(attrWidgets, widget.NewSeparator())

	// Create checkboxes
	for _, attr := range availableAttrs {
		check := widget.NewCheck(attr.Name, nil)
		check.Checked = attrInfo.Attributes[attr.Name]
		checkboxes[attr.Name] = check
		attrDescriptions[attr.Name] = attr.Description

		// Create a container with checkbox and description
		descLabel := widget.NewLabel(attr.Description)
		descLabel.Wrapping = fyne.TextWrapWord
		descLabel.TextStyle = fyne.TextStyle{Italic: true}

		attrContainer := container.NewVBox(
			check,
			descLabel,
			widget.NewSeparator(),
		)

		attrWidgets = append(attrWidgets, attrContainer)
	}

	// Info label
	infoLabel := widget.NewLabel("Note: Changes will be applied immediately when you click 'Apply'")
	infoLabel.Wrapping = fyne.TextWrapWord
	infoLabel.TextStyle = fyne.TextStyle{Italic: true}
	attrWidgets = append(attrWidgets, infoLabel)

	content := container.NewVBox(attrWidgets...)
	scrollContent := container.NewVScroll(content)
	scrollContent.SetMinSize(fyne.NewSize(500, 400))

	// Create a placeholder for the dialog so we can reference it in the button callbacks
	var customDialog dialog.Dialog

	// Add Apply button functionality
	applyBtn := widget.NewButton("Apply", func() {
		ad.applyAttributes(checkboxes, attrInfo)
	})

	closeBtn := widget.NewButton("Close", func() {
		if customDialog != nil {
			customDialog.Hide()
		}
	})

	// Create dialog content with Apply button
	dialogContent := container.NewBorder(
		nil,
		container.NewHBox(
			applyBtn,
			closeBtn,
		),
		nil,
		nil,
		scrollContent,
	)

	// Create the actual dialog
	customDialog = dialog.NewCustom("Edit GPT Attributes", "", dialogContent, ad.window)
	customDialog.Resize(fyne.NewSize(550, 500))
	customDialog.Show()
}

// applyAttributes applies the selected attributes
func (ad *AttributesDialog) applyAttributes(checkboxes map[string]*widget.Check, currentInfo *partition.AttributeInfo) {
	var errors []string
	var changes []string

	// Check each attribute and apply changes
	for attrName, checkbox := range checkboxes {
		wasSet := currentInfo.Attributes[attrName]
		nowSet := checkbox.Checked

		if wasSet != nowSet {
			var err error
			if nowSet {
				// Set the attribute
				err = partition.SetPartitionAttribute(ad.partition.Name, attrName)
				if err == nil {
					changes = append(changes, fmt.Sprintf("Set '%s'", attrName))
					// Record in history
					if ad.history != nil {
						ad.history.RecordAttributeChange(ad.partition.Name, attrName, wasSet, nowSet)
					}
				}
			} else {
				// Unset the attribute
				err = partition.UnsetPartitionAttribute(ad.partition.Name, attrName)
				if err == nil {
					changes = append(changes, fmt.Sprintf("Unset '%s'", attrName))
					// Record in history
					if ad.history != nil {
						ad.history.RecordAttributeChange(ad.partition.Name, attrName, wasSet, nowSet)
					}
				}
			}

			if err != nil {
				errors = append(errors, fmt.Sprintf("Failed to change '%s': %v", attrName, err))
			}
		}
	}

	// Show results
	if len(errors) > 0 {
		errorMsg := "Some attributes could not be changed:\n\n"
		for _, e := range errors {
			errorMsg += "• " + e + "\n"
		}
		dialog.ShowError(fmt.Errorf(errorMsg), ad.window)
	} else if len(changes) > 0 {
		successMsg := "Attributes updated successfully:\n\n"
		for _, c := range changes {
			successMsg += "• " + c + "\n"
		}
		dialog.ShowInformation("Success", successMsg, ad.window)

		// Update current info for subsequent changes
		newInfo, err := partition.GetPartitionAttributes(ad.partition.Name)
		if err == nil {
			*currentInfo = *newInfo
		}

		// Call the update callback
		if ad.onUpdate != nil {
			ad.onUpdate()
		}
	} else {
		dialog.ShowInformation("No Changes", "No attribute changes were made.", ad.window)
	}
}
