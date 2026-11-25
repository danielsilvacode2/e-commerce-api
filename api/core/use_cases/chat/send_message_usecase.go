package use_cases

import (
	"context"
	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	"e-commerce/core/ports"
	useCasesOrder "e-commerce/core/use_cases/order"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	uuidPattern    = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	accentReplacer = strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a",
		"é", "e", "ê", "e",
		"í", "i",
		"ó", "o", "ô", "o", "õ", "o",
		"ú", "u",
		"ç", "c",
	)

	proceduralPhrases = []string{
		"como solicito", "como pedir", "como faco", "como fazer", "como cancelar", "como pagar",
		"como comprar", "como funciona", "como acompanhar", "como ver", "onde vejo", "onde acompanho",
		"onde encontro", "onde fica", "qual o prazo", "prazo de devolucao", "prazo de troca", "politica de",
	}
	orderKeywords = []string{
		"pedido", "status", "reembolso", "estorno", "extorno", "rastreio", "rastreamento", "encomenda", "minha compra",
	}
)

func normalizeText(s string) string {
	return accentReplacer.Replace(strings.ToLower(s))
}

// shouldOfferTools define se as tools devem ser oferecidas ao modelo: somente quando o cliente
// informa um ID de pedido ou pergunta sobre status de pedido/reembolso.
func shouldOfferTools(message string) bool {
	if uuidPattern.MatchString(message) {
		return true
	}
	norm := normalizeText(message)
	for _, p := range proceduralPhrases {
		if strings.Contains(norm, p) {
			return false
		}
	}
	for _, k := range orderKeywords {
		if strings.Contains(norm, k) {
			return true
		}
	}
	return false
}

const (
	maxHistoryMessages = 20
	maxToolIterations  = 3

	toolConsultarPedido    = "consultar_pedido"
	toolConsultarReembolso = "consultar_reembolso"

	orderNotFoundMessage = "Pedido não encontrado"

	systemPrompt = `Você é o assistente virtual da loja NERO (moda casual e streetwear moderno, Coleção Urbana 2026). Responda de forma curta, cordial e em português.
Use consultar_pedido SOMENTE quando o cliente perguntar sobre o status de um pedido. Use consultar_reembolso SOMENTE quando o cliente perguntar sobre o andamento de um reembolso/estorno. Se o cliente informar o ID/número do pedido, passe exatamente esse valor em numero_pedido. Somente se ele perguntar sobre pedido/reembolso SEM informar o ID, deixe numero_pedido vazio para consultar o pedido mais recente. Nunca use ferramentas para saudações, dúvidas gerais sobre a loja ou perguntas de "como"/"onde" fazer algo: nesses casos responda com os passos ou informações abaixo. Nunca invente dados de pedidos: use apenas o que a ferramenta retornar.

Informações da loja NERO (use para responder dúvidas gerais, sem exagerar):
- Sobre: moda casual e streetwear moderno, com mais de 200 marcas parceiras (ex.: Versace, Zara, Gucci, Prada, Calvin Klein) e mais de 2.000 produtos.
- Frete e entrega: frete grátis em todas as compras, com entrega rápida e segura.
- Pagamento: criptografado, via Pix e cartão de crédito em até 12x.
- Boas-vindas: 20% de desconto no primeiro pedido ao se cadastrar.
- Trocas e reembolso: 30 dias para devolução ou troca. Pedido "Aguardando pagamento" pode ser cancelado em Meus Pedidos > Detalhes. Pedido "Em preparação" tem estorno automático ao solicitar. Pedido "Enviado" (em trânsito) não permite reembolso até a entrega. Pedido "Entregue" permite abrir solicitação de estorno com motivo, sujeita à aprovação da equipe.
- Conta: em Minha Conta o cliente vê seus dados, altera a senha (aba Segurança) e acessa o atendimento. Dá para buscar produtos na barra de busca e montar o carrinho mesmo antes de entrar na conta.

REGRA DE ESCOPO: JAMAIS responda qualquer assunto que não seja da loja NERO (política, esportes, receitas, programação, piadas, cultura geral etc.). Nesses casos diga educadamente que só pode ajudar com assuntos da loja NERO (produtos, compras, pagamentos, entregas, conta, pedidos e reembolsos).

Formatos de resposta:
- Status de pedido: "O status do seu pedido é X. Para mais detalhes, acesse o painel Meus Pedidos e clique em Detalhes."
- Status de reembolso: "O status do seu reembolso é X. Para mais detalhes, acompanhe no painel Meus Pedidos e clique em Detalhes."
- Como solicitar reembolso: "Acesse o painel Meus Pedidos, escolha o produto, clique em Abrir extorno, aguarde a aprovação e entregue o pedido de volta."
- Onde ver o status do pedido: "Acesse o painel Meus Pedidos, busque o pedido, escolha-o e clique em Detalhes."
- Se a ferramenta retornar "Pedido não encontrado", responda apenas que o pedido não foi encontrado.`
)

func chatTools() []ports.ToolDefinition {
	params := ports.ToolParameters{
		Type: "object",
		Properties: map[string]ports.ToolProperty{
			"numero_pedido": {Type: "string", Description: "ID/número do pedido informado pelo cliente. Passe exatamente o ID enviado; deixe vazio somente se o cliente perguntar sobre o pedido/reembolso sem informar o ID."},
		},
		Required: []string{},
	}

	return []ports.ToolDefinition{
		{
			Type: "function",
			Function: ports.ToolFunctionDefinition{
				Name:        toolConsultarPedido,
				Description: "Consulta o status de um pedido. Chame somente quando o cliente perguntar pelo status de um pedido. Nunca chame para saudações, dúvidas gerais da loja ou perguntas de como fazer algo.",
				Parameters:  params,
			},
		},
		{
			Type: "function",
			Function: ports.ToolFunctionDefinition{
				Name:        toolConsultarReembolso,
				Description: "Consulta o andamento do reembolso de um pedido. Chame somente quando o cliente perguntar pelo status de um reembolso/estorno. Nunca chame para saudações, dúvidas gerais da loja ou perguntas de como solicitar.",
				Parameters:  params,
			},
		},
	}
}

type SendMessageUseCase struct {
	llmClient          ports.LLMClient
	readOrderById      *useCasesOrder.ReadOrderByIdUseCase
	readOrdersByUserId *useCasesOrder.ReadOrdersByUserIdUseCase
	mu                 sync.RWMutex
	histories          map[string][]ports.ChatMessage
}

func NewSendMessageUseCase(
	llmClient ports.LLMClient,
	readOrderById *useCasesOrder.ReadOrderByIdUseCase,
	readOrdersByUserId *useCasesOrder.ReadOrdersByUserIdUseCase,
) *SendMessageUseCase {
	return &SendMessageUseCase{
		llmClient:          llmClient,
		readOrderById:      readOrderById,
		readOrdersByUserId: readOrdersByUserId,
		histories:          make(map[string][]ports.ChatMessage),
	}
}

// UseCase envia a mensagem ao modelo, executa as tools solicitadas (maximo de 3 iteracoes) e retorna a resposta final.
func (useCase *SendMessageUseCase) UseCase(ctx context.Context, userId string, message string) (string, error) {

	message = strings.TrimSpace(message)

	if message == "" {
		return "", apperrors.ErrInvalidInput
	}

	useCase.mu.RLock()
	history := append([]ports.ChatMessage(nil), useCase.histories[userId]...)
	useCase.mu.RUnlock()

	userMsg := ports.ChatMessage{Role: "user", Content: message}

	conversation := make([]ports.ChatMessage, 0, len(history)+2)
	conversation = append(conversation, ports.ChatMessage{Role: "system", Content: systemPrompt})
	conversation = append(conversation, history...)
	conversation = append(conversation, userMsg)

	newMessages := []ports.ChatMessage{userMsg}

	var tools []ports.ToolDefinition
	if shouldOfferTools(message) {
		tools = chatTools()
	}

	for i := 0; i <= maxToolIterations; i++ {

		// Na ultima iteracao nao oferece tools, forcando uma resposta textual
		currentTools := tools
		if i == maxToolIterations {
			currentTools = nil
		}

		resp, err := useCase.llmClient.Chat(ctx, conversation, currentTools)

		if err != nil {
			return "", err
		}

		conversation = append(conversation, resp.Message)
		newMessages = append(newMessages, resp.Message)

		if len(resp.Message.ToolCalls) == 0 || i == maxToolIterations {
			useCase.saveHistory(userId, append(history, newMessages...))
			return strings.TrimSpace(resp.Message.Content), nil
		}

		for _, tc := range resp.Message.ToolCalls {
			toolMsg := ports.ChatMessage{
				Role:       "tool",
				Content:    useCase.executeTool(tc, userId, message),
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
			}
			conversation = append(conversation, toolMsg)
			newMessages = append(newMessages, toolMsg)
		}
	}

	return "", apperrors.ErrServiceUnavailable
}

func (useCase *SendMessageUseCase) saveHistory(userId string, messages []ports.ChatMessage) {

	if len(messages) > maxHistoryMessages {
		messages = messages[len(messages)-maxHistoryMessages:]
	}

	// Evita iniciar o historico no meio de uma sequencia de tool calls
	for len(messages) > 0 && (messages[0].Role == "tool" || (messages[0].Role == "assistant" && len(messages[0].ToolCalls) > 0)) {
		messages = messages[1:]
	}

	useCase.mu.Lock()
	useCase.histories[userId] = append([]ports.ChatMessage(nil), messages...)
	useCase.mu.Unlock()
}

func (useCase *SendMessageUseCase) executeTool(tc ports.ToolCall, userId string, userMessage string) string {

	numeroPedido := ""
	if v, ok := tc.Function.Arguments["numero_pedido"]; ok && v != nil {
		numeroPedido = strings.TrimPrefix(strings.TrimSpace(fmt.Sprint(v)), "#")
		switch strings.ToLower(numeroPedido) {
		case "null", "none", "<nil>":
			numeroPedido = ""
		}
	}

	// Se o cliente informou o ID na mensagem, ele sempre prevalece sobre o pedido mais recente
	if numeroPedido == "" {
		numeroPedido = uuidPattern.FindString(userMessage)
	}

	var order *entitiesOrder.Order
	var err error

	if numeroPedido == "" {
		// Sem numero informado: usa o pedido mais recente do usuario (lista ordenada por created_at DESC)
		paginated, listErr := useCase.readOrdersByUserId.UseCase(userId, 1, 1, "")
		if listErr != nil || paginated == nil || len(paginated.Data) == 0 {
			return orderNotFoundMessage
		}
		order = &paginated.Data[0]
	} else {
		// Role fixa "user": garante que o pedido pertence ao usuario logado (inclusive para admins)
		order, err = useCase.readOrderById.UseCase(numeroPedido, userId, "user")
		if err != nil {
			return orderNotFoundMessage
		}
	}

	switch tc.Function.Name {
	case toolConsultarPedido:
		return orderStatusLabel(order.Status)
	case toolConsultarReembolso:
		return refundStatusLabel(order)
	}

	return "Ferramenta desconhecida"
}

func orderStatusLabel(status string) string {
	switch status {
	case entitiesOrder.OrderStatusPendingPayment:
		return "Aguardando pagamento"
	case entitiesOrder.OrderStatusPreparing:
		return "Em preparação"
	case entitiesOrder.OrderStatusShipped:
		return "Enviado"
	case entitiesOrder.OrderStatusDelivered:
		return "Entregue"
	case entitiesOrder.OrderStatusCanceled:
		return "Cancelado"
	case entitiesOrder.OrderStatusRefundRequested:
		return "Reembolso solicitado"
	case entitiesOrder.OrderStatusRefunded:
		return "Reembolsado"
	}
	return status
}

func refundStatusLabel(order *entitiesOrder.Order) string {
	switch {
	case order.Status == entitiesOrder.OrderStatusRefundRequested:
		return "Em análise (aguardando aprovação)"
	case order.Status == entitiesOrder.OrderStatusRefunded:
		return "Aprovado e estornado"
	case order.Status == entitiesOrder.OrderStatusDelivered && strings.TrimSpace(order.RefundRejectionReason) != "":
		return "Recusado"
	}
	return "Não há solicitação de reembolso para este pedido"
}
