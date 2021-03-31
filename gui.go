package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/harry1453/go-common-file-dialog/cfd"
	"github.com/harry1453/go-common-file-dialog/cfdutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	openMeshConfig = cfd.DialogConfig{
		Title:       "Open Mesh",
		Role:        "open-mesh",
		FileFilters: []cfd.FileFilter{{DisplayName: "Mesh (*.mesh)", Pattern: "*.mesh"}},
	}
	openGCodeConfig = cfd.DialogConfig{
		Title:       "Open GCode",
		Role:        "open-gcode",
		FileFilters: []cfd.FileFilter{{DisplayName: "GCode (*.g, *.gcode, *.gx)", Pattern: "*.g;*.gcode;*.gx"}},
	}
)

func runGui() {
	a := app.New()
	w := a.NewWindow("Mesh Leveller")
	w.Resize(fyne.NewSize(512, 256))

	var mesh *Mesh

	loadedLabel := widget.NewLabel("No Mesh Loaded")
	loadedLabel.Alignment = fyne.TextAlignCenter

	var selectedMaterial string

	materialSelector := widget.NewSelect([]string{}, func(newOption string) {
		selectedMaterial = newOption
	})

	processButton := widget.NewButton("Process", func() {
		if mesh != nil {
			fileName, err := cfdutil.ShowOpenFileDialog(openGCodeConfig)
			if err == nil {
				processedFile, err := ProcessFile(fileName, mesh, selectedMaterial)
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
			file, err := cfdutil.ShowOpenFileDialog(openMeshConfig)
			if err == nil {
				newMesh, err := LoadMesh(file)
				if err != nil {
					dialog.NewError(err, w).Show()
					return
				}
				mesh = newMesh

				var materials []string
				for material := range mesh.MaterialOffsets {
					materials = append(materials, material)
				}
				materialSelector.Options = materials
				materialSelector.SetSelectedIndex(0)

				loadedLabel.SetText("Mesh Loaded: " + filepath.Base(file))
				processButton.Enable()
			}
		}),
		materialSelector,
		processButton,
	))

	w.ShowAndRun()
}
