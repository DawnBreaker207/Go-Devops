package service

import (
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// qrPixelSize matches the size ticket emails have always embedded their QR at.
const qrPixelSize = 256

func GenerateQRPNG(data string) ([]byte, error) {
	png, err := qrcode.Encode(data, qrcode.Medium, qrPixelSize)
	if err != nil {
		return nil, fmt.Errorf("render QR: %w", err)
	}
	return png, nil
}
