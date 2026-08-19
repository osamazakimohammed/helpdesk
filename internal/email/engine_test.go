package email

import (
	"testing"
)

func TestExtractReferenceToken(t *testing.T) {
	tests := []struct {
		subject  string
		expected string
	}{
		{"Re: [TCK-10042] Server connection timeout", "TCK-10042"},
		{"[TCK-999] Billing issue", "TCK-999"},
		{"General Inquiry without token", ""},
		{"[ALERT] Not a ticket token", ""},
	}

	for _, tt := range tests {
		got := ExtractReferenceToken(tt.subject)
		if got != tt.expected {
			t.Errorf("ExtractReferenceToken(%q) = %q; want %q", tt.subject, got, tt.expected)
		}
	}
}

func TestStripQuotedReply(t *testing.T) {
	body := `Thanks for your help! Everything is working now.

On Mon, Jan 5, 2026 at 10:00 AM Support Team wrote:
> Can you verify if restarting the server fixed the issue?`

	stripped := StripQuotedReply(body)
	expected := "Thanks for your help! Everything is working now."

	if stripped != expected {
		t.Errorf("StripQuotedReply failed:\ngot:\n%q\nwant:\n%q", stripped, expected)
	}
}

func TestSanitizeHTML(t *testing.T) {
	engine := NewEngine(nil, nil)
	dirty := `<p>Hello <script>alert('xss')</script><b>Support</b><img src="http://example.com/pic.png" onerror="alert(1)"></p>`
	clean := engine.SanitizeHTML(dirty)

	if clean == "" || len(clean) >= len(dirty) {
		t.Errorf("SanitizeHTML failed to strip scripts, got: %s", clean)
	}
	if clean != `<p>Hello <b>Support</b><img src="http://example.com/pic.png"></p>` {
		t.Errorf("unexpected sanitized output: %s", clean)
	}
}
