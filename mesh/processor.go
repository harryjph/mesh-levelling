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

func isValid(value float64) bool {
	return !(math.IsNaN(value) || math.IsInf(value, -1) || math.IsInf(value, 1))
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
	var extruder, speed, x, y, z, adjustedZ float64
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

				newExtruder, err := handleMoveArgument(extruderRegex, relativeExtruderPositioning, extruder)
				if err != nil {
					return "", err
				}
				newSpeed, err := handleMoveArgument(speedRegex, false, speed)
				if err != nil {
					return "", err
				}
				newX, err := handleMoveArgument(xRegex, relativePositioning, x)
				if err != nil {
					return "", err
				}
				newY, err := handleMoveArgument(yRegex, relativePositioning, y)
				if err != nil {
					return "", err
				}
				newZ, err := handleMoveArgument(zRegex, relativePositioning, z)
				if err != nil {
					return "", err
				}

				zOffset, err := mesh.GetZOffsetAtPosition(newX, newY, material)
				if err != nil {
					return "", err
				}
				newAdjustedZ := newZ + zOffset

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
