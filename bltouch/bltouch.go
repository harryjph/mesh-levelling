package bltouch

import (
	"MeshLevelling/printer"
	"errors"
	"math"
	"net"
	"time"
)

const (
	StartZ float64 = 57   // Don't bother going higher than this
	ZStep  float64 = 0.05 // Number of mm per Z step (resolution)
	EndZ   float64 = 50   // Don't ever go below this as it could crush the BLTouch

	SpeedXY    = 80 // mm per second
	SpeedZFast = 16 // mm per second
	SpeedZSlow = 1  // mm per second
)

type BLTouch struct {
	conn net.Conn
}

func NewBLTouch(address string) (*BLTouch, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	bltouch := BLTouch{conn}
	if err := bltouch.retract(); err != nil {
		return nil, err
	}
	return &bltouch, nil
}

func (bltouch *BLTouch) Close() error {
	return bltouch.conn.Close()
}

func (bltouch *BLTouch) retract() error {
	_, err := bltouch.conn.Write([]byte{'r'})
	return err
}

func (bltouch *BLTouch) extend() error {
	_, err := bltouch.conn.Write([]byte{'e'})
	return err
}

func (bltouch *BLTouch) hasTouched() (bool, error) {
	for retryCount := 0; retryCount < 3; retryCount++ {
		if _, err := bltouch.conn.Write([]byte{'t'}); err != nil {
			continue
		}
		buffer := make([]byte, 1)
		if err := bltouch.conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			continue
		}
		if _, err := bltouch.conn.Read(buffer); err != nil {
			continue
		}
		return buffer[0] == '1', nil
	}
	return false, errors.New("failed to read bltouch after 3 attempts")
}

func (bltouch *BLTouch) GetZAtPoint(printer *printer.Printer, x, y float64) (float64, error) {
	if err := bltouch.retract(); err != nil {
		return 0, err
	}
	if movementDuration, err := printer.MoveZ(StartZ, SpeedZFast); err != nil {
		return 0, err
	} else {
		time.Sleep(movementDuration)
	}
	if movementDuration, err := printer.MoveXY(x, y, SpeedXY); err != nil {
		return 0, err
	} else {
		time.Sleep(movementDuration)
	}
	if err := bltouch.extend(); err != nil {
		return 0, err
	}

	// We are ready to start moving down.
	for z := StartZ - ZStep; z >= EndZ; z -= ZStep {
		z = math.Round(z*1000) / 1000
		if movementDuration, err := printer.MoveZ(z, SpeedZSlow); err != nil {
			return 0, err
		} else {
			time.Sleep(movementDuration)
		}
		hasTouched, err := bltouch.hasTouched()
		if err != nil {
			return 0, err
		}
		if hasTouched {
			return z, nil
		}
	}

	return 0, errors.New("could not find bed without going below minimum safe Z")
}
