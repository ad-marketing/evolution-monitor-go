package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client é o cliente HTTP para a Evolution API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Instance representa uma instância retornada pelo fetchInstances
type Instance struct {
	Name             string `json:"name"`
	InstanceName     string `json:"instanceName"`
	ConnectionStatus string `json:"connectionStatus"`
	OwnerJid         string `json:"ownerJid"`
	ProfileName      string `json:"profileName"`
}

// ConnectionState representa o retorno do connectionState
type ConnectionState struct {
	Instance struct {
		InstanceName string `json:"instanceName"`
		State        string `json:"state"`
	} `json:"instance"`
	State string `json:"state"`
}

// SendTextRequest é o body para envio de mensagem
type SendTextRequest struct {
	Number string `json:"number"`
	Text   string `json:"text"`
}

// ChatwootConfig representa a configuração da integração Chatwoot de uma instância
type ChatwootConfig struct {
	Enabled                 bool     `json:"enabled"`
	AccountID               string   `json:"accountId"`
	Token                   string   `json:"token"`
	URL                     string   `json:"url"`
	NameInbox               string   `json:"nameInbox"`
	SignMsg                 bool     `json:"signMsg"`
	SignDelimiter           string   `json:"signDelimiter"`
	ReopenConversation      bool     `json:"reopenConversation"`
	ConversationPending     bool     `json:"conversationPending"`
	MergeBrazilContacts     bool     `json:"mergeBrazilContacts"`
	ImportContacts          bool     `json:"importContacts"`
	ImportMessages          bool     `json:"importMessages"`
	DaysLimitImportMessages int      `json:"daysLimitImportMessages"`
	Organization            string   `json:"organization"`
	Logo                    string   `json:"logo"`
	IgnoreJids              []string `json:"ignoreJids"`
	// AutoCreate só é enviado no SET (não vem no find); usado para re-sincronizar a inbox
	AutoCreate bool `json:"autoCreate,omitempty"`
}

// NewClient cria um novo cliente da API
func NewClient(baseURL, apiKey string) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// GetBaseURL retorna a URL base do cliente
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// FetchInstances busca todas as instâncias cadastradas
func (c *Client) FetchInstances() ([]Instance, error) {
	resp, err := c.doRequest("GET", "/instance/fetchInstances", nil)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var instances []Instance
	if err := json.Unmarshal(body, &instances); err != nil {
		return nil, fmt.Errorf("erro ao decodificar JSON: %w", err)
	}

	return instances, nil
}

// GetConnectionState verifica o estado de conexão de uma instância
func (c *Client) GetConnectionState(instanceName string) (string, error) {
	endpoint := fmt.Sprintf("/instance/connectionState/%s", instanceName)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return "error", fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "error", fmt.Errorf("erro ao ler resposta: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return "not_found", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "unknown", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var state ConnectionState
	if err := json.Unmarshal(body, &state); err != nil {
		return "unknown", nil
	}

	if state.Instance.State != "" {
		return state.Instance.State, nil
	}
	if state.State != "" {
		return state.State, nil
	}

	return "unknown", nil
}

// RestartInstance solicita o restart de uma instância
func (c *Client) RestartInstance(instanceName string) error {
	endpoint := fmt.Sprintf("/instance/restart/%s", instanceName)
	resp, err := c.doRequest("POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendText envia uma mensagem de texto via WhatsApp
func (c *Client) SendText(senderInstance, number, text string) error {
	endpoint := fmt.Sprintf("/message/sendText/%s", senderInstance)
	payload := SendTextRequest{
		Number: number,
		Text:   text,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload: %w", err)
	}

	resp, err := c.doRequest("POST", endpoint, jsonBody)
	if err != nil {
		return fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// FindChatwoot busca a configuração atual da integração Chatwoot de uma instância.
// Retorna (config, enabled, error). Se a instância não tiver Chatwoot configurado,
// enabled será false.
func (c *Client) FindChatwoot(instanceName string) (*ChatwootConfig, error) {
	endpoint := fmt.Sprintf("/chatwoot/find/%s", instanceName)
	resp, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Quando não há integração, a API pode retornar "null" ou objeto vazio
	trimmed := string(body)
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return nil, nil
	}

	var cfg ChatwootConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("erro ao decodificar JSON: %w", err)
	}

	return &cfg, nil
}

// SetChatwoot re-aplica a configuração da integração Chatwoot de uma instância.
// Equivale a clicar em "Save / Auto Create" na tela do Evolution Manager.
func (c *Client) SetChatwoot(instanceName string, cfg *ChatwootConfig) error {
	endpoint := fmt.Sprintf("/chatwoot/set/%s", instanceName)

	jsonBody, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("erro ao serializar payload: %w", err)
	}

	resp, err := c.doRequest("POST", endpoint, jsonBody)
	if err != nil {
		return fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) doRequest(method, endpoint string, body []byte) (*http.Response, error) {
	url := c.baseURL + endpoint

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}
