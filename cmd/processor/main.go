package main

import (
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/ncruces/zenity"
	. "mesh-levelling/pkg/mesh"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	openMeshConfig = []zenity.Option{
		zenity.Title("Open Mesh"),
		zenity.FileFilter{
			Name:     "Mesh",
			Patterns: []string{"*.mesh"},
			CaseFold: true,
		},
	}
	openGCodeConfig = []zenity.Option{
		zenity.Title("Open GCode"),
		zenity.FileFilter{
			Name:     "GCode",
			Patterns: []string{"*.g", "*.gcode", "*.gx"},
			CaseFold: true,
		},
	}
)

func textPrompt(app fyne.App, title, prompt string) <-chan string {
	channel := make(chan string)
	window := app.NewWindow(title)
	textBox := widget.NewEntry()
	window.SetContent(container.NewVBox(widget.NewLabel(prompt), textBox, widget.NewButton("OK", func() {
		channel <- textBox.Text
		window.Close()
	})))
	window.SetOnClosed(func() {
		close(channel)
	})
	window.Resize(fyne.NewSize(256, 128))
	window.Show()
	return channel
}

func main() {
	a := app.New()
	w := a.NewWindow("Mesh Leveller")
	w.Resize(fyne.NewSize(512, 256))

	var currentMeshFilepath string
	var currentMesh *Mesh

	loadedLabel := widget.NewLabel("No Mesh Loaded")
	loadedLabel.Alignment = fyne.TextAlignCenter

	var selectedMaterial string

	materialOffsetTextBox := widget.NewEntry()
	blTouchHeightTextBox := widget.NewEntry()
	materialSelector := widget.NewSelect([]string{}, func(newOption string) {
		selectedMaterial = newOption
		materialOffset, ok := currentMesh.MaterialOffsets[selectedMaterial]
		if ok {
			materialOffsetTextBox.Text = strconv.FormatFloat(materialOffset, 'f', 3, 64)
		} else {
			materialOffsetTextBox.Text = "Error"
		}
		materialOffsetTextBox.Refresh()
	})

	processButton := widget.NewButton("Process", func() {
		if currentMesh != nil {
			fileName, err := zenity.SelectFile(openGCodeConfig...)
			if err == nil {
				processedFile, err := ProcessFile(fileName, currentMesh, selectedMaterial)
				if err != nil {
					dialog.NewError(err, w).Show()
					return
				}
				extension := filepath.Ext(fileName)
				fileNameWithoutExtension := strings.TrimSuffix(fileName, extension)
				if extension != ".gx" {
					extension = ".g"
				}
				newFileName := fileNameWithoutExtension + "_ML" + extension
				file, err := os.Create(newFileName)
				if err != nil {
					dialog.NewError(err, w).Show()
					return
				}
				defer file.Close()
				if _, err := file.WriteString(processedFile); err != nil {
					dialog.NewError(err, w).Show()
					return
				} else {
					dialog.NewInformation("Done!", "Processing complete!", w).Show()
				}
			}
		}
	})
	processButton.Disable()

	w.SetContent(container.NewVBox(
		loadedLabel,
		widget.NewButton("Load Mesh", func() {
			file, err := zenity.SelectFile(openMeshConfig...)
			if err == nil {
				newMesh, err := LoadMesh(file)
				if err != nil {
					dialog.NewError(err, w).Show()
					return
				}
				currentMesh = newMesh
				currentMeshFilepath = file

				var materials []string
				for material := range currentMesh.MaterialOffsets {
					materials = append(materials, material)
				}
				materialSelector.Options = materials
				materialSelector.SetSelectedIndex(0)
				blTouchHeightTextBox.SetText(strconv.FormatFloat(newMesh.BLTouchHeight, 'f', 3, 64))

				loadedLabel.SetText("Mesh Loaded: " + filepath.Base(file))
				processButton.Enable()
			}
		}),
		container.NewGridWithColumns(
			3,
			widget.NewLabel("BLTouch Height:"),
			blTouchHeightTextBox,
			widget.NewButton("Save", func() {
				if currentMesh != nil && currentMeshFilepath != "" {
					newBLTouchHeight, err := strconv.ParseFloat(blTouchHeightTextBox.Text, 64)
					if err != nil {
						dialog.NewError(err, w).Show()
						return
					}
					currentMesh.BLTouchHeight = newBLTouchHeight

					// Save Mesh
					if err := SaveMesh(currentMesh, currentMeshFilepath); err != nil {
						dialog.NewError(err, w).Show()
						return
					}

					dialog.NewInformation("Saved", "BLTouch Height Value Saved.", w).Show()
				}
			}),
		),
		container.NewGridWithColumns(
			4,
			materialSelector,
			materialOffsetTextBox,
			widget.NewButton("New", func() {
				if currentMesh == nil {
					dialog.NewError(errors.New("no mesh loaded"), w).Show()
					return
				}
				go func() {
					newMaterialName, ok := <-textPrompt(a, "Prompt", "Material Name:")
					if ok {
						for materialName := range currentMesh.MaterialOffsets {
							if materialName == newMaterialName {
								return
							}
						}
						currentMesh.MaterialOffsets[newMaterialName] = 0
						materialSelector.Options = append(materialSelector.Options, newMaterialName)
						materialSelector.SetSelectedIndex(0)
					}
				}()
			}),
			widget.NewButton("Save", func() {
				if currentMesh != nil && currentMeshFilepath != "" {
					newMaterialOffset, err := strconv.ParseFloat(materialOffsetTextBox.Text, 64)
					if err != nil {
						dialog.NewError(err, w).Show()
						return
					}
					currentMesh.MaterialOffsets[selectedMaterial] = newMaterialOffset

					// Save Mesh
					if err := SaveMesh(currentMesh, currentMeshFilepath); err != nil {
						dialog.NewError(err, w).Show()
						return
					}

					dialog.NewInformation("Saved", "Material Offset Value Saved.", w).Show()
				}
			}),
		),
		processButton,
	))

	w.ShowAndRun()
}
