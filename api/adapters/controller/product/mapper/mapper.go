package mapper

import (
	"e-commerce/adapters/controller/product/dto"
	entities "e-commerce/core/entities/product"
)

func DtoCreateToProduct(dto dto.ProcuctCreateRequestDTO) entities.Product {

	product := entities.Product{
		Name:        dto.Name,
		Description: dto.Description,
		Sku:         dto.Sku,
		Price:       dto.Price,
		Image:       dto.Image,
		Quantity:    dto.Quantity,
	}

	return product
}

func DtoUpdateToProduct(dto dto.ProcuctUpdateRequestDTO) entities.Product {

	product := entities.Product{
		Name:        dto.Name,
		Description: dto.Description,
		Sku:         dto.Sku,
		Price:       dto.Price,
		Image:       dto.Image,
		Quantity:    dto.Quantity,
	}

	return product
}

func ProductToDto(product entities.Product) dto.ProcuctResponseDTO {

	dto := dto.ProcuctResponseDTO{
		Id:          product.Id,
		Name:        product.Name,
		Description: product.Description,
		Sku:         product.Sku,
		Price:       product.Price,
		Image:       product.Image,
		Quantity:    product.Quantity,
		CreateAt:    product.CreateAt,
		UpdateAt:    product.UpdateAt,
	}

	return dto

}
