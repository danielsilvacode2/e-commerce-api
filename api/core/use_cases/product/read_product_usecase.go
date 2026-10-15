package use_cases

import entities "e-commerce/core/entities/product"


type ReadProductByIdUseCase struct{
	 repository entities.ProductRepository
}


func NewReadProductByIdUseCase ( repository entities.ProductRepository ) *ReadProductByIdUseCase{
	return &ReadProductByIdUseCase{
		repository: repository,
	}
}


func(useCase *ReadProductByIdUseCase) UseCase(ID string) (*entities.Product, error) {
	product, err :=  useCase.repository.ReadProdutById(ID)
	
	if err != nil{
		return nil, err
	}


	return product, nil

}
