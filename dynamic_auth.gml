html(
    head(
        meta:charset."utf-8"(),
        title("Secure Bootstrap SSO - Secure Login"),
        script:src."/auth/webauthn.js"(),
        style(
            rule("body", "background-color: #000", "font-family: -apple-system, BlinkMacSystemFont, sans-serif", "margin: 0"),
            rule(".auth-wrapper", "display: flex", "height: 100vh", "width: 100vw", "align-items: center", "justify-content: center"),
            rule(".auth-box", "width: 100%", "max-width: 420px", "background: #16181c", "border-radius: 16px", "padding: 40px", "text-align: center", "border: 1px solid #2f3336", "box-sizing: border-box"),
            rule(".auth-logo", "font-size: 36px", "margin-bottom: 20px"),
            rule(".auth-title", "color: white", "margin: 0 0 10px 0", "font-size: 24px"),
            rule(".auth-desc", "color: #71767b", "margin-bottom: 30px", "font-size: 15px", "line-height: 1.5"),
            rule(".auth-input", "width: 100%", "padding: 16px", "margin-bottom: 20px", "border-radius: 8px", "border: 1px solid #333", "background: #000", "color: white", "box-sizing: border-box"),
            rule(".btn-primary", "background: #1d9bf0", "color: white", "border: none", "padding: 16px", "border-radius: 9999px", "cursor: pointer", "width: 100%", "font-size: 16px", "font-weight: bold", "margin-bottom: 15px"),
            rule(".btn-secondary", "background: transparent", "color: #e7e9ea", "border: 1px solid #536471", "padding: 16px", "border-radius: 9999px", "cursor: pointer", "width: 100%", "font-size: 16px", "font-weight: bold")
        )
    ),
    body(
        div.auth-wrapper(
            div.auth-box(
                div.auth-logo("🌐"),
                h2.auth-title("Sign in to Secure Bootstrap SSO"),
                p.auth-desc("Authenticate securely using your device's native Passkey."),
                form:action."javascript:void(0);"(
                    input.auth-input:id."username":name."username":type."text":placeholder."Enter a Username"(),
                    button.btn-primary:type."button":onclick."passkeyLogin(document.getElementById('username').value)"("Sign In with Passkey"),
                    button.btn-secondary:type."button":onclick."passkeyRegister(document.getElementById('username').value)"("Register New Passkey"),
                )
            )
        )
    )
)