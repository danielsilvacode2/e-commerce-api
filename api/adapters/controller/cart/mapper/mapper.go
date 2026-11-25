package mapper

import (
	"e-commerce/adapters/controller/cart/dto"

	entitiesCart "e-commerce/core/entities/cart"
	entitiesCartProduct "e-commerce/core/entities/cart_product"
	entitiesProduct "e-commerce/core/entities/product"

	"github.com/google/uuid"
)

func ProductCartToDto(productCart entitiesCartProduct.CartProduct) dto.ProcuctCartResponseDTO {



	dto := dto.ProcuctCartResponseDTO{
		Id:          productCart.Id,
		ProductId:   productCart.Product.Id,
		IsSelect:    productCart.IsSelect,
		Quantity:    productCart.Quantity,
		Stock:       productCart.Product.Quantity,
		Name:        productCart.Product.Name,
		Description: productCart.Product.Description,
		Sku:         productCart.Product.Sku,
		Price:       productCart.Product.Price,
		Image:       productCart.Product.Image,
	}

	return dto

}

func UserToResponseCartDTO(cart entitiesCart.Cart) dto.UserCartResponseDTO {

	var userDto dto.UserCartResponseDTO

	userDto.Id = cart.User.ID
	userDto.Name = cart.User.Name
	userDto.Email = cart.User.Email
	userDto.Role = cart.User.Role

	return userDto

}

func CartResponseToDTO(userDTO dto.UserCartResponseDTO, ID uuid.UUID, listproductCartDTO []dto.ProcuctCartResponseDTO) dto.CartResponseDTO {

	dto := dto.CartResponseDTO{
		Id:          ID,
		User:        userDTO,
		ListProduct: listproductCartDTO,
	}

	return dto
}

func DtoRequestToCartProduct(dto dto.CartRequestDTO) (*entitiesCartProduct.CartProduct, error) {

	product := entitiesProduct.Product{}

	productUUID, err := uuid.Parse(dto.ProductId)

	if err != nil {
		return nil, err
	}

	product.Id = productUUID

	cartProduct := entitiesCartProduct.CartProduct{
		Product:  &product,
		Quantity: dto.Quantity,
	}

	return &cartProduct, nil
}
