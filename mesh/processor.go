package mesh

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	absolutePositioningCommandRegex         = regexp.MustCompile("\\s*G90")
	relativePositioningCommandRegex         = regexp.MustCompile("\\s*G91")
	absoluteExtruderPositioningCommandRegex = regexp.MustCompile("\\s*M82")
	relativeExtruderPositioningCommandRegex = regexp.MustCompile("\\s*M83")
	homeAllCommandRegex                     = regexp.MustCompile("\\s*G28")
	homeMinimumCommandRegex                 = regexp.MustCompile("\\s*G161")
	homeMaximumCommandRegex                 = regexp.MustCompile("\\s*G162")
	moveCommandRegex                        = regexp.MustCompile("\\s*(G[0-3] |G92)")
	speedRegex                              = regexp.MustCompile("F([-.\\d]+)")
	extruderRegex                           = regexp.MustCompile("E([-.\\d]+)")
	xRegex                                  = regexp.MustCompile("X([-.\\d]+)")
	yRegex                                  = regexp.MustCompile("Y([-.\\d]+)")
	zRegex                                  = regexp.MustCompile("Z([-.\\d]+)")
)

const (
	Resolution          = 0.02 // Minimum move distance in mm
	MaximumMeshDeviance = 0.02 // Maximum distance that extruder is allowed to deviate from the mesh due to gcode being too simple
)

func isValid(value float64) bool {
	return !(math.IsNaN(value) || math.IsInf(value, -1) || math.IsInf(value, 1))
}

func calculateDistance(x, y, z float64) float64 {
	return math.Sqrt(math.Pow(x, 2) + math.Pow(y, 2) + math.Pow(z, 2))
}

func ProcessFile(filename string, mesh *Mesh, material string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var newLines []string

	// Current printer positions
	relativePositioning := true
	relativeExtruderPositioning := true
	// The current printer position **without offset**
	var extruder, x, y, z float64
	// The current printer position **with offset**
	var speed, adjustedZ float64
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(strings.TrimSpace(line), ";") {
			if moveCommandRegex.MatchString(line) {
				// This is a gcode move instruction!
				matches := moveCommandRegex.FindAllStringSubmatch(line, -1)
				if len(matches) != 1 {
					return "", fmt.Errorf("invalid argument count (%d): %s", len(matches), line)
				}
				if len(matches[0]) != 2 {
					return "", errors.New("regex error")
				}
				gcodeCommand := strings.TrimSpace(matches[0][1])

				handleMoveArgument := func(regex *regexp.Regexp, useRelativePositioning bool, oldValue float64) (float64, error) {
					if regex.MatchString(line) {
						matches := regex.FindAllStringSubmatch(line, -1)
						if len(matches) != 1 {
							return 0, fmt.Errorf("invalid argument count (%d): %s", len(matches), line)
						}
						if len(matches[0]) != 2 {
							return 0, errors.New("regex error")
						}

						newValue, err := strconv.ParseFloat(matches[0][1], 64)
						if err != nil {
							return 0, err
						}
						if useRelativePositioning {
							return oldValue + newValue, nil
						} else {
							return newValue, nil
						}
					} else {
						return oldValue, nil
					}
				}

				// The absolute extruder position **after** this command
				newExtruder, err := handleMoveArgument(extruderRegex, relativeExtruderPositioning, extruder)
				if err != nil {
					return "", err
				}
				// The speed **after and during** this command
				newSpeed, err := handleMoveArgument(speedRegex, false, speed)
				if err != nil {
					return "", err
				}
				// The absolute x position **after** this command
				newX, err := handleMoveArgument(xRegex, relativePositioning, x)
				if err != nil {
					return "", err
				}
				// The absolute y position **after** this command
				newY, err := handleMoveArgument(yRegex, relativePositioning, y)
				if err != nil {
					return "", err
				}
				// The absolute z position **after** this command
				newZ, err := handleMoveArgument(zRegex, relativePositioning, z)
				if err != nil {
					return "", err
				}

				zOffset, err := mesh.GetZOffsetAtPosition(newX, newY, material)
				if err != nil {
					return "", err
				}
				// The adjusted absolute z position **after** this command
				newAdjustedZ := newZ + zOffset

				// Detect the maximum deviation from the mesh to ensure that the mesh is followed accurately.
				// This avoids issues where eg. the bed is a perfect hill, and a command to move from one side to the other would crash into the hill.
				if isValid(x) && isValid(newX) && isValid(y) && isValid(newY) {
					changeInX := newX - x
					changeInY := newY - y
					// The angle of the moment in the XY plane
					angle := math.Atan2(changeInY, changeInX)
					// The distance of the movement in the XY plane
					distance := math.Sqrt(math.Pow(changeInX, 2) + math.Pow(changeInY, 2))

					// Slowly move along the line, trying to detect if the line strays too far from the mesh
					for partialDistance := float64(0); partialDistance < distance; partialDistance += Resolution {
						partialX := x + math.Cos(angle)*partialDistance
						partialY := y + math.Sin(angle)*partialDistance
						// The Z offset at this point
						partialZOffset, err := mesh.GetZOffsetAtPosition(partialX, partialY, material)
						if err != nil {
							return "", err
						}
						partialZ := z + partialZOffset

						// The Z position we'll be at if this line is not segmented
						interpolatedZ := adjustedZ + ((newAdjustedZ - adjustedZ) * (partialDistance / distance))

						if math.Abs(interpolatedZ-partialZ) > MaximumMeshDeviance {
							return "", errors.New("Z deviated too far") // TODO split up the movement into multiple smaller movements
						}
					}
				}

				// Compensate for any increases in distance by increasing extrusion length and speed.
				// Increases in distance come about due to the Z moving along with X and Y once mesh levelled, when only X and Y were supposed to move in the slicer's output.
				// To calculate the distance we need to know the change in X, change in Y, change in Z without adjustment and change in Z with adjustment.
				if isValid(x) && isValid(newX) && isValid(y) && isValid(newY) && isValid(z) && isValid(newZ) && isValid(adjustedZ) && isValid(newAdjustedZ) {
					changeInX := newX - x
					changeInY := newY - y
					changeInZ := newZ - z
					changeInAdjustedZ := newAdjustedZ - adjustedZ
					oldDistance := calculateDistance(changeInX, changeInY, changeInZ)
					if oldDistance != 0 { // prevent divide by 0
						adjustedDistance := calculateDistance(changeInX, changeInY, changeInAdjustedZ)
						distanceMultiplier := adjustedDistance / oldDistance
						// Adjust speed to compensate for increase in distance
						if isValid(newSpeed) {
							newSpeed *= distanceMultiplier
						}
						if isValid(extruder) && isValid(newExtruder) {
							// Adjust extrusion amount to compensate for increase in distance
							newExtruder = extruder + ((newExtruder - extruder) * distanceMultiplier)
						}
					}
				}

				newCommand := new(strings.Builder)
				newCommand.WriteString(gcodeCommand)
				newCommand.WriteRune(' ')

				writeParameter := func(oldValue, newValue float64, parameterPrefix rune, precision int, useRelativePositioning bool) {
					tenPowPrecision := math.Pow10(precision)
					newValueRounded := math.Round(newValue*tenPowPrecision) / tenPowPrecision
					oldValueRounded := math.Round(oldValue*tenPowPrecision) / tenPowPrecision
					if newValueRounded != oldValueRounded && isValid(newValue) && (!useRelativePositioning || isValid(oldValue)) {
						newCommand.WriteRune(parameterPrefix)
						if useRelativePositioning {
							newCommand.WriteString(strconv.FormatFloat(newValue-oldValue, 'f', precision, 64))
						} else {
							newCommand.WriteString(strconv.FormatFloat(newValue, 'f', precision, 64))
						}
						newCommand.WriteRune(' ')
					}
				}

				writeParameter(extruder, newExtruder, 'E', 5, relativeExtruderPositioning)
				writeParameter(speed, newSpeed, 'F', 0, false)
				writeParameter(x, newX, 'X', 3, relativePositioning)
				writeParameter(y, newY, 'Y', 3, relativePositioning)
				writeParameter(adjustedZ, newAdjustedZ, 'Z', 3, relativePositioning)

				extruder = newExtruder
				speed = newSpeed
				x = newX
				y = newY
				z = newZ
				adjustedZ = newAdjustedZ

				line = strings.TrimSpace(newCommand.String())
			} else if homeAllCommandRegex.MatchString(line) || homeMinimumCommandRegex.MatchString(line) {
				movex := strings.ContainsRune(line, 'X')
				movey := strings.ContainsRune(line, 'Y')
				movez := strings.ContainsRune(line, 'Z')
				if !(movex || movey || movez) {
					x = math.Inf(-1)
					y = math.Inf(-1)
					z = 0
				} else {
					if movex {
						x = math.Inf(-1)
					}
					if movey {
						y = math.Inf(-1)
					}
					if movez {
						z = 0
					}
				}
			} else if homeMaximumCommandRegex.MatchString(line) {
				movex := strings.ContainsRune(line, 'X')
				movey := strings.ContainsRune(line, 'Y')
				movez := strings.ContainsRune(line, 'Z')
				if !(movex || movey || movez) {
					x = math.Inf(1)
					y = math.Inf(1)
					z = math.Inf(1)
				} else {
					if movex {
						x = math.Inf(1)
					}
					if movey {
						y = math.Inf(1)
					}
					if movez {
						z = math.Inf(1)
					}
				}
			} else if absolutePositioningCommandRegex.MatchString(line) {
				relativePositioning = false
				relativeExtruderPositioning = false
			} else if relativePositioningCommandRegex.MatchString(line) {
				relativePositioning = true
				relativeExtruderPositioning = true
			} else if absoluteExtruderPositioningCommandRegex.MatchString(line) {
				relativeExtruderPositioning = false
			} else if relativeExtruderPositioningCommandRegex.MatchString(line) {
				relativeExtruderPositioning = true
			}
		}
		newLines = append(newLines, line)
	}

	return strings.Join(newLines, "\n"), nil
}
