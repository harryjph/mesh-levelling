package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/RobinRCM/sklearn/interpolate"
	"os"
	"strconv"
)

// The offset as set in the calibration function in the settings menu
const printerOffset = 1
const MeshSize = 9
const MeshSide = 3
const XYMin = -70
const XYCenter = 0
const XYMax = 70

type Mesh struct {
	X [MeshSize]float64
	Y [MeshSize]float64
	Z [MeshSize]float64
	Interpolator func(x, y float64) (z float64)
	// The adjustment for this material.
	MaterialOffsets map[string]float64
}

func LoadMesh(filename string) (*Mesh, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	csvReader := csv.NewReader(file)
	csvReader.FieldsPerRecord = -1

	all, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(all) < MeshSide + 1 {
		return nil, errors.New("not enough rows")
	}

	mesh := new(Mesh)

	for i := 0; i < MeshSide; i++ {
		if len(all[i]) < MeshSide {
			return nil, fmt.Errorf("not enough columns on row %d", i)
		}

		for j := 0; j < MeshSide; j++ {
			parsed, err := strconv.ParseFloat(all[i][j], 64)
			if err != nil {
				return nil, err
			}

			arrayIndex := MeshSide * i + j
			switch j {
			case 0:
				mesh.X[arrayIndex] = XYMin
			case 1:
				mesh.X[arrayIndex] = XYCenter
			case 2:
				mesh.X[arrayIndex] = XYMax
			}
			switch i {
			case 0:
				mesh.Y[arrayIndex] = XYMax
			case 1:
				mesh.Y[arrayIndex] = XYCenter
			case 2:
				mesh.Y[arrayIndex] = XYMin
			}
			mesh.Z[arrayIndex] = parsed - printerOffset
		}
	}

	mesh.MaterialOffsets = make(map[string]float64)
	for i := MeshSide; i < len(all); i++ {
		if len(all[i]) >= 2 {
			materialName := all[i][0]
			materialOffset, err := strconv.ParseFloat(all[i][1], 64)
			if err != nil {
				return nil, err
			}
			mesh.MaterialOffsets[materialName + " (" + strconv.FormatFloat(materialOffset, 'f', 2, 64) + "mm)"] = materialOffset
		}
	}

	mesh.Interpolator = interpolate.Interp2d(mesh.X[:], mesh.Y[:], mesh.Z[:])

	return mesh, nil
}

func (mesh *Mesh) GetZOffsetAtPosition(x, y float64, material string) (float64, error) {
	materialOffset, ok := mesh.MaterialOffsets[material]
	if !ok {
		return 0, errors.New("material not found")
	}
	offset := mesh.Interpolator(x, y) + materialOffset
	if isValid(offset) {
		return offset, nil
	} else {
		return 0, nil
	}
}
