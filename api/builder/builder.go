package builder

import (
	controllerCart "e-commerce/adapters/controller/cart"
	controllerLogin "e-commerce/adapters/controller/login"
	controllerProduct "e-commerce/adapters/controller/product"
	controllerUser "e-commerce/adapters/controller/user"
	"e-commerce/adapters/database"
	repositoryCart "e-commerce/adapters/repository/cart"
	repositoryCartProduct "e-commerce/adapters/repository/cart_product"
	repositoryProduct "e-commerce/adapters/repository/product"
	repositoryUser "e-commerce/adapters/repository/user"
	"e-commerce/adapters/security"
	useCasesCart "e-commerce/core/use_cases/cart"
	useCasesCartProduct "e-commerce/core/use_cases/cart_product"
	useCasesProduct "e-commerce/core/use_cases/product"
	useCasesUser "e-commerce/core/use_cases/user"
)

func BuilderUserController() *controllerUser.UserController {

	userRepository := repositoryUser.NewUserRepository(database.DB_MySQL)

	passwordService := security.NewPasswordBcript()

	createUser := useCasesUser.NewCreateUserUseCase(userRepository, passwordService)
	deleteUser := useCasesUser.NewDeleteUserUseCase(userRepository)
	readAllUsers := useCasesUser.NewReadAllUsersUseCase(userRepository)
	updateUser := useCasesUser.NewUpdateUseUseCase(userRepository)
	readUserById := useCasesUser.NewReadUserByIdUseCase(userRepository)

	userController := controllerUser.NewUserController(
		createUser,
		deleteUser,
		readAllUsers,
		updateUser,
		readUserById,
	)

	return userController

}

func BuilderLoginController() *controllerLogin.LoginController {
	userRepository := repositoryUser.NewUserRepository(database.DB_MySQL)

	passwordService := security.NewPasswordBcript()
	loginController := controllerLogin.NewLoginController(userRepository, passwordService)

	return loginController
}

func BuilderProductController() *controllerProduct.ProductController {

	productRepository := repositoryProduct.NewProductRepository(database.DB_MySQL)

	createProduct := useCasesProduct.NewCreateProductUseCase(productRepository)
	deleteProduct := useCasesProduct.NewDeleteProductByIdUseCase(productRepository)
	readAllProduct := useCasesProduct.NewReadAllProductsUseCase(productRepository)
	readProductById := useCasesProduct.NewReadProductByIdUseCase(productRepository)
	updateProduct := useCasesProduct.NewUpdateProductUseCase(productRepository)

	productController := controllerProduct.NewProductController(
		createProduct,
		readAllProduct,
		deleteProduct,
		updateProduct,
		readProductById,
	)

	return productController
}

func BuilderCartController() *controllerCart.CartController {

	cartRepository := repositoryCart.NewCartRepository(database.DB_MySQL)
	cartProductRepository := repositoryCartProduct.NewCartProductRepository(database.DB_MySQL)

	readOrCreateCart := useCasesCart.NewReadOrCreateCartUseCase(cartRepository)
	addProductCart := useCasesCartProduct.NewAddProductUseCase(cartProductRepository, *readOrCreateCart)
	readAllCartProduct := useCasesCartProduct.NewReadAllCartProductsUseCase(cartProductRepository)
	readCartProduct := useCasesCartProduct.NewReadCartProductUseCase(cartProductRepository)
	removeProductCart := useCasesCartProduct.NewRemoveProductCartUseCase(cartProductRepository, useCasesCartProduct.ReadCartProductUseCase(*readCartProduct))
	updateIsSelect := useCasesCartProduct.NewUpdateIsSelectUseCase(cartProductRepository, *readCartProduct)
	updateAllIsSelect := useCasesCartProduct.NewAllUpdateIsSelectUseCase(cartProductRepository)

	decreaseProductCart := useCasesCartProduct.NewDecreaseProductCartUseCase(cartProductRepository, *readCartProduct)

	cartController := controllerCart.NewCartController(*addProductCart, *readAllCartProduct, *removeProductCart, *updateIsSelect, *updateAllIsSelect, *decreaseProductCart)

	return cartController

}
