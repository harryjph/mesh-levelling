package bltouch

import (
	"MeshLevelling/printer"
	"net"
)

const (
	StartZ = 65 // Don't bother going higher than this
	EndZ   = 50 // Don't ever go below this as it could crush the BLTouch
)

type BLTouch struct {
	conn net.Conn
}

func NewBLTouch(address string) (*BLTouch, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return &BLTouch{conn}, nil
}

func (bltouch *BLTouch) GetZAtPoint(printer *printer.Printer, x, y float64) (float64, error) {
	return 1, nil // TODO
}
