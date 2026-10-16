package mapper

import (
	"e-commerce/adapters/controller/cart/dto"

	entitiesCart "e-commerce/core/entities/cart"
	entitiesCartProduct "e-commerce/core/entities/cart_product"

	"github.com/google/uuid"
)

func ProductCartToDto(productCart entitiesCartProduct.CartProduct) dto.ProcuctCartResponseDTO {

	price := productCart.Product.Price * float64(productCart.Quantity)

	dto := dto.ProcuctCartResponseDTO{
		Id:          productCart.Id,
		ProductId:   productCart.Product.Id,
		Quantity:    productCart.Quantity,
		Name:        productCart.Product.Name,
		Description: productCart.Product.Description,
		Sku:         productCart.Product.Sku,
		Price:       price,
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
