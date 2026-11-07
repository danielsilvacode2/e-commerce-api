package email

const accountExistsHTML = `<div style="background:#121212;padding:32px 16px;font-family:Arial,sans-serif;">
  <div style="max-width:420px;margin:0 auto;">
    <div style="border-bottom:1px solid #333;padding-bottom:16px;">
      <div style="font-family:'Barlow Condensed',Impact,'Arial Narrow',Arial,sans-serif;font-size:28px;font-weight:bold;text-transform:uppercase;letter-spacing:1px;color:#eaeaea;">
        Você já tem uma conta
      </div>
      <div style="margin-top:6px;font-size:11px;text-transform:uppercase;letter-spacing:1.5px;color:#a3a3a3;">
        Alerta de segurança
      </div>
    </div>

    <div style="margin-top:24px;background:#1c1c1c;border:1px solid #333;border-left:3px solid #ff6a00;padding:20px;">
      <p style="margin:0 0 16px;color:#a3a3a3;font-size:13px;line-height:1.6;">
        Recebemos uma tentativa de cadastro usando este e-mail, mas ele já está vinculado a uma conta existente. Se foi você, basta fazer login normalmente.
      </p>
      <p style="margin:0;color:#eaeaea;font-size:13px;line-height:1.6;">
        <strong style="color:#ff6a00;">Não foi você?</strong> Pode ignorar este e-mail. Sua conta continua segura e nenhuma alteração foi feita.
      </p>
    </div>

    <div style="margin-top:24px;border-top:1px solid #333;padding-top:16px;text-align:center;font-size:10px;text-transform:uppercase;letter-spacing:2px;color:#737373;">
      NERO · Mensagem automática, não responda
    </div>
  </div>
</div>`

const verificationHTML = `<div style="background:#121212;padding:32px 16px;font-family:Arial,sans-serif;">
  <div style="max-width:420px;margin:0 auto;background:#1c1c1c;border:1px solid #333;padding:28px;text-align:center;">
    <h2 style="margin:0 0 8px;color:#eaeaea;text-transform:uppercase;letter-spacing:1px;">Verificar e-mail</h2>
    <p style="margin:0 0 20px;color:#a3a3a3;font-size:13px;">Use o código abaixo para confirmar sua conta</p>
    <div style="display:inline-block;border:1px solid #ff6a00;background:#161616;color:#ff6a00;font-family:'Courier New',monospace;font-size:32px;font-weight:bold;letter-spacing:8px;padding:12px 20px;">%s</div>
    <p style="margin:20px 0 0;color:#a3a3a3;font-size:12px;">Nunca compartilhe este código. Se você não solicitou, ignore este e-mail.</p>
  </div>
</div>`
