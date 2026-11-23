package email

const accountExistsHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Alerta de Segurança — NERO</title>
</head>
<body style="margin:0;padding:0;background-color:#F2F0F1;font-family:'DM Sans',-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;color:#000000;">
  <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background-color:#F2F0F1;padding:32px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:560px;background-color:#FFFFFF;border-radius:24px;border:1px solid rgba(0,0,0,0.1);overflow:hidden;">
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
                Alerta de Seguran&ccedil;a
              </span>

              <h1 style="margin:16px 0 8px;font-size:24px;font-weight:800;line-height:1.2;letter-spacing:-0.4px;text-transform:uppercase;color:#000000;">
                Voc&ecirc; j&aacute; tem uma conta
              </h1>
              <p style="margin:0 0 24px;font-size:14px;line-height:1.6;color:#666666;">
                Detectamos uma tentativa de cadastro com o seu endere&ccedil;o de e-mail.
              </p>

              <div style="background-color:#F2F0F1;border-left:3px solid #000000;border-radius:14px;padding:18px 20px;margin-bottom:20px;">
                <p style="margin:0 0 12px;font-size:14px;line-height:1.6;color:#222222;">
                  Recebemos uma tentativa de cadastro usando este e-mail, mas ele j&aacute; est&aacute; vinculado a uma conta existente na <strong>NERO</strong>. Se foi voc&ecirc;, basta fazer login normalmente.
                </p>
                <p style="margin:0;font-size:13px;line-height:1.6;color:#000000;">
                  <strong>N&atilde;o foi voc&ecirc;?</strong> Pode ignorar este e-mail. Sua conta continua segura e nenhuma altera&ccedil;&atilde;o foi feita.
                </p>
              </div>
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
</html>`

const verificationHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>C&oacute;digo de Verifica&ccedil;&atilde;o — NERO</title>
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
            <td style="padding:32px;text-align:center;">
              <span style="display:inline-block;background-color:#F2F0F1;color:#000000;font-size:11px;font-weight:700;letter-spacing:1.2px;text-transform:uppercase;padding:6px 14px;border-radius:999px;">
                Verifica&ccedil;&atilde;o de Seguran&ccedil;a
              </span>

              <h1 style="margin:16px 0 8px;font-size:24px;font-weight:800;line-height:1.2;letter-spacing:-0.4px;text-transform:uppercase;color:#000000;">
                %s
              </h1>
              <p style="margin:0 0 24px;font-size:14px;line-height:1.6;color:#666666;">
                Use o c&oacute;digo abaixo para confirmar sua solicita&ccedil;&atilde;o na NERO.
              </p>

              <div style="display:inline-block;background-color:#000000;color:#FFFFFF;border-radius:16px;font-family:'Courier New',monospace;font-size:32px;font-weight:800;letter-spacing:8px;padding:16px 28px;">
                %s
              </div>

              <div style="margin-top:24px;background-color:#F2F0F1;border-radius:14px;padding:14px 16px;font-size:12px;line-height:1.5;color:#525252;">
                Nunca compartilhe este c&oacute;digo com ningu&eacute;m. Se voc&ecirc; n&atilde;o solicitou, basta ignorar este e-mail.
              </div>
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
</html>`
