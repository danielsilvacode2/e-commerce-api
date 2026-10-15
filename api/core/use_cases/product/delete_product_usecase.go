package use_cases

import entities "e-commerce/core/entities/product"

func DeleteProductById(Id string, repository entities.ProductRepository) error {

 	if err := repository.DeleteProductById(Id); err != nil {
		return err
	}
	
	return nil
}