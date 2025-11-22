package service

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // JPEG形式のサポート
	_ "image/png"  // PNG形式のサポート
)

// ValidateImageData 画像データを検証
func ValidateImageData(data []byte) error {
	if len(data) == 0 {
		return errors.New("image data is empty")
	}

	if len(data) > 10*1024*1024 { // 10MB制限
		return errors.New("image size exceeds 10MB")
	}

	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid image format: %w", err)
	}

	allowedFormats := map[string]bool{"png": true, "jpeg": true, "jpg": true}
	if !allowedFormats[format] {
		return fmt.Errorf("unsupported format: %s", format)
	}

	return nil
}
