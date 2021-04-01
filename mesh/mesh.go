package mesh

import (
	"encoding/json"
	"errors"
	"github.com/RobinRCM/sklearn/interpolate"
	"os"
)

// The offset as set in the calibration function in the settings menu
const printerZOffset = 1

type Point struct {
	X float64
	Y float64
	Z float64
}

type Mesh struct {
	Points       []Point
	Interpolator func(x, y float64) (z float64) `json:"-"`
	// The adjustment for this material.
	MaterialOffsets map[string]float64
}

func LoadMesh(filename string) (*Mesh, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var mesh Mesh
	if err := json.NewDecoder(file).Decode(&mesh); err != nil {
		return nil, err
	}

	return &mesh, nil
}

func SaveMesh(mesh *Mesh, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(&mesh)
}

func (mesh *Mesh) GetZOffsetAtPosition(x, y float64, material string) (float64, error) {
	// TODO slowly phase out the offset as we move up the print
	materialOffset, ok := mesh.MaterialOffsets[material]
	if !ok {
		return 0, errors.New("material not found")
	}
	if mesh.Interpolator == nil {
		X := make([]float64, len(mesh.Points))
		Y := make([]float64, len(mesh.Points))
		Z := make([]float64, len(mesh.Points))
		for i := 0; i < len(mesh.Points); i++ {
			X[i] = mesh.Points[i].X
			Y[i] = mesh.Points[i].Y
			Z[i] = mesh.Points[i].Z
		}
		mesh.Interpolator = interpolate.Interp2d(X, Y, Z)
	}
	offset := mesh.Interpolator(x, y) + materialOffset
	if isValid(offset) {
		return offset - printerZOffset, nil
	} else {
		return 0, nil
	}
}
