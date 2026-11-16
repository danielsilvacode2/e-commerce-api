package builder

import (
	controllerLogin "e-commerce/adapters/controller/auth"
	controllerCart "e-commerce/adapters/controller/cart"
	controllerProduct "e-commerce/adapters/controller/product"
	controllerUser "e-commerce/adapters/controller/user"
	"e-commerce/adapters/database"
	"e-commerce/adapters/email"
	"e-commerce/adapters/rabbitmq"
	repositoryCart "e-commerce/adapters/repository/cart"
	repositoryCartProduct "e-commerce/adapters/repository/cart_product"
	repositoryProduct "e-commerce/adapters/repository/product"
	repositoryUser "e-commerce/adapters/repository/user"
	"e-commerce/adapters/security"
	"e-commerce/config"
	"e-commerce/core/ports"
	useCasesCart "e-commerce/core/use_cases/cart"
	useCasesCartProduct "e-commerce/core/use_cases/cart_product"
	useCasesProduct "e-commerce/core/use_cases/product"
	useCasesUser "e-commerce/core/use_cases/user"
)

func BuilderUserController() *controllerUser.UserController {

	userRepository := repositoryUser.NewUserRepository(database.DB_MySQL)

	hashService := security.NewPasswordBcript()

	eventQueue := rabbitmq.NewUserRabbitMQ(rabbitmq.ChannelRabbitMQ)

	emailService := email.NewMailer(config.EmailHost, config.EmailPort, config.EmaiUser, config.EmailPassword, config.EmailFrom)
	createUser := useCasesUser.NewCreateUserUseCase(userRepository, hashService, emailService, eventQueue)
	deleteUser := useCasesUser.NewDeleteUserUseCase(userRepository)
	readAllUsers := useCasesUser.NewReadAllUsersUseCase(userRepository)
	updateUser := useCasesUser.NewUpdateUseUseCase(userRepository)
	readUserById := useCasesUser.NewReadUserByIdUseCase(userRepository)
	readUserByEmail := useCasesUser.NewReadUserByEmailUseCase(userRepository)
	verifyUserCode := useCasesUser.NewVerifyUserCodeUseCase(userRepository, readUserByEmail, hashService)
	newUserCode := useCasesUser.NewNewUserCodeUseCase(userRepository, updateUser, readUserByEmail, emailService, hashService)

	resetPassword := useCasesUser.NewResetPasswordUseCase(userRepository, *readUserByEmail, hashService, emailService)
	verifyUserPasswordCode := useCasesUser.NewVerifyUserCodePasswordUseCase(userRepository, *readUserByEmail, hashService)

	userController := controllerUser.NewUserController(
		createUser,
		deleteUser,
		readAllUsers,
		updateUser,
		readUserById,
		verifyUserCode,
		emailService,
		readUserByEmail,
		newUserCode,
		resetPassword,
		verifyUserPasswordCode,
	)

	return userController

}

func BuilderLoginController() *controllerLogin.AuthController {
	userRepository := repositoryUser.NewUserRepository(database.DB_MySQL)

	hashService := security.NewPasswordBcript()
	loginController := controllerLogin.NewAuthController(userRepository, hashService)

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

func BuilderConsumerRabbitMQ() ports.EventeUserQueueConsumerDeclare {
	emailService := email.NewMailer(config.EmailHost, config.EmailPort, config.EmaiUser, config.EmailPassword, config.EmailFrom)
	userRepository := repositoryUser.NewUserRepository(database.DB_MySQL)
	hashService := security.NewPasswordBcript()

	rabbitConsumer := rabbitmq.NewUserRabbitMQConsumer(rabbitmq.ChannelRabbitMQ, userRepository, emailService, hashService)

	return rabbitConsumer
}
