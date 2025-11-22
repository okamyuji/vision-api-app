package service

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func createTestPNGImage(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// 白で塗りつぶし
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestValidateImageData(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "空データ",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "有効なPNG画像",
			data:    createTestPNGImage(100, 100),
			wantErr: false,
		},
		{
			name:    "無効なデータ",
			data:    []byte("invalid image data"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateImageData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateImageData_SizeLimit(t *testing.T) {
	// 10MB超のデータ
	largeData := make([]byte, 11*1024*1024)
	err := ValidateImageData(largeData)
	if err == nil {
		t.Error("Expected error for large image, got nil")
	}
}
