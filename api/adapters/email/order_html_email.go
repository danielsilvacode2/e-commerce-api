package email

import (
	"e-commerce/core/ports"
	"fmt"
	"html"
)

func renderOrderEmailHTML(badge, title, subtitle, messageHTML, extraBoxHTML string, orderId string, total float64) string {
	totalFormatted := fmt.Sprintf("R$ %.2f", total)

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>%s — NERO</title>
</head>
<body style="margin:0;padding:0;background-color:#F2F0F1;font-family:'DM Sans',-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;color:#000000;">
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background-color:#F2F0F1;padding:32px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="max-width:560px;background-color:#FFFFFF;border-radius:24px;border:1px solid rgba(0,0,0,0.1);overflow:hidden;">
          <!-- Header NERO -->
          <tr>
            <td style="background-color:#000000;padding:28px 32px;text-align:left;">
              <div style="font-size:24px;font-weight:800;letter-spacing:-0.5px;color:#FFFFFF;text-transform:uppercase;">
                NERO
              </div>
              <div style="margin-top:4px;font-size:11px;letter-spacing:1.8px;color:#A3A3A3;text-transform:uppercase;">
                Moda &amp; Streetwear Moderno
              </div>
            </td>
          </tr>

          <!-- Conteudo -->
          <tr>
            <td style="padding:32px;">
              <span style="display:inline-block;background-color:#F2F0F1;color:#000000;font-size:11px;font-weight:700;letter-spacing:1.2px;text-transform:uppercase;padding:6px 14px;border-radius:999px;">
                %s
              </span>

              <h1 style="margin:16px 0 8px;font-size:24px;font-weight:800;line-height:1.2;letter-spacing:-0.4px;text-transform:uppercase;color:#000000;">
                %s
              </h1>
              <p style="margin:0 0 24px;font-size:14px;line-height:1.6;color:#666666;">
                %s
              </p>

              <div style="background-color:#F2F0F1;border-radius:16px;padding:18px 20px;margin-bottom:20px;">
                <table role="presentation" width="100%%" cellspacing="0" cellpadding="0">
                  <tr>
                    <td style="font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:1px;color:#666666;padding-bottom:6px;">
                      Pedido
                    </td>
                    <td align="right" style="font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:1px;color:#666666;padding-bottom:6px;">
                      Valor Total
                    </td>
                  </tr>
                  <tr>
                    <td style="font-family:'Courier New',monospace;font-size:13px;font-weight:700;color:#000000;">
                      #%s
                    </td>
                    <td align="right" style="font-size:16px;font-weight:800;color:#000000;">
                      %s
                    </td>
                  </tr>
                </table>
              </div>

              <div style="font-size:14px;line-height:1.65;color:#222222;">
                %s
              </div>

              %s
            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="border-top:1px solid rgba(0,0,0,0.08);padding:20px 32px;text-align:center;background-color:#FAFAFA;">
              <p style="margin:0;font-size:11px;letter-spacing:1.4px;text-transform:uppercase;color:#666666;">
                NERO &middot; E-mail transacional autom&aacute;tico, n&atilde;o responda
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
		html.EscapeString(title),
		html.EscapeString(badge),
		html.EscapeString(title),
		html.EscapeString(subtitle),
		html.EscapeString(orderId),
		totalFormatted,
		messageHTML,
		extraBoxHTML,
	)
}

func buildOrderNotificationEmail(event ports.OrderNotificationEvent) (subject string, text string, htmlBody string) {
	userName := html.EscapeString(event.UserName)
	if userName == "" {
		userName = "Cliente"
	}

	switch event.EventType {
	case ports.EventOrderPaymentApproved:
		subject = fmt.Sprintf("NERO — Pagamento aprovado! Pedido #%s em preparação", event.OrderId[:minLen(event.OrderId, 8)])
		text = fmt.Sprintf("Olá %s, seu pagamento do pedido #%s (R$ %.2f) foi aprovado e o pedido já está em preparação.", event.UserName, event.OrderId, event.OrderTotal)
		msg := fmt.Sprintf("<p style=\"margin:0;\">Ol&aacute;, <strong>%s</strong>! Recebemos a confirma&ccedil;&atilde;o do seu pagamento. Nossa equipe j&aacute; est&aacute; separando e preparando seus itens para envio.</p>", userName)
		extra := ""
		if event.PaymentMethod != "" {
			extra = fmt.Sprintf(`<div style="margin-top:16px;border:1px solid rgba(0,0,0,0.1);border-radius:14px;padding:14px 16px;font-size:13px;color:#333333;"><strong>M&eacute;todo de pagamento:</strong> %s</div>`, html.EscapeString(event.PaymentMethod))
		}
		htmlBody = renderOrderEmailHTML("Pagamento Aprovado", "Pedido Confirmado", "Seu pedido entrou em preparação no nosso centro de distribuição.", msg, extra, event.OrderId, event.OrderTotal)

	case ports.EventOrderPaymentFailed:
		subject = fmt.Sprintf("NERO — Pedido #%s cancelado (pagamento não concluído)", event.OrderId[:minLen(event.OrderId, 8)])
		text = fmt.Sprintf("Olá %s, o pagamento do pedido #%s foi recusado ou expirou. O pedido foi cancelado e os itens retornaram ao estoque.", event.UserName, event.OrderId)
		msg := fmt.Sprintf("<p style=\"margin:0;\">Ol&aacute;, <strong>%s</strong>. N&atilde;o foi poss&iacute;vel confirmar o pagamento do seu pedido dentro do prazo ou a transa&ccedil;&atilde;o foi recusada. Por seguran&ccedil;a, o pedido foi cancelado e os itens voltaram ao estoque.</p>", userName)
		extra := `<div style="margin-top:16px;border:1px solid rgba(0,0,0,0.1);border-radius:14px;padding:14px 16px;font-size:13px;color:#333333;">Voc&ecirc; pode refazer sua compra a qualquer momento diretamente na loja.</div>`
		htmlBody = renderOrderEmailHTML("Pagamento Não Concluído", "Pedido Cancelado", "A transação foi recusada ou expirou.", msg, extra, event.OrderId, event.OrderTotal)

	case ports.EventOrderShipped:
		subject = fmt.Sprintf("NERO — Seu pedido #%s foi enviado!", event.OrderId[:minLen(event.OrderId, 8)])
		text = fmt.Sprintf("Olá %s, seu pedido #%s foi enviado e está a caminho!", event.UserName, event.OrderId)
		if event.TrackingCode != "" {
			text += " Código de rastreio: " + event.TrackingCode
		}
		msg := fmt.Sprintf("<p style=\"margin:0;\">Ol&aacute;, <strong>%s</strong>! Boas not&iacute;cias: seu pedido j&aacute; saiu do nosso centro de distribui&ccedil;&atilde;o e est&aacute; a caminho do endere&ccedil;o informado.</p>", userName)
		extra := ""
		if event.TrackingCode != "" {
			extra = fmt.Sprintf(`<div style="margin-top:16px;background-color:#000000;color:#FFFFFF;border-radius:14px;padding:16px 20px;"><div style="font-size:10px;text-transform:uppercase;letter-spacing:1.5px;color:#A3A3A3;">C&oacute;digo de Rastreio</div><div style="margin-top:4px;font-family:'Courier New',monospace;font-size:16px;font-weight:700;letter-spacing:1px;">%s</div></div>`, html.EscapeString(event.TrackingCode))
		}
		htmlBody = renderOrderEmailHTML("Em Trânsito", "Pedido Enviado", "Seu pedido está a caminho.", msg, extra, event.OrderId, event.OrderTotal)

	case ports.EventOrderDelivered:
		subject = fmt.Sprintf("NERO — Pedido #%s entregue com sucesso!", event.OrderId[:minLen(event.OrderId, 8)])
		text = fmt.Sprintf("Olá %s, seu pedido #%s foi entregue com sucesso! Obrigado por comprar na NERO.", event.UserName, event.OrderId)
		msg := fmt.Sprintf("<p style=\"margin:0;\">Ol&aacute;, <strong>%s</strong>! Confirmamos que seu pedido foi entregue. Esperamos que voc&ecirc; curta suas pe&ccedil;as!</p>", userName)
		extra := `<div style="margin-top:16px;border:1px solid rgba(0,0,0,0.1);border-radius:14px;padding:14px 16px;font-size:13px;color:#333333;">Caso precise de ajuda ou devolu&ccedil;&atilde;o, acesse <strong>Meus Pedidos</strong> na sua conta NERO.</div>`
		htmlBody = renderOrderEmailHTML("Entregue", "Pedido Entregue", "Sua encomenda chegou ao destino.", msg, extra, event.OrderId, event.OrderTotal)

	case ports.EventOrderRefundApproved:
		subject = fmt.Sprintf("NERO — Estorno aprovado para o pedido #%s", event.OrderId[:minLen(event.OrderId, 8)])
		text = fmt.Sprintf("Olá %s, o estorno do pedido #%s (R$ %.2f) foi aprovado e concluído.", event.UserName, event.OrderId, event.OrderTotal)
		msg := fmt.Sprintf("<p style=\"margin:0;\">Ol&aacute;, <strong>%s</strong>! Seu estorno foi autorizado e processado com sucesso. O valor ser&aacute; creditado de acordo com o m&eacute;todo de pagamento utilizado na compra.</p>", userName)
		extra := ""
		if event.RefundReason != "" {
			extra = fmt.Sprintf(`<div style="margin-top:16px;border:1px solid rgba(0,0,0,0.1);border-radius:14px;padding:14px 16px;font-size:13px;color:#333333;"><strong>Motivo registrado:</strong> %s</div>`, html.EscapeString(event.RefundReason))
		}
		htmlBody = renderOrderEmailHTML("Estorno Concluído", "Estorno Aprovado", "O reembolso do seu pedido foi efetivado.", msg, extra, event.OrderId, event.OrderTotal)

	case ports.EventOrderRefundRejected:
		subject = fmt.Sprintf("NERO — Atualização sobre o estorno do pedido #%s", event.OrderId[:minLen(event.OrderId, 8)])
		text = fmt.Sprintf("Olá %s, sua solicitação de estorno do pedido #%s foi recusada. Motivo: %s", event.UserName, event.OrderId, event.RejectionReason)
		msg := fmt.Sprintf("<p style=\"margin:0;\">Ol&aacute;, <strong>%s</strong>. Analisamos sua solicita&ccedil;&atilde;o de estorno para este pedido e, no momento, ela n&atilde;o p&ocirc;de ser aprovada. O pedido retornou ao status <strong>Entregue</strong>.</p>", userName)
		extra := ""
		if event.RejectionReason != "" {
			extra = fmt.Sprintf(`<div style="margin-top:16px;border:1px solid rgba(0,0,0,0.15);background-color:#F2F0F1;border-radius:14px;padding:14px 16px;font-size:13px;color:#000000;"><strong>Justificativa:</strong> %s</div>`, html.EscapeString(event.RejectionReason))
		}
		htmlBody = renderOrderEmailHTML("Estorno Recusado", "Solicitação Não Aprovada", "Confira os detalhes da análise do seu pedido.", msg, extra, event.OrderId, event.OrderTotal)

	default:
		subject = fmt.Sprintf("NERO — Atualização do pedido #%s", event.OrderId[:minLen(event.OrderId, 8)])
		text = fmt.Sprintf("Olá %s, há uma nova atualização no seu pedido #%s.", event.UserName, event.OrderId)
		msg := fmt.Sprintf("<p style=\"margin:0;\">Ol&aacute;, <strong>%s</strong>! Seu pedido recebeu uma nova atualiza&ccedil;&atilde;o.</p>", userName)
		htmlBody = renderOrderEmailHTML("Atualização", "Status Atualizado", "Confira os detalhes do seu pedido.", msg, "", event.OrderId, event.OrderTotal)
	}

	return subject, text, htmlBody
}

func minLen(s string, max int) int {
	if len(s) < max {
		return len(s)
	}
	return max
}
