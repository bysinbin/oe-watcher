package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

// SendDiscordNotification sends a rich embed notification to a Discord webhook URL.
func SendDiscordNotification(webhookURL, title, description string, fields map[string]string) error {
	if strings.TrimSpace(webhookURL) == "" {
		return nil
	}

	embedFields := make([]map[string]interface{}, 0, len(fields))
	for k, v := range fields {
		embedFields = append(embedFields, map[string]interface{}{
			"name":   k,
			"value":  v,
			"inline": true,
		})
	}

	payload := map[string]interface{}{
		"username":   "OE Watcher",
		"avatar_url": "https://raw.githubusercontent.com/bysinbin/oe-watcher/main/extension/icons/icon-128.png",
		"embeds": []map[string]interface{}{
			{
				"title":       title,
				"description": description,
				"color":       0xE11D48, // Crimson Red
				"fields":      embedFields,
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
				"footer": map[string]string{
					"text": "OE Watcher Canlı Koruma",
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := httpClient.Post(webhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("discord webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// SendTelegramNotification sends an alert message to a Telegram chat via bot token.
func SendTelegramNotification(botToken, chatID, message string) error {
	if strings.TrimSpace(botToken) == "" || strings.TrimSpace(chatID) == "" {
		return nil
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", strings.TrimSpace(botToken))
	payload := map[string]interface{}{
		"chat_id":    strings.TrimSpace(chatID),
		"text":       message,
		"parse_mode": "HTML",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := httpClient.Post(apiURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("telegram request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// SendWebhookAlert sends background notifications to Discord and Telegram if configured.
func SendWebhookAlert(discordURL, tgToken, tgChatID, title, description string, fields map[string]string) {
	if discordURL != "" {
		go func() {
			_ = SendDiscordNotification(discordURL, title, description, fields)
		}()
	}

	if tgToken != "" && tgChatID != "" {
		go func() {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("<b>🚨 %s</b>\n%s\n\n", title, description))
			for k, v := range fields {
				sb.WriteString(fmt.Sprintf("• <b>%s:</b> %s\n", k, v))
			}
			_ = SendTelegramNotification(tgToken, tgChatID, sb.String())
		}()
	}
}
