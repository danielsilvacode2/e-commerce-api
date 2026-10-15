package use_cases

import entities "e-commerce/core/entities/product"

func ReadProductById(ID string, repository entities.ProductRepository) (*entities.Product, error) {


	product, err := repository.ReadProdutById(ID)
	
	if err != nil{
		return nil, err
	}


	return product, nil

}
