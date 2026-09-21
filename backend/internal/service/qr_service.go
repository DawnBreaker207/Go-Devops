package service

import (
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// qrPixelSize matches the size ticket emails have always embedded their QR at.
const qrPixelSize = 256

// GenerateQRPNG renders data (a ticket code) as a PNG-encoded QR code. It is the
// single place QR pixels get produced: ticket_email_service.go and the
// GET /tickets/{id}/qr endpoint both call this instead of encoding their own.
func GenerateQRPNG(data string) ([]byte, error) {
	png, err := qrcode.Encode(data, qrcode.Medium, qrPixelSize)
	if err != nil {
		return nil, fmt.Errorf("render QR: %w", err)
	}
	return png, nil
}
