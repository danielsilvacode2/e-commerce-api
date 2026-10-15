package dto

import (
	fielderrors "e-commerce/adapters/controller/fieldErrors"
	"net/url"
	"path"
	"strings"
)

type ProcuctCreateRequestDTO struct {
	Name        string  `json:"name"`
	Sku         string  `json:"sku"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    int64   `json:"quantity"`
	Image       string  `json:"image"`
}

func (dto *ProcuctCreateRequestDTO) Validator() []fielderrors.FieldValidateErrors {

	var listFieldErrors []fielderrors.FieldValidateErrors

	if len(strings.TrimSpace(dto.Name)) < 2 {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "name",
			Message: "the name field is invalid and required",
		})
	}

	if len(strings.TrimSpace(dto.Sku)) < 2 {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "sku",
			Message: "the sku field is invalid and required",
		})
	}

	if len(strings.TrimSpace(dto.Description)) < 20 {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "description",
			Message: "the description field is invalid and required 20 caracteres min",
		})
	}

	if dto.Price < 0 {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "price",
			Message: "the price field is invalid and requires a positive value ",
		})
	}

	if dto.Quantity < 0 {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "quantity",
			Message: "the quantity field is invalid and requires a positive value ",
		})
	}

	imageURL, err := url.ParseRequestURI(strings.TrimSpace(dto.Image))
	imageExtension := ""
	if err == nil {
		imageExtension = strings.ToLower(path.Ext(imageURL.Path))
	}

	validImageExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
		".gif":  true,
		".avif": true,
		".svg":  true,
	}

	if !validImageExtensions[imageExtension] {
		listFieldErrors = append(listFieldErrors, fielderrors.FieldValidateErrors{
			Field:   "image",
			Message: "the image field must contain a valid image path",
		})
	}

	return listFieldErrors

}
