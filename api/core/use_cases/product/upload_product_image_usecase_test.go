package use_cases

import (
	"context"
	"e-commerce/core/apperrors"
	"errors"
	"testing"
)

type fakeImageUploader struct {
	gotFilename string
	gotData     []byte
	returnURL   string
	returnErr   error
	called      bool
}

func (f *fakeImageUploader) UploadImage(ctx context.Context, filename string, data []byte) (string, error) {
	f.called = true
	f.gotFilename = filename
	f.gotData = data
	return f.returnURL, f.returnErr
}

func TestUploadProductImageUseCase(t *testing.T) {
	validPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	validJPEG := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	validWEBP := []byte{'R', 'I', 'F', 'F', 0x24, 0x00, 0x00, 0x00, 'W', 'E', 'B', 'P', 'V', 'P', '8', ' '}
	validGIF := []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")

	cases := []struct {
		name         string
		filename     string
		data         []byte
		wantFilename string
		wantErr      error
	}{
		{
			name:         "aceita PNG valido",
			filename:     "camiseta.png",
			data:         validPNG,
			wantFilename: "camiseta.png",
		},
		{
			name:         "aceita JPEG valido",
			filename:     "tenis.jpeg",
			data:         validJPEG,
			wantFilename: "tenis.jpg",
		},
		{
			name:         "aceita WEBP valido",
			filename:     "jaqueta.webp",
			data:         validWEBP,
			wantFilename: "jaqueta.webp",
		},
		{
			name:         "aceita GIF valido",
			filename:     "banner.gif",
			data:         validGIF,
			wantFilename: "banner.gif",
		},
		{
			name:         "sanitiza path traversal no nome do arquivo",
			filename:     `..\..\evil.exe`,
			data:         validPNG,
			wantFilename: "evil.png",
		},
		{
			name:    "rejeita arquivo vazio",
			data:    nil,
			wantErr: apperrors.ErrInvalidInput,
		},
		{
			name:    "rejeita conteudo nao imagem",
			data:    []byte("<html><script>alert(1)</script></html>"),
			wantErr: apperrors.ErrInvalidInput,
		},
		{
			name:    "rejeita arquivo maior que 5 MB",
			data:    append(validPNG, make([]byte, MaxProductImageSize)...),
			wantErr: apperrors.ErrPayloadTooLarge,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uploader := &fakeImageUploader{returnURL: "https://i.ibb.co/test/img.webp"}
			uc := NewUploadProductImageUseCase(uploader)

			gotURL, err := uc.UseCase(context.Background(), tc.filename, tc.data)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("erro = %v, esperado %v", err, tc.wantErr)
				}
				if uploader.called {
					t.Fatal("nao deveria chamar uploader quando a validacao falha")
				}
				return
			}

			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if gotURL != uploader.returnURL {
				t.Fatalf("url = %q, esperado %q", gotURL, uploader.returnURL)
			}
			if uploader.gotFilename != tc.wantFilename {
				t.Fatalf("filename = %q, esperado %q", uploader.gotFilename, tc.wantFilename)
			}
		})
	}
}

func TestUploadProductImageUseCasePropagaErroDoUploader(t *testing.T) {
	validPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	uploader := &fakeImageUploader{returnErr: apperrors.ErrImageUploadUnavailable}
	uc := NewUploadProductImageUseCase(uploader)

	if _, err := uc.UseCase(context.Background(), "foto.png", validPNG); !errors.Is(err, apperrors.ErrImageUploadUnavailable) {
		t.Fatalf("erro = %v, esperado ErrImageUploadUnavailable", err)
	}
}
