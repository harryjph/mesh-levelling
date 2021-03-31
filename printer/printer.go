package printer

import (
	"fmt"
	"net"
	"time"
)

type Printer struct {
	conn net.Conn
}

func NewPrinter(address string) (*Printer, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	printer := Printer{conn}
	if err := printer.execGcode("G90"); err != nil { // Set to absolute positioning
		return nil, err
	}
	return &printer, nil
}

func (printer *Printer) GoTo(x, y, z float64) error {
	command := fmt.Sprintf("G1 E0 F2000 X%.3f Y%.3f Z%.3f", x, y, z)
	return printer.execGcode(command)
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
