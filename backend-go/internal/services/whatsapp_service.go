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
	cfg        *config.WhatsAppConfig
	httpClient *http.Client
}

// ============================================
// MENSAGENS
// ============================================

type WhatsAppMessage struct {
	MessagingProduct string              `json:"messaging_product"`
	To               string              `json:"to"`
	Type             string              `json:"type"`
	Text             *TextMessage        `json:"text,omitempty"`
	Document         *DocumentMessage    `json:"document,omitempty"`
	Image            *ImageMessage       `json:"image,omitempty"`
	Audio            *AudioMessage       `json:"audio,omitempty"`
	Location         *LocationMessage    `json:"location,omitempty"`
	Interactive      *InteractiveMessage `json:"interactive,omitempty"`
}

type TextMessage struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url,omitempty"`
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

type AudioMessage struct {
	ID string `json:"id"`
}

type LocationMessage struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type InteractiveMessage struct {
	Type   string             `json:"type"`
	Header *InteractiveHeader `json:"header,omitempty"`
	Body   *InteractiveBody   `json:"body"`
	Footer *InteractiveFooter `json:"footer,omitempty"`
	Action *InteractiveAction `json:"action"`
}

type InteractiveHeader struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type InteractiveBody struct {
	Text string `json:"text"`
}

type InteractiveFooter struct {
	Text string `json:"text"`
}

type InteractiveAction struct {
	Buttons  []InteractiveButton  `json:"buttons,omitempty"`
	Sections []InteractiveSection `json:"sections,omitempty"`
}

type InteractiveButton struct {
	Type  string `json:"type"`
	Reply struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"reply"`
}

type InteractiveSection struct {
	Title string `json:"title"`
	Rows  []struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description,omitempty"`
	} `json:"rows"`
}

// ============================================
// WEBHOOK
// ============================================

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
						URL      string `json:"url,omitempty"`
					} `json:"document,omitempty"`
					Image *struct {
						ID       string `json:"id"`
						Caption  string `json:"caption"`
						MimeType string `json:"mime_type"`
						URL      string `json:"url,omitempty"`
					} `json:"image,omitempty"`
					Audio *struct {
						ID       string `json:"id"`
						MimeType string `json:"mime_type"`
						URL      string `json:"url,omitempty"`
					} `json:"audio,omitempty"`
					Location *struct {
						Latitude  float64 `json:"latitude"`
						Longitude float64 `json:"longitude"`
						Name      string  `json:"name,omitempty"`
						Address   string  `json:"address,omitempty"`
					} `json:"location,omitempty"`
					Interactive *struct {
						Type        string `json:"type"`
						ButtonReply *struct {
							ID    string `json:"id"`
							Title string `json:"title"`
						} `json:"button_reply,omitempty"`
						ListReply *struct {
							ID    string `json:"id"`
							Title string `json:"title"`
						} `json:"list_reply,omitempty"`
					} `json:"interactive,omitempty"`
				} `json:"messages"`
				Statuses []struct {
					ID          string `json:"id"`
					Status      string `json:"status"`
					Timestamp   string `json:"timestamp"`
					RecipientID string `json:"recipient_id"`
				} `json:"statuses"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}

// ============================================
// RESPOSTAS
// ============================================

type WhatsAppResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

type WebhookMessage struct {
	From      string        `json:"from"`
	ID        string        `json:"id"`
	Timestamp string        `json:"timestamp"`
	Type      string        `json:"type"`
	Text      string        `json:"text,omitempty"`
	Document  *DocumentInfo `json:"document,omitempty"`
	Image     *ImageInfo    `json:"image,omitempty"`
	Audio     *AudioInfo    `json:"audio,omitempty"`
	Location  *LocationInfo `json:"location,omitempty"`
	ButtonID  string        `json:"button_id,omitempty"`
	ListID    string        `json:"list_id,omitempty"`
}

type DocumentInfo struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	URL      string `json:"url,omitempty"`
}

type ImageInfo struct {
	ID       string `json:"id"`
	Caption  string `json:"caption"`
	MimeType string `json:"mime_type"`
	URL      string `json:"url,omitempty"`
}

type AudioInfo struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	URL      string `json:"url,omitempty"`
}

type LocationInfo struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// ============================================
// CONSTRUTOR
// ============================================

func NewWhatsAppService(cfg *config.WhatsAppConfig) *WhatsAppService {
	return &WhatsAppService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ============================================
// ENVIO DE MENSAGENS
// ============================================

// SendTextMessage envia uma mensagem de texto
func (s *WhatsAppService) SendTextMessage(ctx context.Context, to string, text string) error {
	message := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
		Text: &TextMessage{
			Body:       text,
			PreviewURL: false,
		},
	}
	return s.sendMessage(ctx, message)
}

// SendTextMessageWithPreview envia texto com preview de URL
func (s *WhatsAppService) SendTextMessageWithPreview(ctx context.Context, to string, text string) error {
	message := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
		Text: &TextMessage{
			Body:       text,
			PreviewURL: true,
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

// SendAudioMessage envia um áudio
func (s *WhatsAppService) SendAudioMessage(ctx context.Context, to string, mediaID string) error {
	message := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "audio",
		Audio: &AudioMessage{
			ID: mediaID,
		},
	}
	return s.sendMessage(ctx, message)
}

// SendLocationMessage envia uma localização
func (s *WhatsAppService) SendLocationMessage(ctx context.Context, to string, lat, lon float64, name, address string) error {
	message := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "location",
		Location: &LocationMessage{
			Latitude:  lat,
			Longitude: lon,
			Name:      name,
			Address:   address,
		},
	}
	return s.sendMessage(ctx, message)
}

// SendInteractiveButtons envia mensagem com botões interativos
func (s *WhatsAppService) SendInteractiveButtons(ctx context.Context, to string, header, body, footer string, buttons []InteractiveButton) error {
	message := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "interactive",
		Interactive: &InteractiveMessage{
			Type: "button",
			Header: &InteractiveHeader{
				Type: "text",
				Text: header,
			},
			Body: &InteractiveBody{
				Text: body,
			},
			Footer: &InteractiveFooter{
				Text: footer,
			},
			Action: &InteractiveAction{
				Buttons: buttons,
			},
		},
	}
	return s.sendMessage(ctx, message)
}

// SendInteractiveList envia mensagem com lista interativa
func (s *WhatsAppService) SendInteractiveList(ctx context.Context, to string, header, body, footer, buttonText string, sections []InteractiveSection) error {
	message := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "interactive",
		Interactive: &InteractiveMessage{
			Type: "list",
			Header: &InteractiveHeader{
				Type: "text",
				Text: header,
			},
			Body: &InteractiveBody{
				Text: body,
			},
			Footer: &InteractiveFooter{
				Text: footer,
			},
			Action: &InteractiveAction{
				Sections: sections,
			},
		},
	}
	return s.sendMessage(ctx, message)
}

// ============================================
// MÉTODO PRIVADO DE ENVIO
// ============================================

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

// ============================================
// WEBHOOK
// ============================================

// VerifyWebhook verifica o webhook (Meta)
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
			// Processar mensagens
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
						URL:      msg.Document.URL,
					}
				}
				if msg.Image != nil {
					webhookMsg.Image = &ImageInfo{
						ID:       msg.Image.ID,
						Caption:  msg.Image.Caption,
						MimeType: msg.Image.MimeType,
						URL:      msg.Image.URL,
					}
				}
				if msg.Audio != nil {
					webhookMsg.Audio = &AudioInfo{
						ID:       msg.Audio.ID,
						MimeType: msg.Audio.MimeType,
						URL:      msg.Audio.URL,
					}
				}
				if msg.Location != nil {
					webhookMsg.Location = &LocationInfo{
						Latitude:  msg.Location.Latitude,
						Longitude: msg.Location.Longitude,
						Name:      msg.Location.Name,
						Address:   msg.Location.Address,
					}
				}
				if msg.Interactive != nil {
					if msg.Interactive.ButtonReply != nil {
						webhookMsg.ButtonID = msg.Interactive.ButtonReply.ID
					}
					if msg.Interactive.ListReply != nil {
						webhookMsg.ListID = msg.Interactive.ListReply.ID
					}
				}

				messages = append(messages, webhookMsg)
			}
		}
	}

	return messages, nil
}

// ============================================
// STATUS
// ============================================

// GetStatus retorna o status do webhook
func (s *WhatsAppService) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", s.cfg.APIVersion, s.cfg.PhoneNumberID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.cfg.AccessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return map[string]interface{}{
		"status":          "ok",
		"phone_number_id": s.cfg.PhoneNumberID,
		"api_version":     s.cfg.APIVersion,
	}, nil
}
