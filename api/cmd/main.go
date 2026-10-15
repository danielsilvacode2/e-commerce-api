package main

import (
	controllerLogin "e-commerce/adapters/controller/login"
	controllerProduct "e-commerce/adapters/controller/product"
	controllerUser "e-commerce/adapters/controller/user"
	"e-commerce/adapters/database"
	repositoryProduct "e-commerce/adapters/repository/product"
	repositoryUser "e-commerce/adapters/repository/user"
	"e-commerce/adapters/security"
	"e-commerce/adapters/server/router"
	"e-commerce/config"
	useCasesProduct "e-commerce/core/use_cases/product"
	useCasesUser "e-commerce/core/use_cases/user"
)

func init() {
	config.LoadConfig()
	database.NewMySQLConnection()
}

func main() {

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

	loginController := controllerLogin.NewLoginController(userRepository, passwordService)

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
		readProductById)

	r := router.NewGinRouter(userController, loginController, productController)
	r.Run(config.ApiPort)

}
