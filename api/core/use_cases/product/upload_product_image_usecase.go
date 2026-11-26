package use_cases

import (
	"context"
	"e-commerce/core/apperrors"
	"e-commerce/core/ports"
	"net/http"
	"path"
	"strings"
)

const MaxProductImageSize = 5 * 1024 * 1024 // 5 MB

var allowedProductImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

type UploadProductImageUseCase struct {
	uploader ports.ImageUploader
}

func NewUploadProductImageUseCase(uploader ports.ImageUploader) *UploadProductImageUseCase {
	return &UploadProductImageUseCase{
		uploader: uploader,
	}
}

func (useCase *UploadProductImageUseCase) UseCase(ctx context.Context, filename string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", apperrors.ErrInvalidInput
	}

	if len(data) > MaxProductImageSize {
		return "", apperrors.ErrPayloadTooLarge
	}

	detectedType := http.DetectContentType(data)
	ext, allowed := allowedProductImageTypes[detectedType]
	if !allowed {
		return "", apperrors.ErrInvalidInput
	}

	safeFilename := sanitizeImageFilename(filename, ext)
	return useCase.uploader.UploadImage(ctx, safeFilename, data)
}

func sanitizeImageFilename(rawFilename string, ext string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(rawFilename), "\\", "/")
	base := path.Base(normalized)
	if base == "" || base == "." || base == "/" {
		return "product" + ext
	}

	stem := strings.TrimSuffix(base, path.Ext(base))
	stem = strings.TrimSpace(stem)
	if stem == "" || stem == "." || stem == ".." {
		stem = "product"
	}

	return stem + ext
}
