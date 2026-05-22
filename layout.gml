html(
    head(
        meta:charset."utf-8"(),
        meta:name."viewport":content."width=device-width, initial-scale=1.0"(),
        
        title("Secure Logger - Telemetry Dashboard"),
        
        style(
            rule("body", "background-color: #0d1117", "color: #c9d1d9", "font-family: -apple-system, BlinkMacSystemFont, sans-serif", "margin: 0", "height: 100vh", "overflow: hidden"),
            rule(".app-wrapper", "display: flex", "height: 100%"),
            
            // Left Sidebar (Navigation / Indices)
            rule(".sidebar", "width: 260px", "background: #010409", "border-right: 1px solid #30363d", "display: flex", "flex-direction: column"),
            rule(".brand-header", "padding: 20px", "border-bottom: 1px solid #30363d", "font-size: 18px", "font-weight: bold", "color: #58a6ff", "display: flex", "align-items: center", "gap: 10px"),
            rule(".nav-menu", "padding: 15px", "list-style: none", "margin: 0", "display: flex", "flex-direction: column", "gap: 8px"),
            rule(".nav-link", "color: #8b949e", "text-decoration: none", "font-size: 14px", "padding: 8px 12px", "border-radius: 6px", "transition: background 0.2s, color 0.2s"),
            rule(".nav-link:hover", "background: #161b22", "color: #c9d1d9"),
            
            // Main Content Area
            rule(".main-feed", "flex: 1", "display: flex", "flex-direction: column", "background: #0d1117"),
            rule(".font-mono", "font-family: ui-monospace, SFMono-Regular, Consolas, 'Courier New', monospace")
        )
    ),
    body(
        div.app-wrapper(
            
            // Sidebar Navigation
            div.sidebar(
                div.brand-header(
                    span:style."background: #238636; width: 12px; height: 12px; border-radius: 50%; display: inline-block;"(),
                    "LogOps Console"
                ),
                ul.nav-menu(
                    li( a.nav-link:href."/"("📡 Live Tail") ),
                    li( a.nav-link:href."/?q=error"("🚨 Error Logs") ),
                    li( a.nav-link:href."/settings"("⚙️ Engine Settings") )
                )
            ),
            
            // Dynamic View Injection (index.gml goes here)
            div.main-feed( slot() )
        )
    )
)
