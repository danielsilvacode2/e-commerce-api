package controller

import (
	"e-commerce/adapters/controller/chat/dto"
	"e-commerce/adapters/controller/helpers"
	"e-commerce/adapters/security/auth"
	"e-commerce/core/apperrors"
	useCasesChat "e-commerce/core/use_cases/chat"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ChatController struct {
	sendMessage *useCasesChat.SendMessageUseCase
}

func NewChatController(sendMessage *useCasesChat.SendMessageUseCase) *ChatController {
	return &ChatController{
		sendMessage: sendMessage,
	}
}

func (chatCtrl *ChatController) getClaims(c *gin.Context) (*auth.Claims, error) {

	val, exists := c.Get("claims")

	if !exists {
		return nil, apperrors.ErrUnauthorized
	}

	claims, ok := val.(*auth.Claims)

	if !ok {
		return nil, errors.New("Error converting declarations")
	}

	return claims, nil
}

// SendMessage envia a mensagem do usuario logado ao chatbot e retorna a resposta.
// @Summary      Enviar mensagem ao assistente
// @Description  Envia uma mensagem do usuario autenticado para o chatbot (OpenRouter / Gemini 2.5 Flash Lite) e retorna a resposta gerada.
// @Tags         Chat
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        request  body      dto.ChatRequestDTO  true  "Mensagem do usuario"
// @Success      200      {object}  helpers.SwaggerDataResponse[dto.ChatResponseDTO]
// @Failure      400      {object}  helpers.SwaggerValidationErrorResponse
// @Failure      401      {object}  helpers.SwaggerMessageResponse
// @Failure      503      {object}  helpers.SwaggerMessageResponse
// @Failure      500      {object}  helpers.SwaggerMessageResponse
// @Router       /chat [post]
func (chatCtrl *ChatController) SendMessage(c *gin.Context) {

	claims, err := chatCtrl.getClaims(c)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	var dtoRequest dto.ChatRequestDTO

	if err = c.ShouldBindJSON(&dtoRequest); err != nil {
		helpers.HelperErrResponse(nil, c, apperrors.ErrInvalidInput)
		return
	}

	if listErrField := dtoRequest.Validator(); listErrField != nil {
		helpers.HelperErrResponse(listErrField, c, apperrors.ErrInvalidInput)
		return
	}

	reply, err := chatCtrl.sendMessage.UseCase(c.Request.Context(), claims.Id.String(), dtoRequest.Message)

	if err != nil {
		helpers.HelperErrResponse(nil, c, err)
		return
	}

	helpers.HelperResponse(dto.ChatResponseDTO{Reply: reply}, http.StatusOK, "chat reply", c)
}
