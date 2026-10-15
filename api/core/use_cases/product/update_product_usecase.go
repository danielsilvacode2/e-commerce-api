package use_cases

import entities "e-commerce/core/entities/product"

func UpdateProduct(product entities.Product, repository entities.ProductRepository, ID string) error {
	if err := repository.UpdateProduct(product, ID); err != nil {
		return err
	}
	return nil
}
