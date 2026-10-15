package use_cases

import entities "e-commerce/core/entities/product"

func ReadAllProducts(repository entities.ProductRepository) ([]entities.Product, error) {
	listProduct, err := repository.ReadAllProduct()

	if err != nil {
		return nil, err
	}

	return listProduct, nil
}
