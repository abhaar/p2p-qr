package api

import (
	"net/http"
)

// handleGetAddresses returns all custody addresses as a JSON array.
func (s *Server) handleGetAddresses(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	addresses, err := s.custody.GetAddresses(r.Context())
	if err != nil {
		s.logger.Error("failed to get addresses")
		s.writeError(w, http.StatusInternalServerError, "failed to get custody addresses")
		return
	}

	s.writeJSON(w, http.StatusOK, addresses)
}

// handleServeDashboard serves the self-contained frontend app.
func (s *Server) handleServeDashboard(w http.ResponseWriter, r *http.Request) {
	// Only serve dashboard at the root path "/"
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(dashboardHTML)); err != nil {
		s.logger.Error("failed to write dashboard response")
	}
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>P2P Stablecoin Payments Client Terminal</title>
    <!-- Google Fonts Outfit & JetBrains Mono -->
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;700&family=Outfit:wght@300;400;500;600;700;800&display=swap" rel="stylesheet">

    <style>
        :root {
            --bg-color: #080c14;
            --card-bg: rgba(13, 20, 35, 0.7);
            --card-border: rgba(255, 255, 255, 0.06);
            --accent-primary: #3b82f6;
            --accent-primary-glow: rgba(59, 130, 246, 0.35);
            --accent-secondary: #10b981;
            --accent-secondary-glow: rgba(16, 185, 129, 0.35);
            --text-primary: #f8fafc;
            --text-secondary: #94a3b8;
            --text-muted: #64748b;
            --danger-color: #ef4444;
            --danger-glow: rgba(239, 68, 68, 0.2);
            --input-bg: rgba(2, 6, 12, 0.6);
            --input-border: rgba(255, 255, 255, 0.1);
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Outfit', sans-serif;
            background-color: var(--bg-color);
            background-image: 
                radial-gradient(circle at 10% 20%, rgba(59, 130, 246, 0.08) 0%, transparent 40%),
                radial-gradient(circle at 90% 80%, rgba(16, 185, 129, 0.08) 0%, transparent 40%);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            overflow-x: hidden;
        }

        header {
            padding: 24px 40px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-bottom: 1px solid rgba(255, 255, 255, 0.05);
            backdrop-filter: blur(12px);
            position: sticky;
            top: 0;
            z-index: 100;
        }

        .logo-group {
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .logo-icon {
            width: 36px;
            height: 36px;
            background: linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%);
            border-radius: 10px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 800;
            color: white;
            box-shadow: 0 4px 12px var(--accent-primary-glow);
        }

        .logo-text h1 {
            font-size: 20px;
            font-weight: 800;
            letter-spacing: -0.5px;
            background: linear-gradient(135deg, #fff 40%, #cbd5e1 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .logo-text p {
            font-size: 11px;
            color: var(--accent-secondary);
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 1px;
        }

        .connection-status {
            display: flex;
            align-items: center;
            gap: 8px;
            background: rgba(16, 185, 129, 0.1);
            border: 1px solid rgba(16, 185, 129, 0.2);
            padding: 6px 14px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 600;
            color: #34d399;
        }

        .status-dot {
            width: 8px;
            height: 8px;
            background-color: #10b981;
            border-radius: 50%;
            box-shadow: 0 0 8px #10b981;
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7); }
            70% { transform: scale(1); box-shadow: 0 0 0 6px rgba(16, 185, 129, 0); }
            100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
        }

        main {
            flex: 1;
            max-width: 1200px;
            width: 100%;
            margin: 40px auto;
            padding: 0 24px;
            display: grid;
            grid-template-columns: 1fr 2fr;
            gap: 32px;
        }

        @media (max-width: 960px) {
            main {
                grid-template-columns: 1fr;
            }
        }

        /* Common Glass Card styling */
        .glass-card {
            background: var(--card-bg);
            backdrop-filter: blur(16px);
            border: 1px solid var(--card-border);
            border-radius: 24px;
            padding: 30px;
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
            display: flex;
            flex-direction: column;
            gap: 24px;
            transition: all 0.3s ease;
        }

        .glass-card:hover {
            border-color: rgba(255, 255, 255, 0.1);
        }

        .card-header h2 {
            font-size: 20px;
            font-weight: 700;
            letter-spacing: -0.5px;
            display: flex;
            align-items: center;
            gap: 10px;
        }

        /* Wallet Section */
        .wallet-selector-group {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }

        label {
            font-size: 13px;
            font-weight: 600;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        select, input, textarea {
            width: 100%;
            background: var(--input-bg);
            border: 1px solid var(--input-border);
            border-radius: 14px;
            padding: 14px 16px;
            color: var(--text-primary);
            font-family: inherit;
            font-size: 15px;
            outline: none;
            transition: all 0.2s ease;
        }

        select:focus, input:focus, textarea:focus {
            border-color: var(--accent-primary);
            box-shadow: 0 0 10px rgba(59, 130, 246, 0.2);
            background: rgba(2, 6, 12, 0.8);
        }

        .wallet-details {
            background: linear-gradient(135deg, rgba(59, 130, 246, 0.1) 0%, rgba(16, 185, 129, 0.05) 100%);
            border: 1px solid rgba(255, 255, 255, 0.05);
            border-radius: 18px;
            padding: 24px;
            position: relative;
            overflow: hidden;
            display: flex;
            flex-direction: column;
            gap: 16px;
        }

        .wallet-details::after {
            content: '';
            position: absolute;
            top: -50%;
            right: -50%;
            width: 150px;
            height: 150px;
            background: var(--accent-primary-glow);
            filter: blur(50px);
            border-radius: 50%;
        }

        .balance-label {
            font-size: 12px;
            font-weight: 600;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 1px;
        }

        .balance-amount {
            font-size: 38px;
            font-weight: 800;
            letter-spacing: -1px;
            display: flex;
            align-items: baseline;
            gap: 6px;
            background: linear-gradient(135deg, #ffffff 0%, #94a3b8 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .balance-currency {
            font-size: 18px;
            font-weight: 600;
            color: var(--accent-secondary);
            -webkit-text-fill-color: initial;
        }

        .address-hash {
            font-family: 'JetBrains Mono', monospace;
            font-size: 13px;
            color: var(--text-secondary);
            background: rgba(0, 0, 0, 0.2);
            padding: 6px 12px;
            border-radius: 8px;
            word-break: break-all;
            display: flex;
            align-items: center;
            justify-content: space-between;
            gap: 8px;
        }

        .btn-mini-copy {
            background: none;
            border: none;
            color: var(--text-muted);
            cursor: pointer;
            padding: 2px;
            transition: color 0.2s ease;
        }

        .btn-mini-copy:hover {
            color: var(--text-primary);
        }

        /* Tabs Interface */
        .tabs-header {
            display: flex;
            border-bottom: 1px solid rgba(255, 255, 255, 0.05);
            gap: 8px;
        }

        .tab-btn {
            background: none;
            border: none;
            padding: 12px 24px;
            color: var(--text-secondary);
            font-family: inherit;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s ease;
            border-bottom: 2px solid transparent;
            position: relative;
        }

        .tab-btn.active {
            color: var(--text-primary);
            border-bottom-color: var(--accent-primary);
        }

        .tab-content {
            display: none;
            flex-direction: column;
            gap: 24px;
            animation: fadeIn 0.4s ease;
        }

        .tab-content.active {
            display: flex;
        }

        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(8px); }
            to { opacity: 1; transform: translateY(0); }
        }

        /* Action Buttons */
        .btn-action {
            width: 100%;
            background: linear-gradient(135deg, var(--accent-primary) 0%, #2563eb 100%);
            color: white;
            border: none;
            border-radius: 14px;
            padding: 16px 24px;
            font-family: inherit;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            box-shadow: 0 4px 15px rgba(37, 99, 235, 0.3);
            transition: all 0.2s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }

        .btn-action:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 24px rgba(37, 99, 235, 0.5), 0 0 15px var(--accent-primary-glow);
            filter: brightness(1.05);
        }

        .btn-action:active {
            transform: translateY(1px);
        }

        .btn-action.sec {
            background: linear-gradient(135deg, var(--accent-secondary) 0%, #059669 100%);
            box-shadow: 0 4px 15px rgba(5, 150, 105, 0.3);
        }

        .btn-action.sec:hover {
            box-shadow: 0 8px 24px rgba(5, 150, 105, 0.5), 0 0 15px var(--accent-secondary-glow);
        }

        /* Loader */
        .loader {
            width: 20px;
            height: 20px;
            border: 3px solid rgba(255, 255, 255, 0.3);
            border-radius: 50%;
            border-top-color: white;
            animation: spin 1s ease-in-out infinite;
            display: none;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        /* Form Layout */
        .form-row {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 16px;
        }

        @media (max-width: 600px) {
            .form-row {
                grid-template-columns: 1fr;
            }
        }

        .form-group {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }

        /* QR Code Output styling */
        .qr-output-container {
            display: grid;
            grid-template-columns: 200px 1fr;
            gap: 24px;
            background: rgba(0, 0, 0, 0.25);
            border: 1px dashed rgba(255, 255, 255, 0.08);
            border-radius: 18px;
            padding: 24px;
            margin-top: 8px;
        }

        @media (max-width: 600px) {
            .qr-output-container {
                grid-template-columns: 1fr;
                justify-items: center;
                text-align: center;
            }
        }

        .qr-canvas-wrapper {
            background: white;
            padding: 12px;
            border-radius: 12px;
            width: 200px;
            height: 200px;
            display: flex;
            align-items: center;
            justify-content: center;
            box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3);
            cursor: pointer;
            transition: transform 0.2s ease;
            position: relative;
        }

        .qr-canvas-wrapper:hover {
            transform: scale(1.02);
        }

        .qr-canvas-wrapper canvas {
            max-width: 100%;
            max-height: 100%;
        }

        .qr-details {
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            gap: 16px;
        }

        .qr-details-title {
            font-size: 16px;
            font-weight: 700;
            color: var(--accent-secondary);
        }

        .raw-payload-box {
            font-family: 'JetBrains Mono', monospace;
            font-size: 12px;
            background: rgba(0, 0, 0, 0.4);
            border: 1px solid var(--input-border);
            padding: 12px;
            border-radius: 10px;
            word-break: break-all;
            max-height: 80px;
            overflow-y: auto;
            color: var(--text-secondary);
        }

        /* Preview Transaction panel */
        .preview-box {
            background: rgba(16, 185, 129, 0.04);
            border: 1px solid rgba(16, 185, 129, 0.15);
            border-radius: 16px;
            padding: 20px;
            display: flex;
            flex-direction: column;
            gap: 14px;
        }

        .preview-header {
            font-size: 14px;
            font-weight: 700;
            color: var(--accent-secondary);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            display: flex;
            align-items: center;
            gap: 8px;
            border-bottom: 1px solid rgba(16, 185, 129, 0.15);
            padding-bottom: 8px;
        }

        .preview-grid {
            display: grid;
            grid-template-columns: auto 1fr;
            gap: 10px 16px;
            font-size: 14px;
        }

        .preview-label {
            color: var(--text-secondary);
            font-weight: 600;
        }

        .preview-value {
            font-family: 'JetBrains Mono', monospace;
            color: var(--text-primary);
            word-break: break-all;
        }

        .preview-value.amount {
            font-family: inherit;
            font-size: 18px;
            font-weight: 700;
            color: var(--accent-secondary);
        }

        /* Receipt alert box */
        .receipt-card {
            background: rgba(59, 130, 246, 0.05);
            border: 1px solid rgba(59, 130, 246, 0.2);
            border-radius: 16px;
            padding: 20px;
            display: flex;
            flex-direction: column;
            gap: 12px;
            animation: fadeIn 0.4s ease;
        }

        .receipt-card.error {
            background: rgba(239, 68, 68, 0.05);
            border-color: rgba(239, 68, 68, 0.25);
        }

        .receipt-header {
            font-size: 15px;
            font-weight: 700;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .receipt-header.success { color: var(--accent-secondary); }
        .receipt-header.error { color: var(--danger-color); }

        .receipt-hash {
            font-family: 'JetBrains Mono', monospace;
            font-size: 12px;
            background: rgba(0, 0, 0, 0.3);
            padding: 8px 12px;
            border-radius: 8px;
            word-break: break-all;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }

        /* Toast notification */
        .toast {
            position: fixed;
            bottom: -80px;
            left: 50%;
            transform: translateX(-50%);
            background: var(--accent-secondary);
            color: white;
            padding: 14px 28px;
            border-radius: 30px;
            font-weight: 600;
            font-size: 14px;
            box-shadow: 0 10px 25px rgba(16, 185, 129, 0.4);
            display: flex;
            align-items: center;
            gap: 10px;
            transition: bottom 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            z-index: 1000;
        }

        .toast.show {
            bottom: 40px;
        }

        .toast.error {
            background: var(--danger-color);
            box-shadow: 0 10px 25px var(--danger-glow);
        }
    </style>
</head>
<body>

    <header>
        <div class="logo-group">
            <div class="logo-icon">&Delta;</div>
            <div class="logo-text">
                <h1>P2P Stablecoin Payments</h1>
                <p>Client Terminal</p>
            </div>
        </div>
        <div class="connection-status">
            <div class="status-dot"></div>
            Connected to Blockchain Node
        </div>
    </header>

    <main>
        <!-- Left Column: wallet selector and balances -->
        <section class="glass-card">
            <div class="card-header">
                <h2>
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="4" width="20" height="16" rx="2" ry="2"></rect><line x1="12" y1="4" x2="12" y2="20"></line><line x1="2" y1="12" x2="22" y2="12"></line></svg>
                    Wallet Registry
                </h2>
            </div>

            <div class="wallet-selector-group">
                <label for="walletSelector">Active Custody Wallet</label>
                <select id="walletSelector">
                    <option value="">Loading wallets...</option>
                </select>
            </div>

            <div class="wallet-details">
                <div class="balance-label">Available Balance</div>
                <div class="balance-amount" id="balanceDisplay">
                    -- <span class="balance-currency">USDC</span>
                </div>
                <div class="address-hash">
                    <span id="activeAddressDisplay">0x...</span>
                    <button class="btn-mini-copy" id="btnCopyActiveAddr" title="Copy Wallet Address">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                    </button>
                </div>
            </div>

            <button class="btn-action" id="btnRefreshBalance" style="margin-top: 8px;">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"></polyline><polyline points="1 20 1 14 7 14"></polyline><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path></svg>
                Sync Ledger Balance
            </button>
        </section>

        <!-- Right Column: Tabs (Generate QR, Pay QR) -->
        <section class="glass-card">
            <div class="tabs-header">
                <button class="tab-btn active" data-tab="tab-generate">Request Payment (QR)</button>
                <button class="tab-btn" data-tab="tab-pay">Pay / Execute QR</button>
            </div>

            <!-- Tab 1: Generate Payment QR -->
            <div class="tab-content active" id="tab-generate">
                <div class="form-group">
                    <label for="genRecipient">Recipient Address</label>
                    <select id="genRecipient">
                        <!-- Populated dynamically -->
                    </select>
                </div>

                <div class="form-row">
                    <div class="form-group">
                        <label for="genAmount">Amount (USDC)</label>
                        <input type="number" id="genAmount" placeholder="e.g. 15.00" min="0.000001" step="any" required>
                    </div>
                    <div class="form-group">
                        <label for="genMemo">Memo / Description</label>
                        <input type="text" id="genMemo" placeholder="e.g. dinner bill" maxlength="25">
                    </div>
                </div>

                <button class="btn-action sec" id="btnGenerate">
                    <span class="loader" id="genLoader"></span>
                    <span id="genBtnText">Generate Payment Request</span>
                </button>

                <!-- QR Output -->
                <div class="qr-output-container" id="qrOutputContainer" style="display: none;">
                    <div class="qr-canvas-wrapper" id="qrCanvasWrapper" title="Click to copy raw payload">
                        <img id="qrImg" style="max-width: 100%; max-height: 100%; border-radius: 8px;">
                    </div>
                    <div class="qr-details">
                        <div>
                            <div class="qr-details-title">X9.150 Compliant QR Code</div>
                            <p style="font-size: 13px; color: var(--text-secondary); margin-top: 4px;">Click the QR code to copy the raw payload or copy it from below.</p>
                        </div>
                        <div class="raw-payload-box" id="rawPayloadText"></div>
                        <button class="btn-action" id="btnCopyPayload" style="padding: 10px 16px; font-size: 13px; border-radius: 8px;">
                            Copy Raw Payload
                        </button>
                    </div>
                </div>
            </div>

            <!-- Tab 2: Pay QR Code -->
            <div class="tab-content" id="tab-pay">
                <div class="form-group">
                    <label for="paySender">Sender Account (From)</label>
                    <select id="paySender">
                        <!-- Populated dynamically -->
                    </select>
                </div>

                <div class="form-group">
                    <label for="payPayload">Raw QR Payload</label>
                    <textarea id="payPayload" placeholder="Paste the EMVCo compliant TLV payload here..." rows="4"></textarea>
                </div>

                <!-- Live Transaction Preview -->
                <div class="preview-box" id="previewContainer" style="display: none;">
                    <div class="preview-header">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
                        Transaction Preview
                    </div>
                    <div class="preview-grid">
                        <div class="preview-label">Recipient:</div>
                        <div class="preview-value" id="previewRecipient">0x...</div>

                        <div class="preview-label">Amount:</div>
                        <div class="preview-value amount" id="previewAmount">0.00 USDC</div>

                        <div class="preview-label">Network:</div>
                        <div class="preview-value" id="previewNetwork">ethereum</div>

                        <div class="preview-label">Memo:</div>
                        <div class="preview-value" id="previewMemo">N/A</div>
                    </div>
                </div>

                <button class="btn-action" id="btnPay" disabled>
                    <span class="loader" id="payLoader"></span>
                    <span id="payBtnText">Execute QR Transfer</span>
                </button>

                <!-- Receipt Output -->
                <div id="receiptContainer"></div>
            </div>
        </section>
    </main>

    <!-- Toast Component -->
    <div class="toast" id="toast">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
        <span id="toastMessage">Action successful</span>
    </div>

    <script>
        // Global State
        let walletsList = [];
        let activeAddress = "";
        let generatedPayload = "";

        // Elements
        const walletSelector = document.getElementById("walletSelector");
        const balanceDisplay = document.getElementById("balanceDisplay");
        const activeAddressDisplay = document.getElementById("activeAddressDisplay");
        const btnRefreshBalance = document.getElementById("btnRefreshBalance");
        const btnCopyActiveAddr = document.getElementById("btnCopyActiveAddr");

        // Tab Buttons & Contents
        const tabButtons = document.querySelectorAll(".tab-btn");
        const tabContents = document.querySelectorAll(".tab-content");

        // Tab: Generate Elements
        const genRecipient = document.getElementById("genRecipient");
        const genAmount = document.getElementById("genAmount");
        const genMemo = document.getElementById("genMemo");
        const btnGenerate = document.getElementById("btnGenerate");
        const genLoader = document.getElementById("genLoader");
        const genBtnText = document.getElementById("genBtnText");
        const qrOutputContainer = document.getElementById("qrOutputContainer");
        const qrImg = document.getElementById("qrImg");
        const qrCanvasWrapper = document.getElementById("qrCanvasWrapper");
        const rawPayloadText = document.getElementById("rawPayloadText");
        const btnCopyPayload = document.getElementById("btnCopyPayload");

        // Tab: Pay Elements
        const paySender = document.getElementById("paySender");
        const payPayload = document.getElementById("payPayload");
        const btnPay = document.getElementById("btnPay");
        const payLoader = document.getElementById("payLoader");
        const payBtnText = document.getElementById("payBtnText");
        const previewContainer = document.getElementById("previewContainer");
        const previewRecipient = document.getElementById("previewRecipient");
        const previewAmount = document.getElementById("previewAmount");
        const previewNetwork = document.getElementById("previewNetwork");
        const previewMemo = document.getElementById("previewMemo");
        const receiptContainer = document.getElementById("receiptContainer");

        // Toast
        const toast = document.getElementById("toast");
        const toastMessage = document.getElementById("toastMessage");

        // Toast Helper
        function showToast(message, isError) {
            toastMessage.textContent = message;
            if (isError) {
                toast.classList.add("error");
            } else {
                toast.classList.remove("error");
            }
            toast.classList.add("show");
            setTimeout(function() {
                toast.classList.remove("show");
            }, 3000);
        }

        // Copy Helper
        function copyText(text, successMsg) {
            if (!successMsg) successMsg = "Copied to clipboard!";
            navigator.clipboard.writeText(text).then(function() {
                showToast(successMsg, false);
            }).catch(function(err) {
                console.error("Copy failed: ", err);
                showToast("Copy failed", true);
            });
        }

        // Tab Navigation
        tabButtons.forEach(function(btn) {
            btn.addEventListener("click", function() {
                tabButtons.forEach(function(b) { b.classList.remove("active"); });
                tabContents.forEach(function(c) { c.classList.remove("active"); });

                btn.classList.add("active");
                const targetTab = btn.getAttribute("data-tab");
                document.getElementById(targetTab).classList.add("active");
            });
        });

        // Initialize App
        async function init() {
            try {
                const response = await fetch("/addresses");
                if (!response.ok) throw new Error("Failed to load wallets");
                walletsList = await response.json();

                if (walletsList.length === 0) {
                    showToast("No wallets returned from custody registry", true);
                    return;
                }

                // Populate selects
                walletSelector.innerHTML = "";
                genRecipient.innerHTML = "";
                paySender.innerHTML = "";

                walletsList.forEach(function(addr) {
                    const opt = document.createElement("option");
                    opt.value = addr;
                    opt.textContent = addr;
                    
                    walletSelector.appendChild(opt.cloneNode(true));
                    genRecipient.appendChild(opt.cloneNode(true));
                    paySender.appendChild(opt.cloneNode(true));
                });

                // Set default active wallet
                setActiveWallet(walletsList[0]);

            } catch (err) {
                console.error(err);
                showToast("Error connecting to custody API", true);
            }
        }

        function setActiveWallet(address) {
            activeAddress = address;
            activeAddressDisplay.textContent = address;
            walletSelector.value = address;
            refreshBalance();
        }

        async function refreshBalance() {
            if (!activeAddress) return;
            balanceDisplay.innerHTML = "Loading... <span class=\"balance-currency\">USDC</span>";
            
            try {
                const response = await fetch("/balance/" + activeAddress);
                if (!response.ok) throw new Error();
                const data = await response.json();
                
                // Format decimal output
                const balanceVal = parseFloat(data.balance).toLocaleString(undefined, {
                    minimumFractionDigits: 2,
                    maximumFractionDigits: 6
                });
                balanceDisplay.innerHTML = balanceVal + " <span class=\"balance-currency\">USDC</span>";
            } catch (err) {
                balanceDisplay.innerHTML = "Error <span class=\"balance-currency\">USDC</span>";
                showToast("Failed to fetch wallet balance", true);
            }
        }

        // Generate QR Action
        btnGenerate.addEventListener("click", async function() {
            const recipient = genRecipient.value;
            const amount = parseFloat(genAmount.value);
            const memo = genMemo.value.trim();

            if (!recipient || isNaN(amount) || amount <= 0) {
                showToast("Please enter a valid amount", true);
                return;
            }

            // Set loading
            genLoader.style.display = "inline-block";
            genBtnText.textContent = "Generating...";
            btnGenerate.disabled = true;
            qrOutputContainer.style.display = "none";

            try {
                const response = await fetch("/payment-request/qr", {
                    method: "POST",
                    headers: { 
                        "Content-Type": "application/json",
                        "Accept": "application/json"
                    },
                    body: JSON.stringify({
                        address: recipient,
                        amount: amount.toString(),
                        currency: "USDC",
                        network: "ethereum",
                        memo: memo
                    })
                });

                if (!response.ok) throw new Error("Server rejected QR request");
                const data = await response.json();
                generatedPayload = data.payload;

                // Render QR Code using the server-generated base64 image
                qrImg.src = "data:image/png;base64," + data.qr_image_base64;

                rawPayloadText.textContent = data.payload;
                qrOutputContainer.style.display = "grid";
                showToast("Payment request QR generated successfully!", false);

            } catch (err) {
                console.error(err);
                showToast(err.message || "Failed to generate QR payload", true);
            } finally {
                genLoader.style.display = "none";
                genBtnText.textContent = "Generate Payment Request";
                btnGenerate.disabled = false;
            }
        });

        // Parse EMVCo TLV in JavaScript for real-time validation & preview
        function parseEMVCoTLV(payload) {
            if (!payload || payload.length < 4) return null;

            try {
                let data = { address: "", amount: "", currency: "", network: "", memo: "" };
                let i = 0;
                
                while (i < payload.length) {
                    if (i + 4 > payload.length) break;
                    const tag = payload.substring(i, i + 2);
                    const length = parseInt(payload.substring(i + 2, i + 4), 10);
                    i += 4;
                    if (i + length > payload.length) break;
                    const value = payload.substring(i, i + length);
                    i += length;

                    if (tag === "26") { // Merchant Account Information
                        let subIdx = 0;
                        while (subIdx < value.length) {
                            if (subIdx + 4 > value.length) break;
                            const subTag = value.substring(subIdx, subIdx + 2);
                            const subLen = parseInt(value.substring(subIdx + 2, subIdx + 4), 10);
                            subIdx += 4;
                            if (subIdx + subLen > value.length) break;
                            const subVal = value.substring(subIdx, subIdx + subLen);
                            subIdx += subLen;

                            if (subTag === "01") data.address = subVal;
                            if (subTag === "02") data.network = subVal;
                            if (subTag === "03") data.currency = subVal;
                        }
                    } else if (tag === "54") {
                        data.amount = value;
                    } else if (tag === "62") { // Additional Data
                        let subIdx = 0;
                        while (subIdx < value.length) {
                            if (subIdx + 4 > value.length) break;
                            const subTag = value.substring(subIdx, subIdx + 2);
                            const subLen = parseInt(value.substring(subIdx + 2, subIdx + 4), 10);
                            subIdx += 4;
                            if (subIdx + subLen > value.length) break;
                            const subVal = value.substring(subIdx, subIdx + subLen);
                            subIdx += subLen;

                            if (subTag === "05") data.memo = subVal;
                        }
                    }
                }

                if (data.address && data.amount) return data;
            } catch (e) {
                console.error("Error parsing payload", e);
            }
            return null;
        }

        // Pay QR Action - Live Preview Input Listener
        payPayload.addEventListener("input", function() {
            const rawVal = payPayload.value.trim();
            const parsed = parseEMVCoTLV(rawVal);

            if (parsed) {
                previewRecipient.textContent = parsed.address;
                previewAmount.textContent = parseFloat(parsed.amount).toFixed(2) + " " + parsed.currency;
                previewNetwork.textContent = parsed.network;
                previewMemo.textContent = parsed.memo || "N/A";
                previewContainer.style.display = "flex";
                btnPay.disabled = false;
            } else {
                previewContainer.style.display = "none";
                btnPay.disabled = true;
            }
        });

        // Pay QR Execution
        btnPay.addEventListener("click", async function() {
            const sender = paySender.value;
            const payload = payPayload.value.trim();

            if (!sender || !payload) return;

            payLoader.style.display = "inline-block";
            payBtnText.textContent = "Executing...";
            btnPay.disabled = true;
            receiptContainer.innerHTML = "";

            try {
                const response = await fetch("/transfer/qr", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({
                        from: sender,
                        payload: payload
                    })
                });

                const result = await response.json();

                if (!response.ok) {
                    throw new Error(result.error || "Transfer failed");
                }

                // Render success receipt
                receiptContainer.innerHTML = '<div class="receipt-card"><div class="receipt-header success"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>Transfer Broadcasted Successfully</div><p style="font-size: 13px; color: var(--text-secondary);">Transaction Hash:</p><div class="receipt-hash"><span>' + result.tx_hash + '</span><button class="btn-mini-copy" id="btnCopyResultHash"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button></div><p style="font-size: 12px; color: var(--text-muted);">Status: ' + result.status + '</p></div>';

                document.getElementById("btnCopyResultHash").addEventListener("click", function() {
                    copyText(result.tx_hash, "Transaction hash copied!");
                });

                showToast("Transaction submitted successfully!", false);
                payPayload.value = "";
                previewContainer.style.display = "none";
                
                // Refresh balance after transaction (after a brief delay for ledger update)
                setTimeout(refreshBalance, 2000);

            } catch (err) {
                console.error(err);
                receiptContainer.innerHTML = '<div class="receipt-card error"><div class="receipt-header error"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>Execution Failed</div><p style="font-size: 13px; color: var(--text-secondary);">' + err.message + '</p></div>';
                showToast(err.message, true);
            } finally {
                payLoader.style.display = "none";
                payBtnText.textContent = "Execute QR Transfer";
                btnPay.disabled = false;
            }
        });

        // Event Listeners
        walletSelector.addEventListener("change", function(e) {
            setActiveWallet(e.target.value);
        });

        btnRefreshBalance.addEventListener("click", function() {
            refreshBalance();
            showToast("Syncing wallet balance...", false);
        });

        btnCopyActiveAddr.addEventListener("click", function() {
            if (activeAddress) {
                copyText(activeAddress, "Wallet address copied!");
            }
        });

        qrCanvasWrapper.addEventListener("click", function() {
            if (generatedPayload) {
                copyText(generatedPayload, "QR Code raw payload copied!");
            }
        });

        btnCopyPayload.addEventListener("click", function() {
            if (generatedPayload) {
                copyText(generatedPayload, "QR Code raw payload copied!");
            }
        });

        // Run
        window.addEventListener("DOMContentLoaded", init);
    </script>
</body>
</html>
`
