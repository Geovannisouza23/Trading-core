// Package telegram implements output.Notifier via the Telegram Bot API.
package telegram

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"trading-core/internal/application/ports/output"
)

type Notifier struct {
	httpClient *http.Client
	botToken   string
	chatID     string
}

func NewNotifier(botToken, chatID string) *Notifier {
	return &Notifier{httpClient: &http.Client{Timeout: 10 * time.Second}, botToken: botToken, chatID: chatID}
}

var _ output.Notifier = (*Notifier)(nil)

// Notify sends a message via the Telegram Bot API. Without credentials
// configured it is a deliberate, silent no-op: alerting is best-effort and
// must never block a critical action like a kill switch.
func (n *Notifier) Notify(ctx context.Context, notification output.Notification) error {
	if n.botToken == "" || n.chatID == "" {
		return nil
	}

	text := fmt.Sprintf("[%s] %s\n%s", notification.Severity, notification.Title, notification.Message)
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)

	form := url.Values{}
	form.Set("chat_id", n.chatID)
	form.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram: http %d", resp.StatusCode)
	}
	return nil
}
