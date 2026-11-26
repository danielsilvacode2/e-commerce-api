package ports

import "context"

type ImageUploader interface {
	UploadImage(ctx context.Context, filename string, data []byte) (string, error)
}
