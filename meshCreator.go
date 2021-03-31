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
	defer printer.Close()

	bltouch, err := bltouch2.NewBLTouch("192.168.1.13:9988")
	if err != nil {
		panic(err)
	}
	defer bltouch.Close()

	mcp := MeshCreationParameters{
		MinX:                    -70,
		MinY:                    -70,
		MaxX:                    70,
		MaxY:                    70,
		NumberOfPointsPerSide:   5,
		NumberOfRepeatsPerPoint: 3,
	}

	meshPoints := make([]mesh.Point, 0, mcp.NumberOfPointsPerSide*mcp.NumberOfPointsPerSide)
	var centerPoint *mesh.Point = nil

	reverseYDirection := false
	for xIndex := uint8(0); xIndex < mcp.NumberOfPointsPerSide; xIndex++ {
		xCoordinate := mcp.MinX + ((mcp.MaxX - mcp.MinX) * (float64(xIndex) / float64(mcp.NumberOfPointsPerSide-1)))
		for yIndex := uint8(0); yIndex < mcp.NumberOfPointsPerSide; yIndex++ {
			var actualYIndex uint8
			if reverseYDirection {
				actualYIndex = mcp.NumberOfPointsPerSide - 1 - yIndex
			} else {
				actualYIndex = yIndex
			}
			yCoordinate := mcp.MinY + ((mcp.MaxY - mcp.MinY) * (float64(actualYIndex) / float64(mcp.NumberOfPointsPerSide-1)))
			var z float64
			for i := uint8(0); i < mcp.NumberOfRepeatsPerPoint; i++ {
				newZ, err := bltouch.GetZAtPoint(printer, xCoordinate, yCoordinate)
				if err != nil {
					panic(err)
				}
				z += newZ
			}
			z /= float64(mcp.NumberOfRepeatsPerPoint)
			meshPoint := mesh.Point{X: xCoordinate, Y: yCoordinate, Z: z}
			meshPoints = append(meshPoints, meshPoint)
			if xCoordinate == 0 && yCoordinate == 0 {
				centerPoint = &meshPoint
			}
		}
		reverseYDirection = !reverseYDirection
	}

	if centerPoint == nil {
		var z float64
		for i := uint8(0); i < mcp.NumberOfRepeatsPerPoint; i++ {
			newZ, err := bltouch.GetZAtPoint(printer, 0, 0)
			if err != nil {
				panic(err)
			}
			z += newZ
		}
		z /= float64(mcp.NumberOfRepeatsPerPoint)
		meshPoint := mesh.Point{X: 0, Y: 0, Z: z}
		meshPoints = append(meshPoints, meshPoint)
		centerPoint = &meshPoint
	}

	for i := 0; i < len(meshPoints); i++ {
		meshPoints[i].Z -= centerPoint.Z
	}

	resultingMesh := mesh.Mesh{
		Points:          meshPoints,
		Interpolator:    nil,
		MaterialOffsets: make(map[string]float64),
	}

	resultingMesh.MaterialOffsets["Default"] = 0

	if err := mesh.SaveMesh(&resultingMesh, "newMesh.mesh"); err != nil {
		panic(err)
	}
}
