package api

import (
	"fmt"
	"strings"
)

// ParsedQRData holds the payment details extracted from an EMVCo QR payload.
type ParsedQRData struct {
	Address  string
	Amount   string
	Currency string
	Network  string
	Memo     string
}

// formatEMVCoTLV formats a Tag-Length-Value field.
func formatEMVCoTLV(tag string, value string) string {
	return fmt.Sprintf("%s%02d%s", tag, len(value), value)
}

// calculateCRC16 calculates the CRC-16/CCITT-FALSE checksum for EMVCo specifications.
func calculateCRC16(data string) string {
	crc := uint16(0xFFFF)
	polynomial := uint16(0x1021)
	for i := 0; i < len(data); i++ {
		crc ^= uint16(data[i]) << 8
		for j := 0; j < 8; j++ {
			if (crc & 0x8000) != 0 {
				crc = (crc << 1) ^ polynomial
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}

// buildEMVCoQRContent builds the X9.150 compliant EMVCo MPM QR payload.
func buildEMVCoQRContent(req X9APaymentRequest) (string, error) {
	var parts []string

	// Tag 00: Payload Format Indicator (mandatory, value = "01")
	parts = append(parts, formatEMVCoTLV("00", "01"))

	// Tag 01: Point of Initiation Method (mandatory, dynamic = "12")
	parts = append(parts, formatEMVCoTLV("01", "12"))

	// Tag 26: Merchant Account Information (mandatory for X9.150)
	var t26Parts []string
	// Sub-tag 00: Globally Unique Identifier (GUI)
	t26Parts = append(t26Parts, formatEMVCoTLV("00", "org.x9.payment"))
	if req.Address != "" {
		t26Parts = append(t26Parts, formatEMVCoTLV("01", req.Address))
	}
	if req.Network != "" {
		t26Parts = append(t26Parts, formatEMVCoTLV("02", req.Network))
	}
	if req.Currency != "" {
		t26Parts = append(t26Parts, formatEMVCoTLV("03", req.Currency))
	}
	t26Value := strings.Join(t26Parts, "")
	parts = append(parts, formatEMVCoTLV("26", t26Value))

	// Tag 52: Merchant Category Code (standard placeholder)
	parts = append(parts, formatEMVCoTLV("52", "0000"))

	// Tag 53: Transaction Currency (ISO 4217, USD = "840")
	parts = append(parts, formatEMVCoTLV("53", "840"))

	// Tag 54: Transaction Amount
	parts = append(parts, formatEMVCoTLV("54", req.Amount))

	// Tag 58: Country Code (ISO 3166)
	parts = append(parts, formatEMVCoTLV("58", "US"))

	// Tag 59: Merchant Name
	parts = append(parts, formatEMVCoTLV("59", "Custody API"))

	// Tag 60: Merchant City
	parts = append(parts, formatEMVCoTLV("60", "New York"))

	// Tag 62: Additional Data Template (optional, for memo)
	if req.Memo != "" {
		var t62Parts []string
		t62Parts = append(t62Parts, formatEMVCoTLV("05", req.Memo))
		t62Value := strings.Join(t62Parts, "")
		parts = append(parts, formatEMVCoTLV("62", t62Value))
	}

	// Tag 63: CRC16 checksum
	payloadStrWithoutCRC := strings.Join(parts, "") + "6304"
	crc := calculateCRC16(payloadStrWithoutCRC)
	return payloadStrWithoutCRC + crc, nil
}

// parseEMVCoTLV parses the X9.150 compliant EMVCo TLV payload and validates the CRC.
func parseEMVCoTLV(payload string) (ParsedQRData, error) {
	var data ParsedQRData

	// Validate CRC16
	if len(payload) < 4 {
		return data, fmt.Errorf("payload too short for CRC verification")
	}
	payloadWithoutCRC := payload[:len(payload)-4]
	expectedCRC := payload[len(payload)-4:]
	calculatedCRC := calculateCRC16(payloadWithoutCRC)
	if !strings.EqualFold(calculatedCRC, expectedCRC) {
		return data, fmt.Errorf("invalid CRC: calculated %s, expected %s", calculatedCRC, expectedCRC)
	}

	i := 0
	for i < len(payload) {
		if i+4 > len(payload) {
			break
		}
		tag := payload[i : i+2]
		lengthStr := payload[i+2 : i+4]
		var length int
		if _, err := fmt.Sscanf(lengthStr, "%d", &length); err != nil {
			return data, fmt.Errorf("invalid length: %s", lengthStr)
		}
		i += 4
		if i+length > len(payload) {
			return data, fmt.Errorf("invalid payload: truncated value for tag %s", tag)
		}
		value := payload[i : i+length]
		i += length

		switch tag {
		case "26": // Merchant Account Template
			subIdx := 0
			for subIdx < len(value) {
				if subIdx+4 > len(value) {
					break
				}
				subTag := value[subIdx : subIdx+2]
				subLenStr := value[subIdx+2 : subIdx+4]
				var subLen int
				if _, err := fmt.Sscanf(subLenStr, "%d", &subLen); err != nil {
					break
				}
				subIdx += 4
				if subIdx+subLen > len(value) {
					break
				}
				subVal := value[subIdx : subIdx+subLen]
				subIdx += subLen

				switch subTag {
				case "01":
					data.Address = subVal
				case "02":
					data.Network = subVal
				case "03":
					data.Currency = subVal
				}
			}
		case "54": // Amount
			data.Amount = value
		case "62": // Additional Data Template
			subIdx := 0
			for subIdx < len(value) {
				if subIdx+4 > len(value) {
					break
				}
				subTag := value[subIdx : subIdx+2]
				subLenStr := value[subIdx+2 : subIdx+4]
				var subLen int
				if _, err := fmt.Sscanf(subLenStr, "%d", &subLen); err != nil {
					break
				}
				subIdx += 4
				if subIdx+subLen > len(value) {
					break
				}
				subVal := value[subIdx : subIdx+subLen]
				subIdx += subLen

				if subTag == "05" {
					data.Memo = subVal
				}
			}
		}
	}
	return data, nil
}

// qrHTMLTemplate is the template for self-contained, interactive payment QR request pages.
const qrHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>X9.150 Payment QR Code Viewer</title>
    <!-- Google Fonts Outfit -->
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;600;800&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-color: #0b0f19;
            --card-bg: rgba(17, 24, 39, 0.7);
            --accent-color: #3b82f6;
            --accent-glow: rgba(59, 130, 246, 0.4);
            --text-primary: #f3f4f6;
            --text-secondary: #9ca3af;
            --success-color: #10b981;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Outfit', sans-serif;
            background: radial-gradient(circle at 50% 50%, #1e293b 0%, var(--bg-color) 100%);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            overflow: hidden;
            position: relative;
        }

        /* Ambient background glow */
        body::before {
            content: '';
            position: absolute;
            width: 300px;
            height: 300px;
            background: var(--accent-glow);
            filter: blur(120px);
            border-radius: 50%;
            top: 20%;
            left: 30%;
            z-index: 0;
        }

        body::after {
            content: '';
            position: absolute;
            width: 250px;
            height: 250px;
            background: rgba(16, 185, 129, 0.2);
            filter: blur(100px);
            border-radius: 50%;
            bottom: 20%;
            right: 30%;
            z-index: 0;
        }

        .container {
            position: relative;
            z-index: 10;
            background: var(--card-bg);
            backdrop-filter: blur(16px);
            -webkit-backdrop-filter: blur(16px);
            border: 1px solid rgba(255, 255, 255, 0.08);
            border-radius: 24px;
            padding: 40px;
            width: 100%;
            max-width: 440px;
            text-align: center;
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
            transition: transform 0.3s ease, box-shadow 0.3s ease;
        }

        .container:hover {
            transform: translateY(-4px);
            box-shadow: 0 24px 48px rgba(0, 0, 0, 0.4), 0 0 30px var(--accent-glow);
            border-color: rgba(59, 130, 246, 0.2);
        }

        h1 {
            font-size: 24px;
            font-weight: 800;
            margin-bottom: 8px;
            background: linear-gradient(135deg, #60a5fa 0%, #34d399 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            letter-spacing: -0.5px;
        }

        p.subtitle {
            font-size: 14px;
            color: var(--text-secondary);
            margin-bottom: 30px;
        }

        .qr-wrapper {
            position: relative;
            background: rgba(255, 255, 255, 0.03);
            border: 1px dashed rgba(255, 255, 255, 0.15);
            border-radius: 16px;
            padding: 24px;
            display: inline-block;
            cursor: pointer;
            transition: all 0.3s ease;
            margin-bottom: 24px;
        }

        .qr-wrapper:hover {
            background: rgba(59, 130, 246, 0.05);
            border-color: var(--accent-color);
            transform: scale(1.02);
        }

        .qr-image {
            display: block;
            width: 200px;
            height: 200px;
            border-radius: 8px;
            pointer-events: none;
        }

        .hover-overlay {
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.5);
            border-radius: 16px;
            opacity: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: opacity 0.3s ease;
            color: white;
            font-size: 14px;
            font-weight: 600;
        }

        .qr-wrapper:hover .hover-overlay {
            opacity: 1;
        }

        .btn-copy {
            background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
            color: white;
            border: none;
            padding: 12px 24px;
            font-family: inherit;
            font-size: 15px;
            font-weight: 600;
            border-radius: 12px;
            cursor: pointer;
            transition: all 0.2s ease;
            box-shadow: 0 4px 12px rgba(37, 99, 235, 0.3);
            width: 100%;
            margin-bottom: 20px;
        }

        .btn-copy:hover {
            transform: translateY(-1px);
            box-shadow: 0 6px 16px rgba(37, 99, 235, 0.4);
            filter: brightness(1.1);
        }

        .btn-copy:active {
            transform: translateY(1px);
        }

        .payload-preview {
            background: rgba(0, 0, 0, 0.2);
            border: 1px solid rgba(255, 255, 255, 0.05);
            border-radius: 10px;
            padding: 12px;
            font-family: monospace;
            font-size: 11px;
            color: var(--text-secondary);
            word-break: break-all;
            max-height: 80px;
            overflow-y: auto;
            text-align: left;
        }

        /* Toast notification */
        .toast {
            position: fixed;
            bottom: -60px;
            left: 50%;
            transform: translateX(-50%);
            background: var(--success-color);
            color: white;
            padding: 12px 24px;
            border-radius: 30px;
            font-weight: 600;
            font-size: 14px;
            box-shadow: 0 10px 20px rgba(16, 185, 129, 0.3);
            display: flex;
            align-items: center;
            gap: 8px;
            transition: bottom 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            z-index: 100;
        }

        .toast.show {
            bottom: 40px;
        }
    </style>
</head>
<body>

    <div class="container">
        <h1>X9.150 QR Payload</h1>
        <p class="subtitle">Click the QR code or button to copy raw TLV payload</p>

        <div class="qr-wrapper" id="qrWrapper">
            <img class="qr-image" src="data:image/png;base64,{{.QR_IMAGE_BASE64}}" alt="X9.150 Payment QR Code">
            <div class="hover-overlay">
                Click to Copy Payload
            </div>
        </div>

        <button class="btn-copy" id="btnCopy">Copy Raw Payload</button>

        <div class="payload-preview" id="payloadText">{{.PAYLOAD}}</div>
    </div>

    <div class="toast" id="toast">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
        Copied to clipboard!
    </div>

    <script>
        const payload = "{{.PAYLOAD}}";
        
        const copyToClipboard = () => {
            navigator.clipboard.writeText(payload).then(() => {
                showToast();
            }).catch(err => {
                console.error("Failed to copy: ", err);
            });
        };

        const showToast = () => {
            const toast = document.getElementById("toast");
            toast.classList.add("show");
            setTimeout(() => {
                toast.classList.remove("show");
            }, 2000);
        };

        document.getElementById("qrWrapper").addEventListener("click", copyToClipboard);
        document.getElementById("btnCopy").addEventListener("click", copyToClipboard);
    </script>
</body>
</html>`
