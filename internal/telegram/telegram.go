package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Bot representa um bot do Telegram
type Bot struct {
	token  string
	client *http.Client
}

// New cria uma nova instância do Bot
func New(token string) *Bot {
	return &Bot{
		token: token,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SendMessage envia uma mensagem de texto para um chat
func (b *Bot) SendMessage(chatID, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.token)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload: %w", err)
	}

	resp, err := b.client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API retornou status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ValidateToken verifica se o token do bot é válido
func (b *Bot) ValidateToken() (string, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", b.token)

	resp, err := b.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("erro ao validar token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token inválido (status %d)", resp.StatusCode)
	}

	var result struct {
		Ok     bool `json:"ok"`
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if !result.Ok {
		return "", fmt.Errorf("token inválido")
	}

	return result.Result.Username, nil
}
