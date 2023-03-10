package printer

import (
	"fmt"
	"math"
	"net"
	"time"
)

const (
	MovementTimeMultiplier = 1.5
)

type Printer struct {
	conn       net.Conn
	lastKnownX float64
	lastKnownY float64
	lastKnownZ float64
}

func NewPrinter(address string) (*Printer, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	printer := Printer{conn, 0, 0, 100}
	if err := printer.execGcode("G90"); err != nil { // Set to absolute positioning
		return nil, err
	}
	if err := printer.execGcode("G1 E0 F2000 X0 Y0 Z100"); err != nil { // Go to 0, 0, 100
		return nil, err
	}
	time.Sleep(5 * time.Second)
	return &printer, nil
}

func (printer *Printer) MoveXY(x, y, speed float64) (time.Duration, error) {
	command := fmt.Sprintf("G1 E0 F%.0f X%.3f Y%.3f", speed*60, x, y)
	if err := printer.execGcode(command); err != nil {
		return 0, err
	}
	distance := math.Sqrt(math.Pow(math.Abs(printer.lastKnownX-x), 2) + math.Pow(math.Abs(printer.lastKnownY-y), 2))
	movementDuration := time.Duration(distance/speed*1000*MovementTimeMultiplier) * time.Millisecond
	printer.lastKnownX = x
	printer.lastKnownY = y
	return movementDuration, nil
}

func (printer *Printer) MoveZ(z, speed float64) (time.Duration, error) {
	if z < 50 {
		panic("Z too low")
	}
	command := fmt.Sprintf("G1 E0 F%.0f Z%.3f", speed*60, z)
	if err := printer.execGcode(command); err != nil {
		return 0, err
	}
	movementDuration := time.Duration(math.Abs(printer.lastKnownZ-z)/speed*1000*MovementTimeMultiplier) * time.Millisecond
	printer.lastKnownZ = z
	return movementDuration, nil
}

func (printer *Printer) Close() error {
	return printer.conn.Close()
}

func (printer *Printer) execGcode(gcode string) error {
	if _, err := printer.conn.Write([]byte("~" + gcode + "\r\n")); err != nil {
		return err
	}
	if err := printer.conn.SetReadDeadline(time.Now().Add(100 * time.Second)); err != nil {
		return err
	}
	buffer := make([]byte, 1024)
	if _, err := printer.conn.Read(buffer); err != nil {
		return err
	} else {
		// TODO check response
		return nil
	}
}
