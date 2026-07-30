package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/config"
)

type WhatsAppService struct {
	cfg       *config.WhatsAppConfig
	httpClient *http.Client
}

type WhatsAppMessage struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             *TextMessage `json:"text,omitempty"`
	Document         *DocumentMessage `json:"document,omitempty"`
	Image            *ImageMessage `json:"image,omitempty"`
}

type TextMessage struct {
	Body string `json:"body"`
}

type DocumentMessage struct {
	ID       string `json:"id"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename"`
}

type ImageMessage struct {
	ID      string `json:"id"`
	Caption string `json:"caption,omitempty"`
}

type WhatsAppWebhook struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaID string `json:"wa_id"`
				} `json:"contacts"`
				Messages []struct {
					From      string `json:"from"`
					ID        string `json:"id"`
					Timestamp string `json:"timestamp"`
					Type      string `json:"type"`
					Text      *struct {
						Body string `json:"body"`
					} `json:"text,omitempty"`
					Document *struct {
						ID       string `json:"id"`
						Filename string `json:"filename"`
						MimeType string `json:"mime_type"`
					} `json:"document,omitempty"`
					Image *struct {
						ID       string `json:"id"`
						Caption  string `json:"caption"`
						MimeType string `json:"mime_type"`
					} `json:"image,omitempty"`
				} `json:"messages"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}

func NewWhatsAppService(cfg *config.WhatsAppConfig) *WhatsAppService {
	return &WhatsAppService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendTextMessage envia uma mensagem de texto
func (s *WhatsAppService) SendTextMessage(ctx context.Context, to string, text string) error {
	message := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
		Text: &TextMessage{
			Body: text,
		},
	}

	return s.sendMessage(ctx, message)
}

// SendDocumentMessage envia um documento
func (s *WhatsAppService) SendDocumentMessage(ctx context.Context, to string, mediaID string, filename string, caption string) error {
	message := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "document",
		Document: &DocumentMessage{
			ID:       mediaID,
			Filename: filename,
			Caption:  caption,
		},
	}

	return s.sendMessage(ctx, message)
}

// SendImageMessage envia uma imagem
func (s *WhatsAppService) SendImageMessage(ctx context.Context, to string, mediaID string, caption string) error {
	message := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "image",
		Image: &ImageMessage{
			ID:      mediaID,
			Caption: caption,
		},
	}

	return s.sendMessage(ctx, message)
}

// sendMessage envia a mensagem para a API do WhatsApp
func (s *WhatsAppService) sendMessage(ctx context.Context, message WhatsAppMessage) error {
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages",
		s.cfg.APIVersion,
		s.cfg.PhoneNumberID,
	)

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("erro ao serializar mensagem: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.cfg.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("erro na API WhatsApp: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// VerifyWebhook verifica o webhook (meta)
func (s *WhatsAppService) VerifyWebhook(mode, token, challenge string) (string, bool) {
	if mode == "subscribe" && token == s.cfg.VerifyToken {
		return challenge, true
	}
	return "", false
}

// ProcessWebhook processa o webhook recebido
func (s *WhatsAppService) ProcessWebhook(webhook *WhatsAppWebhook) ([]WebhookMessage, error) {
	var messages []WebhookMessage

	for _, entry := range webhook.Entry {
		for _, change := range entry.Changes {
			for _, msg := range change.Value.Messages {
				webhookMsg := WebhookMessage{
					From:      msg.From,
					ID:        msg.ID,
					Timestamp: msg.Timestamp,
					Type:      msg.Type,
				}

				if msg.Text != nil {
					webhookMsg.Text = msg.Text.Body
				}
				if msg.Document != nil {
					webhookMsg.Document = &DocumentInfo{
						ID:       msg.Document.ID,
						Filename: msg.Document.Filename,
						MimeType: msg.Document.MimeType,
					}
				}
				if msg.Image != nil {
					webhookMsg.Image = &ImageInfo{
						ID:       msg.Image.ID,
						Caption:  msg.Image.Caption,
						MimeType: msg.Image.MimeType,
					}
				}

				messages = append(messages, webhookMsg)
			}
		}
	}

	return messages, nil
}

type WebhookMessage struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	Document  *DocumentInfo `json:"document,omitempty"`
	Image     *ImageInfo `json:"image,omitempty"`
}

type DocumentInfo struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
}

type ImageInfo struct {
	ID       string `json:"id"`
	Caption  string `json:"caption"`
	MimeType string `json:"mime_type"`
}