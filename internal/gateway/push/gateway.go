package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Request - структура запроса к Push Service
type Request struct {
	Kind        string            `json:"kind"`
	RecipientID string            `json:"recipient_id"`
	Data        map[string]string `json:"data"`
}

type Gateway struct {
	client *http.Client
	url    string
}

// NewGateway создает клиент с собственным http.Client и таймаутом
func NewGateway(url string, timeout time.Duration) *Gateway {
	return &Gateway{
		client: &http.Client{
			Timeout: timeout,
		},
		url: url,
	}
}

func (g *Gateway) SendRequestPosition(ctx context.Context, driverID, tripID string) error {
	reqBody := Request{
		Kind:        "REQUEST_POSITION",
		RecipientID: driverID,
		Data: map[string]string{
			"trip_id": tripID,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// В контракте Push Service (contracts/openapi/push-service.openapi.yaml) 
	// путь обычно /push или /api/v1/push. Проверьте, какой именно путь там указан.
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.url+"api/v1/push", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer func() {
		_ = resp.Body.Close() // Всегда закрываем тело
	}()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("push service returned status: %d", resp.StatusCode)
	}

	return nil
}