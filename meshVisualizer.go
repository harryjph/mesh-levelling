package main

import (
	mesh2 "MeshLevelling/mesh"
	"github.com/tidwall/pinhole"
	"log"
	"math"
	"sort"
)

func main() {
	log.Println("Loading Mesh")
	mesh, err := mesh2.LoadMesh("newMesh.mesh")
	if err != nil {
		panic(err)
	}

	p := pinhole.New()
	sort.Slice(mesh.Points, func(i, j int) bool {
		if mesh.Points[i].X == mesh.Points[j].X {
			return mesh.Points[i].Y < mesh.Points[j].Y
		} else {
			return mesh.Points[i].X < mesh.Points[j].X
		}
	})

	p.Begin()
	scale := 1.0 / 150
	translateZ := func(z float64) float64 {
		const zScale = 5.0
		return (z - mesh.BLTouchHeight) * zScale
	}
	log.Println("Scale", scale)
	numberOfPointsPerSide := int(math.Sqrt(float64(len(mesh.Points))))
	log.Println("NPPS", numberOfPointsPerSide)
	for i := 0; i < numberOfPointsPerSide; i++ {
		for j := 0; j < numberOfPointsPerSide; j++ {
			point := &mesh.Points[i*numberOfPointsPerSide+j]
			if i == 0 && j == 0 {
				p.DrawDot(point.X, point.Y, translateZ(point.Z), 3*scale)
			}
			if i+1 < numberOfPointsPerSide {
				nextPointX := mesh.Points[(i+1)*numberOfPointsPerSide+j]
				p.DrawLine(point.X, point.Y, translateZ(point.Z), nextPointX.X, nextPointX.Y, translateZ(nextPointX.Z))
			}
			if j+1 < numberOfPointsPerSide {
				nextPointY := mesh.Points[i*numberOfPointsPerSide+(j+1)]
				p.DrawLine(point.X, point.Y, translateZ(point.Z), nextPointY.X, nextPointY.Y, translateZ(nextPointY.Z))
			}
		}
	}
	p.End()
	p.Scale(scale, scale, scale)
	//p.Rotate(math.Pi / 4, math.Pi / 4, -math.Pi / 6)
	p.Rotate(math.Pi/2, math.Pi/4, math.Pi)
	p.Rotate(math.Pi/4, math.Pi, 0)
	log.Println("Saving Mesh Image")
	if err := p.SavePNG("R:\\mesh.png", 750, 750, pinhole.DefaultImageOptions); err != nil {
		panic(err)
	}
}
