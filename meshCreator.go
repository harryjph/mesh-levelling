package main

import (
	bltouch2 "MeshLevelling/bltouch"
	"MeshLevelling/mesh"
	printer2 "MeshLevelling/printer"
)

type MeshCreationParameters struct {
	MinX                    float64
	MinY                    float64
	MaxX                    float64
	MaxY                    float64
	NumberOfPointsPerSide   uint8
	NumberOfRepeatsPerPoint uint8
}

func main() {
	printer, err := printer2.NewPrinter("192.168.1.22:8899")
	if err != nil {
		panic(err)
	}

	bltouch, err := bltouch2.NewBLTouch("localhost:6969")
	if err != nil {
		panic(err)
	}

	mcp := MeshCreationParameters{
		MinX:                    -75,
		MinY:                    -75,
		MaxX:                    75,
		MaxY:                    75,
		NumberOfPointsPerSide:   7,
		NumberOfRepeatsPerPoint: 1,
	}

	meshPoints := make([]mesh.Point, 0, mcp.NumberOfPointsPerSide*mcp.NumberOfPointsPerSide)

	for xIndex := uint8(0); xIndex < mcp.NumberOfPointsPerSide; xIndex++ {
		xCoordinate := mcp.MinX + ((mcp.MaxX - mcp.MinX) * (float64(xIndex) / float64(mcp.NumberOfPointsPerSide-1)))
		for yIndex := uint8(0); yIndex < mcp.NumberOfPointsPerSide; yIndex++ {
			yCoordinate := mcp.MinY + ((mcp.MaxY - mcp.MinY) * (float64(yIndex) / float64(mcp.NumberOfPointsPerSide-1)))
			var z float64
			for i := uint8(0); i < mcp.NumberOfRepeatsPerPoint; i++ {
				newZ, err := bltouch.GetZAtPoint(printer, xCoordinate, yCoordinate)
				if err != nil {
					panic(err)
				}
				z += newZ
			}
			z /= float64(mcp.NumberOfRepeatsPerPoint)
			meshPoints = append(meshPoints, mesh.Point{X: xCoordinate, Y: yCoordinate, Z: z})
		}
	}

	resultingMesh := mesh.Mesh{
		Points:          meshPoints,
		Interpolator:    nil,
		MaterialOffsets: make(map[string]float64),
	}

	if err := mesh.SaveMesh(&resultingMesh, "newMesh.mesh"); err != nil {
		panic(err)
	}
}
