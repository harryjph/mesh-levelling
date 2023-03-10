package main

import (
	"github.com/ncruces/zenity"
	"log"
	"mesh-levelling/pkg/bltouch"
	"mesh-levelling/pkg/mesh"
	"mesh-levelling/pkg/printer"
	"os"
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
	log.Println("Connecting to printer...")
	printer, err := printer.NewPrinter("HarryPrinter:8899")
	if err != nil {
		panic(err)
	}
	defer printer.Close()

	log.Println("Connecting to BLTouch...")
	bltouch, err := bltouch.NewBLTouch("HarryUnoWifiRev2.lan:9988")
	if err != nil {
		panic(err)
	}
	defer bltouch.Close()

	mcp := MeshCreationParameters{
		MinX:                    -75,
		MinY:                    -75,
		MaxX:                    75,
		MaxY:                    75,
		NumberOfPointsPerSide:   7,
		NumberOfRepeatsPerPoint: 1,
	}

	meshPoints := make([]mesh.Point, 0, mcp.NumberOfPointsPerSide*mcp.NumberOfPointsPerSide)
	var averageZ float64
	var averageZCount uint

	log.Println("Starting...")
	reverseYDirection := false
	numberOfPoints := mcp.NumberOfPointsPerSide * mcp.NumberOfPointsPerSide * mcp.NumberOfRepeatsPerPoint
	numberOfPointsDone := 0
	printProgress := func() {
		numberOfPointsDone++
		log.Printf("%.1f%%\r\n", float64(numberOfPointsDone)/float64(numberOfPoints)*100.0)
	}
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
				log.Println("X:", xCoordinate, "Y:", yCoordinate)
				newZ, err := bltouch.GetZAtPoint(printer, xCoordinate, yCoordinate)
				if err != nil {
					panic(err)
				}
				z += newZ
				printProgress()
			}
			z /= float64(mcp.NumberOfRepeatsPerPoint)
			meshPoint := mesh.Point{X: xCoordinate, Y: yCoordinate, Z: z}
			meshPoints = append(meshPoints, meshPoint)
			averageZ += z
			averageZCount++
		}
		reverseYDirection = !reverseYDirection
	}

	averageZ /= float64(averageZCount)

	resultingMesh := mesh.Mesh{
		BLTouchHeight:   averageZ,
		Points:          meshPoints,
		Interpolator:    nil,
		MaterialOffsets: make(map[string]float64),
	}

	openMeshConfig := []zenity.Option{
		zenity.Title("Open Mesh"),
		zenity.FileFilter{
			Name:     "Mesh",
			Patterns: []string{"*.mesh"},
			CaseFold: true,
		},
	}

	for {
		file, err := zenity.SelectFile(openMeshConfig...)
		if err == nil {
			// Backup old file first
			if err := copyFile(file, file+".backup"); err != nil {
				log.Fatalln("Could not back up mesh")
			}
			oldMesh, err := mesh.LoadMesh(file)
			if err == nil {
				// Update the existing mesh's BLTouchHeight FIRST before updating the mesh points
				// Find a common point between the two meshes
				commonPointFound := false
				var i, j int
			outerLoop:
				for i = range resultingMesh.Points {
					for j = range oldMesh.Points {
						if resultingMesh.Points[i].X == oldMesh.Points[j].X && resultingMesh.Points[i].Y == oldMesh.Points[j].Y {
							// Found a matching point
							commonPointFound = true
							break outerLoop
						}
					}
				}
				if commonPointFound {
					newCommonZ := resultingMesh.Points[i].Z
					oldCommonZ := oldMesh.Points[j].Z
					oldMesh.BLTouchHeight += oldCommonZ - newCommonZ
				} else {
					log.Println("Could not find common point between the two meshes, switching to averaging method")
					var oldAverageZ float64
					for i := range oldMesh.Points {
						oldAverageZ += oldMesh.Points[i].Z
					}
					oldAverageZ /= float64(len(oldMesh.Points))
					oldMesh.BLTouchHeight += oldAverageZ - averageZ
				}

				// Update existing mesh points
				oldMesh.Points = resultingMesh.Points

				if err := mesh.SaveMesh(oldMesh, file); err != nil {
					panic(err)
				}

				log.Println("Complete! Mesh Updated.")
				return
			} else {
				log.Println("Error opening Mesh. Creating mesh instead.")
			}
		}
		break
	}

	if err := mesh.SaveMesh(&resultingMesh, "newMesh.mesh"); err != nil {
		panic(err)
	}

	log.Println("Complete! Mesh Created.")
}

func copyFile(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	err = os.WriteFile(to, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
