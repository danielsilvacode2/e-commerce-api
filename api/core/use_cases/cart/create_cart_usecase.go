package use_cases

import entities "e-commerce/core/entities/cart"

func CreateCart( cart entities.Cart, repository entities.CartRepository) error {
	if err := repository.CreateCart(cart); err !=  nil {
		return  err
	}

	return nil
}