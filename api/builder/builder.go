package builder

import (
	adapterBrasilAPI "e-commerce/adapters/brasilapi"
	controllerAddress "e-commerce/adapters/controller/address"
	controllerLogin "e-commerce/adapters/controller/auth"
	controllerCart "e-commerce/adapters/controller/cart"
	controllerChat "e-commerce/adapters/controller/chat"
	controllerOrder "e-commerce/adapters/controller/order"
	controllerPayment "e-commerce/adapters/controller/payment"
	controllerProduct "e-commerce/adapters/controller/product"
	controllerUser "e-commerce/adapters/controller/user"
	"e-commerce/adapters/database"
	"e-commerce/adapters/email"
	adapterImgBB "e-commerce/adapters/imgbb"
	adapterOpenRouter "e-commerce/adapters/openrouter"
	adapterPayment "e-commerce/adapters/payment"
	"e-commerce/adapters/rabbitmq"
	repositoryAddress "e-commerce/adapters/repository/address"
	repositoryCart "e-commerce/adapters/repository/cart"
	repositoryCartProduct "e-commerce/adapters/repository/cart_product"
	repositoryOrder "e-commerce/adapters/repository/order"
	repositoryOrderItem "e-commerce/adapters/repository/order_item"
	repositoryOrderStatusHistory "e-commerce/adapters/repository/order_status_history"
	repositoryPayment "e-commerce/adapters/repository/payment"
	repositoryProduct "e-commerce/adapters/repository/product"
	repositoryUser "e-commerce/adapters/repository/user"
	"e-commerce/adapters/security"
	"e-commerce/config"
	"e-commerce/core/ports"
	useCasesAddress "e-commerce/core/use_cases/address"
	useCasesCart "e-commerce/core/use_cases/cart"
	useCasesCartProduct "e-commerce/core/use_cases/cart_product"
	useCasesChat "e-commerce/core/use_cases/chat"
	useCasesOrder "e-commerce/core/use_cases/order"
	useCasesOrderItem "e-commerce/core/use_cases/order_item"
	useCasesOrderStatusHistory "e-commerce/core/use_cases/order_status_history"
	useCasesPayment "e-commerce/core/use_cases/payment"
	useCasesProduct "e-commerce/core/use_cases/product"
	useCasesUser "e-commerce/core/use_cases/user"
	"log"
	"strings"
	"sync"
)

var (
	logEmailServiceOnce     sync.Once
	addressGatewayOnce      sync.Once
	sharedAddressLookupGw   ports.AddressLookupGateway
)

func buildAddressLookupGateway() ports.AddressLookupGateway {
	addressGatewayOnce.Do(func() {
		sharedAddressLookupGw = adapterBrasilAPI.NewBrasilAPIClient()
	})
	return sharedAddressLookupGw
}

func buildEmailService() ports.EmailService {
	from := strings.TrimSpace(config.EmailFrom)
	if from == "" {
		from = strings.TrimSpace(config.EmaiUser)
	}

	if strings.TrimSpace(config.BrevoAPIKey) != "" {
		logEmailServiceOnce.Do(func() {
			log.Println("[INFO] BREVO_API_KEY configurada; usando BrevoMailer (API HTTP).")
		})
		return email.NewBrevoMailer(config.BrevoAPIKey, from)
	}

	logEmailServiceOnce.Do(func() {
		log.Println("[INFO] BREVO_API_KEY não configurada; usando Mailer SMTP.")
	})
	return email.NewMailer(config.EmailHost, config.EmailPort, config.EmaiUser, config.EmailPassword, from)
}

func buildPaymentGateway() ports.PaymentGateway {
	if strings.TrimSpace(config.AccessTokenPayment) != "" {
		log.Println("[INFO] ACCESS_TOKEN_PAYMENT configurado; usando MercadoPagoGateway.")
		return adapterPayment.NewMercadoPagoGateway(config.AccessTokenPayment)
	}
	log.Println("[WARN] ACCESS_TOKEN_PAYMENT não configurado; usando MockPaymentGateway local.")
	return adapterPayment.NewMockPaymentGateway()
}

func BuilderUserController() *controllerUser.UserController {

	userRepository := repositoryUser.NewUserRepository(database.DB_MySQL)

	hashService := security.NewPasswordBcript()

	eventQueue := rabbitmq.NewUserRabbitMQ(rabbitmq.ChannelRabbitMQ)

	emailService := buildEmailService()
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
	searchProducts := useCasesProduct.NewSearchProductsUseCase(productRepository)

	if config.ImgBBAPIKey == "" {
		log.Println("[WARN] IMGBB_API_KEY não configurada; o endpoint de upload de imagem retornará indisponível.")
	}
	imageUploader := adapterImgBB.NewImgBBClient(config.ImgBBAPIKey)
	uploadProductImage := useCasesProduct.NewUploadProductImageUseCase(imageUploader)

	productController := controllerProduct.NewProductController(
		createProduct,
		readAllProduct,
		deleteProduct,
		updateProduct,
		readProductById,
		searchProducts,
		uploadProductImage,
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
	emailService := buildEmailService()
	userRepository := repositoryUser.NewUserRepository(database.DB_MySQL)
	hashService := security.NewPasswordBcript()

	rabbitConsumer := rabbitmq.NewUserRabbitMQConsumer(rabbitmq.ChannelRabbitMQ, userRepository, emailService, hashService)

	return rabbitConsumer
}

func BuilderOrderNotificationConsumerRabbitMQ() ports.EventOrderQueueConsumer {
	emailService := buildEmailService()
	return rabbitmq.NewOrderNotificationConsumer(rabbitmq.ChannelRabbitMQ, emailService)
}

func BuilderPaymentSyncConsumerRabbitMQ() ports.EventPaymentSyncConsumer {
	paymentRepository := repositoryPayment.NewPaymentRepository(database.DB_MySQL)
	emailService := buildEmailService()
	orderEventQueue := rabbitmq.NewOrderNotificationRabbitMQ(rabbitmq.ChannelRabbitMQ, emailService)
	return rabbitmq.NewPaymentSyncConsumer(rabbitmq.ChannelRabbitMQ, paymentRepository, orderEventQueue)
}

func BuilderOrderController() *controllerOrder.OrderController {

	orderRepository := repositoryOrder.NewOrderRepository(database.DB_MySQL)
	addressRepository := repositoryAddress.NewAddressRepository(database.DB_MySQL)
	paymentRepository := repositoryPayment.NewPaymentRepository(database.DB_MySQL)
	orderItemRepository := repositoryOrderItem.NewOrderItemRepository(database.DB_MySQL)
	orderStatusHistoryRepository := repositoryOrderStatusHistory.NewOrderStatusHistoryRepository(database.DB_MySQL)
	userRepository := repositoryUser.NewUserRepository(database.DB_MySQL)
	productRepository := repositoryProduct.NewProductRepository(database.DB_MySQL)
	cartProductRepository := repositoryCartProduct.NewCartProductRepository(database.DB_MySQL)

	emailService := buildEmailService()
	orderEventQueue := rabbitmq.NewOrderNotificationRabbitMQ(rabbitmq.ChannelRabbitMQ, emailService)
	paymentGateway := buildPaymentGateway()

	readUserById := useCasesUser.NewReadUserByIdUseCase(userRepository)
	readProductById := useCasesProduct.NewReadProductByIdUseCase(productRepository)
	readAllCartProduct := useCasesCartProduct.NewReadAllCartProductsUseCase(cartProductRepository)

	addressGateway := buildAddressLookupGateway()
	validateAddress := useCasesAddress.NewValidateAddressUseCase(addressGateway)

	readOrderById := useCasesOrder.NewReadOrderByIdUseCase(orderRepository)
	createOrder := useCasesOrder.NewCreateOrderUseCase(orderRepository, addressRepository, paymentRepository, readUserById, readAllCartProduct, validateAddress)
	readAllOrders := useCasesOrder.NewReadAllOrdersUseCase(orderRepository)
	readOrdersByUserId := useCasesOrder.NewReadOrdersByUserIdUseCase(orderRepository)
	updateOrder := useCasesOrder.NewUpdateOrderUseCase(orderRepository, addressRepository, readOrderById, validateAddress)
	updateOrderStatus := useCasesOrder.NewUpdateOrderStatusUseCase(orderRepository, readOrderById, orderEventQueue)
	cancelOrder := useCasesOrder.NewCancelOrderUseCase(orderRepository, readOrderById)
	requestRefund := useCasesOrder.NewRequestRefundUseCase(orderRepository, paymentRepository, readOrderById, paymentGateway, orderEventQueue)
	approveRefund := useCasesOrder.NewApproveRefundUseCase(orderRepository, paymentRepository, readOrderById, paymentGateway, orderEventQueue)
	rejectRefund := useCasesOrder.NewRejectRefundUseCase(orderRepository, readOrderById, orderEventQueue)
	deleteOrder := useCasesOrder.NewDeleteOrderUseCase(orderRepository)

	readOrderItems := useCasesOrderItem.NewReadOrderItemsUseCase(orderItemRepository, readOrderById)
	addOrderItem := useCasesOrderItem.NewAddOrderItemUseCase(orderItemRepository, readOrderById, readProductById)
	updateOrderItem := useCasesOrderItem.NewUpdateOrderItemUseCase(orderItemRepository, readOrderById)
	removeOrderItem := useCasesOrderItem.NewRemoveOrderItemUseCase(orderItemRepository, readOrderById)

	readOrderHistory := useCasesOrderStatusHistory.NewReadOrderStatusHistoryUseCase(orderStatusHistoryRepository, readOrderById)

	orderController := controllerOrder.NewOrderController(
		createOrder,
		readOrderById,
		readAllOrders,
		readOrdersByUserId,
		updateOrder,
		updateOrderStatus,
		cancelOrder,
		requestRefund,
		approveRefund,
		rejectRefund,
		deleteOrder,
		readOrderItems,
		addOrderItem,
		updateOrderItem,
		removeOrderItem,
		readOrderHistory,
	)

	return orderController
}

func BuilderChatController() *controllerChat.ChatController {

	orderRepository := repositoryOrder.NewOrderRepository(database.DB_MySQL)

	readOrderById := useCasesOrder.NewReadOrderByIdUseCase(orderRepository)
	readOrdersByUserId := useCasesOrder.NewReadOrdersByUserIdUseCase(orderRepository)
	if config.KeyOpenRouter == "" {
		log.Println("[WARN] KEY_OPEN_ROUTER não configurada; o endpoint de chat retornará indisponível.")
	}
	llmClient := adapterOpenRouter.NewOpenRouterClient(config.KeyOpenRouter, config.OpenRouterModel)
	sendMessage := useCasesChat.NewSendMessageUseCase(llmClient, readOrderById, readOrdersByUserId)

	return controllerChat.NewChatController(sendMessage)
}

func BuilderPaymentController() *controllerPayment.PaymentController {
	orderRepository := repositoryOrder.NewOrderRepository(database.DB_MySQL)
	paymentRepository := repositoryPayment.NewPaymentRepository(database.DB_MySQL)
	emailService := buildEmailService()
	orderEventQueue := rabbitmq.NewOrderNotificationRabbitMQ(rabbitmq.ChannelRabbitMQ, emailService)
	paymentSyncQueue := rabbitmq.NewPaymentSyncRabbitMQ(rabbitmq.ChannelRabbitMQ)
	paymentGateway := buildPaymentGateway()

	readOrderById := useCasesOrder.NewReadOrderByIdUseCase(orderRepository)
	processPayment := useCasesPayment.NewProcessPaymentUseCase(paymentRepository, readOrderById, paymentGateway, orderEventQueue, paymentSyncQueue)
	readPaymentByOrder := useCasesPayment.NewReadPaymentByOrderUseCase(paymentRepository, readOrderById, paymentGateway, orderEventQueue)
	webhookPayment := useCasesPayment.NewWebhookPaymentUseCase(paymentRepository, orderRepository, paymentGateway, orderEventQueue)

	return controllerPayment.NewPaymentController(processPayment, readPaymentByOrder, webhookPayment)
}

func BuilderAddressController() *controllerAddress.AddressController {
	addressRepository := repositoryAddress.NewAddressRepository(database.DB_MySQL)
	addressGateway := buildAddressLookupGateway()
	lookupAddressByCep := useCasesAddress.NewLookupAddressByCepUseCase(addressGateway)
	validateAddress := useCasesAddress.NewValidateAddressUseCase(addressGateway)
	readMyAddresses := useCasesAddress.NewReadMyAddressesUseCase(addressRepository)
	createAddress := useCasesAddress.NewCreateAddressUseCase(addressRepository, validateAddress)
	return controllerAddress.NewAddressController(lookupAddressByCep, readMyAddresses, createAddress)
}
