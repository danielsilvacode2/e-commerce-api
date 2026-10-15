package use_cases

import entities "e-commerce/core/entities/product"

type DeleteProductByIdUseCase struct{
	 repository entities.ProductRepository
}

func NewDeleteProductByIdUseCase ( repository entities.ProductRepository ) *DeleteProductByIdUseCase{
	return &DeleteProductByIdUseCase{
		repository: repository,
	}
}

func (useCase *DeleteProductByIdUseCase) UseCase(Id string) error {

 	if err :=  useCase.repository.DeleteProductById(Id); err != nil {
		return err
	}
	
	return nil
}