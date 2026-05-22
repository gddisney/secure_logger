wrapper("views/layout.gml",
    style(
        // Top Search Header
        rule(".search-header", "padding: 15px 25px", "border-bottom: 1px solid #30363d", "background: #161b22", "display: flex", "align-items: center", "gap: 15px"),
        rule(".search-input", "flex: 1", "padding: 12px 18px", "background: #0d1117", "border: 1px solid #30363d", "color: white", "border-radius: 6px", "font-size: 14px", "outline: none", "transition: border-color 0.2s"),
        rule(".search-input:focus", "border-color: #58a6ff"),
        rule(".search-btn", "background: #238636", "color: white", "border: none", "padding: 12px 24px", "border-radius: 6px", "cursor: pointer", "font-weight: bold", "font-size: 14px", "transition: background 0.2s"),
        rule(".search-btn:hover", "background: #2ea043"),
        
        // Log Table Wrapper
        rule(".log-container", "flex: 1", "overflow-y: auto", "padding: 20px", "display: flex", "flex-direction: column", "gap: 6px"),
        
        // Individual Log Row
        rule(".log-row", "display: flex", "align-items: flex-start", "gap: 15px", "padding: 12px 15px", "background: #161b22", "border: 1px solid #21262d", "border-radius: 6px", "font-size: 13px", "transition: background 0.1s"),
        rule(".log-row:hover", "background: #21262d"),
        
        // Data Columns
        rule(".log-time", "color: #8b949e", "width: 170px", "flex-shrink: 0"),
        rule(".log-service", "color: #bc8cff", "width: 140px", "flex-shrink: 0", "font-weight: bold"),
        rule(".log-message", "color: #e6edf3", "flex: 1", "word-break: break-word"),
        
        // Severity Tag Colors
        rule(".log-level", "padding: 2px 8px", "border-radius: 12px", "font-size: 11px", "font-weight: bold", "text-transform: uppercase", "width: 50px", "text-align: center", "flex-shrink: 0"),
        rule(".level-error", "background: rgba(248,81,73,0.1)", "color: #ff7b72", "border: 1px solid rgba(248,81,73,0.4)"),
        rule(".level-warn", "background: rgba(210,153,34,0.1)", "color: #d29922", "border: 1px solid rgba(210,153,34,0.4)"),
        rule(".level-info", "background: rgba(88,166,255,0.1)", "color: #58a6ff", "border: 1px solid rgba(88,166,255,0.4)")
    ),
    
    // The Active Search Form
    form.search-header:action."/":method."GET"(
        input.search-input:name."q":type."text":placeholder."Search logs (e.g., 'level:error AND service:auth')...":value."{{ .Query }}"(),
        button.search-btn:type."submit"("Search Engine")
    ),

    // Log Render Output
    div.log-container.font-mono(
        
        "{{ if .Results }}",
            "{{ range .Results }}",
            div.log-row(
                // Expecting a Go struct like: { Level: "ERROR", LevelClass: "level-error", Time: "2026-05-22 09:41", Service: "auth-svc", Message: "..." }
                div:class."log-level {{ .LevelClass }}"("{{ .Level }}"),
                div.log-time("{{ .Time }}"),
                div.log-service("{{ .Service }}"),
                div.log-message("{{ .Message }}")
            ),
            "{{ end }}",
        "{{ else }}",
            div:style."display: flex; flex-direction: column; align-items: center; justify-content: center; height: 100%; color: #8b949e;"(
                span:style."font-size: 32px; margin-bottom: 10px;"("📭"),
                span("Awaiting query or no matching telemetry found.")
            ),
        "{{ end }}"
    )
)
