package use_cases_test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"e-commerce/core/apperrors"
	entitiesOrder "e-commerce/core/entities/order"
	entitiesUser "e-commerce/core/entities/user"
	"e-commerce/core/ports"
	useCasesChat "e-commerce/core/use_cases/chat"
	useCasesOrder "e-commerce/core/use_cases/order"

	"github.com/google/uuid"
)

type fakeOrderRepository struct {
	orders map[string]*entitiesOrder.Order
}

func (f *fakeOrderRepository) CreateOrder(order entitiesOrder.Order, cartProductIds []string) error {
	return nil
}
func (f *fakeOrderRepository) ReadOrderById(ID string) (*entitiesOrder.Order, error) {
	o, ok := f.orders[ID]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return o, nil
}
func (f *fakeOrderRepository) ReadAllOrders(filter entitiesOrder.OrderFilter) (*entitiesOrder.PaginatedOrders, error) {
	var list []entitiesOrder.Order
	for _, o := range f.orders {
		if filter.UserId != "" && o.User.ID.String() != filter.UserId {
			continue
		}
		list = append(list, *o)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreateAt.After(list[j].CreateAt) })
	if filter.Limit > 0 && len(list) > filter.Limit {
		list = list[:filter.Limit]
	}
	return &entitiesOrder.PaginatedOrders{Data: list, Page: filter.Page, Limit: filter.Limit}, nil
}
func (f *fakeOrderRepository) UpdateOrderZipCode(zipCode string, ID string) error { return nil }
func (f *fakeOrderRepository) UpdateOrderStatus(ID string, currentStatus string, newStatus string, restoreStock bool) error {
	return nil
}
func (f *fakeOrderRepository) UpdateOrderStatusWithDetails(ID string, currentStatus string, newStatus string, restoreStock bool, trackingCode *string, refundReason *string, refundRejectionReason *string) error {
	return nil
}
func (f *fakeOrderRepository) DeleteOrder(ID string, restoreStock bool) error { return nil }

type fakeLLM struct {
	responses []ports.ChatMessage
	calls     [][]ports.ChatMessage
	toolsSeen []int
	err       error
}

func (f *fakeLLM) Chat(ctx context.Context, messages []ports.ChatMessage, tools []ports.ToolDefinition) (*ports.ChatResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.calls = append(f.calls, append([]ports.ChatMessage(nil), messages...))
	f.toolsSeen = append(f.toolsSeen, len(tools))
	idx := len(f.calls) - 1
	if idx >= len(f.responses) {
		idx = len(f.responses) - 1
	}
	return &ports.ChatResponse{Message: f.responses[idx]}, nil
}

func toolCall(name string, orderId string) ports.ChatMessage {
	return ports.ChatMessage{Role: "assistant", ToolCalls: []ports.ToolCall{{
		ID:       "call_1",
		Function: ports.ToolCallFunction{Name: name, Arguments: map[string]any{"numero_pedido": orderId}},
	}}}
}

func setup(llm *fakeLLM, status string, rejection string) (*useCasesChat.SendMessageUseCase, string, string) {
	ownerId := uuid.New()
	orderId := uuid.New()
	repo := &fakeOrderRepository{orders: map[string]*entitiesOrder.Order{
		orderId.String(): {Id: orderId, Status: status, RefundRejectionReason: rejection, User: &entitiesUser.User{ID: ownerId}},
	}}
	return useCasesChat.NewSendMessageUseCase(llm, useCasesOrder.NewReadOrderByIdUseCase(repo), useCasesOrder.NewReadOrdersByUserIdUseCase(repo)), ownerId.String(), orderId.String()
}

func TestSemNumeroUsaPedidoMaisRecente(t *testing.T) {
	ownerId := uuid.New()
	older := uuid.New()
	newer := uuid.New()
	repo := &fakeOrderRepository{orders: map[string]*entitiesOrder.Order{
		older.String(): {Id: older, Status: entitiesOrder.OrderStatusDelivered, CreateAt: time.Now().Add(-48 * time.Hour), User: &entitiesUser.User{ID: ownerId}},
		newer.String(): {Id: newer, Status: entitiesOrder.OrderStatusRefundRequested, CreateAt: time.Now(), User: &entitiesUser.User{ID: ownerId}},
	}}
	build := func(tool string) (*useCasesChat.SendMessageUseCase, *fakeLLM) {
		llm := &fakeLLM{responses: []ports.ChatMessage{toolCall(tool, ""), {Role: "assistant", Content: "ok"}}}
		return useCasesChat.NewSendMessageUseCase(llm, useCasesOrder.NewReadOrderByIdUseCase(repo), useCasesOrder.NewReadOrdersByUserIdUseCase(repo)), llm
	}

	uc, llm := build("consultar_pedido")
	if _, err := uc.UseCase(context.Background(), ownerId.String(), "status do meu pedido"); err != nil {
		t.Fatal(err)
	}
	if got := lastToolContent(t, llm); got != "Reembolso solicitado" {
		t.Fatalf("pedido: %q", got)
	}

	uc, llm = build("consultar_reembolso")
	if _, err := uc.UseCase(context.Background(), ownerId.String(), "e meu reembolso?"); err != nil {
		t.Fatal(err)
	}
	if got := lastToolContent(t, llm); got != "Em análise (aguardando aprovação)" {
		t.Fatalf("reembolso: %q", got)
	}

	// usuario sem pedidos
	uc, llm = build("consultar_pedido")
	if _, err := uc.UseCase(context.Background(), uuid.NewString(), "status"); err != nil {
		t.Fatal(err)
	}
	if got := lastToolContent(t, llm); got != "Pedido não encontrado" {
		t.Fatalf("sem pedidos: %q", got)
	}
}

func lastToolContent(t *testing.T, llm *fakeLLM) string {
	t.Helper()
	last := llm.calls[len(llm.calls)-1]
	for i := len(last) - 1; i >= 0; i-- {
		if last[i].Role == "tool" {
			return last[i].Content
		}
	}
	t.Fatal("no tool message sent to model")
	return ""
}

func TestConsultarPedidoExistente(t *testing.T) {
	llm := &fakeLLM{}
	uc, userId, orderId := setup(llm, entitiesOrder.OrderStatusShipped, "")
	llm.responses = []ports.ChatMessage{toolCall("consultar_pedido", orderId), {Role: "assistant", Content: "ok"}}

	reply, err := uc.UseCase(context.Background(), userId, "status do pedido "+orderId)
	if err != nil || reply != "ok" {
		t.Fatalf("unexpected: %q %v", reply, err)
	}
	if got := lastToolContent(t, llm); got != "Enviado" {
		t.Fatalf("tool content = %q", got)
	}
}

func TestPedidoInexistenteOuDeOutroUsuario(t *testing.T) {
	llm := &fakeLLM{}
	uc, _, orderId := setup(llm, entitiesOrder.OrderStatusShipped, "")
	llm.responses = []ports.ChatMessage{toolCall("consultar_pedido", orderId), {Role: "assistant", Content: "x"}}

	if _, err := uc.UseCase(context.Background(), uuid.NewString(), "pedido "+orderId); err != nil {
		t.Fatal(err)
	}
	if got := lastToolContent(t, llm); got != "Pedido não encontrado" {
		t.Fatalf("other user: %q", got)
	}

	llm2 := &fakeLLM{responses: []ports.ChatMessage{toolCall("consultar_pedido", uuid.NewString()), {Role: "assistant", Content: "x"}}}
	uc2, userId, _ := setup(llm2, entitiesOrder.OrderStatusShipped, "")
	if _, err := uc2.UseCase(context.Background(), userId, "pedido"); err != nil {
		t.Fatal(err)
	}
	if got := lastToolContent(t, llm2); got != "Pedido não encontrado" {
		t.Fatalf("missing: %q", got)
	}
}

func TestConsultarReembolso(t *testing.T) {
	cases := []struct {
		status, rejection, want string
	}{
		{entitiesOrder.OrderStatusRefundRequested, "", "Em análise (aguardando aprovação)"},
		{entitiesOrder.OrderStatusRefunded, "", "Aprovado e estornado"},
		{entitiesOrder.OrderStatusDelivered, "fora do prazo", "Recusado"},
		{entitiesOrder.OrderStatusDelivered, "", "Não há solicitação de reembolso para este pedido"},
	}
	for _, c := range cases {
		llm := &fakeLLM{}
		uc, userId, orderId := setup(llm, c.status, c.rejection)
		llm.responses = []ports.ChatMessage{toolCall("consultar_reembolso", orderId), {Role: "assistant", Content: "ok"}}
		if _, err := uc.UseCase(context.Background(), userId, "reembolso"); err != nil {
			t.Fatal(err)
		}
		if got := lastToolContent(t, llm); got != c.want {
			t.Fatalf("%s: got %q want %q", c.status, got, c.want)
		}
	}
}

func TestRespostaDiretaSemTool(t *testing.T) {
	llm := &fakeLLM{responses: []ports.ChatMessage{{Role: "assistant", Content: "Acesse o painel Meus Pedidos"}}}
	uc, userId, _ := setup(llm, entitiesOrder.OrderStatusShipped, "")
	reply, err := uc.UseCase(context.Background(), userId, "como solicito reembolso?")
	if err != nil || reply != "Acesse o painel Meus Pedidos" || len(llm.calls) != 1 {
		t.Fatalf("unexpected: %q %v calls=%d", reply, err, len(llm.calls))
	}
}

func TestNaoOfereceToolsForaDeConsultaDePedido(t *testing.T) {
	msgs := []string{
		"oi",
		"quais formas de pagamento vocês aceitam?",
		"vocês têm frete grátis?",
		"qual a capital da França?",
		"como solicito reembolso?",
		"onde vejo o status do meu pedido?",
	}
	for _, m := range msgs {
		llm := &fakeLLM{responses: []ports.ChatMessage{{Role: "assistant", Content: "ok"}}}
		uc, userId, _ := setup(llm, entitiesOrder.OrderStatusShipped, "")
		if _, err := uc.UseCase(context.Background(), userId, m); err != nil {
			t.Fatal(err)
		}
		if llm.toolsSeen[0] != 0 {
			t.Fatalf("%q: tools = %d, want 0", m, llm.toolsSeen[0])
		}
	}
}

func TestOfereceToolsParaConsultaDePedidoEReembolso(t *testing.T) {
	msgs := []string{
		"qual o status do meu pedido?",
		"como está meu pedido?",
		"e meu reembolso?",
		"cadê o rastreio?",
		uuid.NewString(),
	}
	for _, m := range msgs {
		llm := &fakeLLM{responses: []ports.ChatMessage{{Role: "assistant", Content: "ok"}}}
		uc, userId, _ := setup(llm, entitiesOrder.OrderStatusShipped, "")
		if _, err := uc.UseCase(context.Background(), userId, m); err != nil {
			t.Fatal(err)
		}
		if llm.toolsSeen[0] != 2 {
			t.Fatalf("%q: tools = %d, want 2", m, llm.toolsSeen[0])
		}
	}
}

func TestUsaIdDoPedidoInformadoNaMensagemMesmoSeLLMOmitir(t *testing.T) {
	ownerId := uuid.New()
	older := uuid.New()
	newer := uuid.New()
	repo := &fakeOrderRepository{orders: map[string]*entitiesOrder.Order{
		older.String(): {Id: older, Status: entitiesOrder.OrderStatusDelivered, CreateAt: time.Now().Add(-48 * time.Hour), User: &entitiesUser.User{ID: ownerId}},
		newer.String(): {Id: newer, Status: entitiesOrder.OrderStatusPreparing, CreateAt: time.Now(), User: &entitiesUser.User{ID: ownerId}},
	}}
	llm := &fakeLLM{responses: []ports.ChatMessage{toolCall("consultar_pedido", ""), {Role: "assistant", Content: "ok"}}}
	uc := useCasesChat.NewSendMessageUseCase(llm, useCasesOrder.NewReadOrderByIdUseCase(repo), useCasesOrder.NewReadOrdersByUserIdUseCase(repo))

	if _, err := uc.UseCase(context.Background(), ownerId.String(), "status do pedido "+older.String()); err != nil {
		t.Fatal(err)
	}
	if got := lastToolContent(t, llm); got != "Entregue" {
		t.Fatalf("tool content = %q, want Entregue", got)
	}
}

func TestSystemPromptContemInfoDaLojaEGuardrail(t *testing.T) {
	llm := &fakeLLM{responses: []ports.ChatMessage{{Role: "assistant", Content: "ok"}}}
	uc, userId, _ := setup(llm, entitiesOrder.OrderStatusShipped, "")
	if _, err := uc.UseCase(context.Background(), userId, "oi"); err != nil {
		t.Fatal(err)
	}
	sys := llm.calls[0][0]
	if sys.Role != "system" || !strings.Contains(sys.Content, "NERO") || !strings.Contains(sys.Content, "JAMAIS responda qualquer assunto que não seja da loja NERO") {
		t.Fatalf("system prompt sem info da loja/guardrail: %q", sys.Content)
	}
}

func TestLimiteDeIteracoes(t *testing.T) {
	llm := &fakeLLM{}
	uc, userId, orderId := setup(llm, entitiesOrder.OrderStatusShipped, "")
	llm.responses = []ports.ChatMessage{toolCall("consultar_pedido", orderId)}

	if _, err := uc.UseCase(context.Background(), userId, "pedido"); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls) != 4 || llm.toolsSeen[3] != 0 {
		t.Fatalf("calls=%d toolsSeen=%v", len(llm.calls), llm.toolsSeen)
	}
}

func TestHistoricoPorUsuarioELimitado(t *testing.T) {
	llm := &fakeLLM{responses: []ports.ChatMessage{{Role: "assistant", Content: "r"}}}
	uc, userA, _ := setup(llm, entitiesOrder.OrderStatusShipped, "")

	for i := 0; i < 15; i++ {
		if _, err := uc.UseCase(context.Background(), userA, "oi"); err != nil {
			t.Fatal(err)
		}
	}
	last := llm.calls[len(llm.calls)-1]
	if len(last) != 1+20+1 { // system + historico limitado + nova mensagem
		t.Fatalf("len = %d", len(last))
	}

	before := len(llm.calls)
	if _, err := uc.UseCase(context.Background(), uuid.NewString(), "oi"); err != nil {
		t.Fatal(err)
	}
	if len(llm.calls[before]) != 2 {
		t.Fatalf("other user history leaked: %d", len(llm.calls[before]))
	}
}

func TestPropagaToolCallIDParaMensagemTool(t *testing.T) {
	llm := &fakeLLM{}
	uc, userId, orderId := setup(llm, entitiesOrder.OrderStatusShipped, "")
	llm.responses = []ports.ChatMessage{
		{
			Role: "assistant",
			ToolCalls: []ports.ToolCall{{
				ID:       "call_abc_123",
				Function: ports.ToolCallFunction{Name: "consultar_pedido", Arguments: map[string]any{"numero_pedido": orderId}},
			}},
		},
		{Role: "assistant", Content: "ok"},
	}

	if _, err := uc.UseCase(context.Background(), userId, "status do pedido "+orderId); err != nil {
		t.Fatal(err)
	}
	last := llm.calls[len(llm.calls)-1]
	found := false
	for _, msg := range last {
		if msg.Role == "tool" {
			found = true
			if msg.ToolCallID != "call_abc_123" || msg.ToolName != "consultar_pedido" {
				t.Fatalf("tool msg invalida: id=%q name=%q", msg.ToolCallID, msg.ToolName)
			}
		}
	}
	if !found {
		t.Fatal("nenhuma mensagem role=tool enviada ao modelo")
	}
}

func TestErroLLMNaoCorrompeHistorico(t *testing.T) {
	llm := &fakeLLM{responses: []ports.ChatMessage{{Role: "assistant", Content: "r"}}}
	uc, userId, _ := setup(llm, entitiesOrder.OrderStatusShipped, "")
	if _, err := uc.UseCase(context.Background(), userId, "oi"); err != nil {
		t.Fatal(err)
	}

	llm.err = apperrors.ErrServiceUnavailable
	if _, err := uc.UseCase(context.Background(), userId, "falha"); !errors.Is(err, apperrors.ErrServiceUnavailable) {
		t.Fatalf("err = %v", err)
	}

	llm.err = nil
	if _, err := uc.UseCase(context.Background(), userId, "volta"); err != nil {
		t.Fatal(err)
	}
	last := llm.calls[len(llm.calls)-1]
	if len(last) != 1+2+1 {
		t.Fatalf("len = %d", len(last))
	}
}
