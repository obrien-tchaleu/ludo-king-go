// internal/shared/protocol/serializer.go
package protocol

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/models"
)

// Serializer gère la sérialisation des messages
type Serializer struct {
	encoder *json.Encoder
	decoder *json.Decoder
}

// NewSerializer crée un nouveau sérialiseur
func NewSerializer(reader io.Reader, writer io.Writer) *Serializer {
	return &Serializer{
		encoder: json.NewEncoder(writer),
		decoder: json.NewDecoder(reader),
	}
}

// Encode encode un message en JSON
func (s *Serializer) Encode(msg *models.NetworkMessage) error {
	if err := s.encoder.Encode(msg); err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}
	return nil
}

// Decode décode un message JSON
func (s *Serializer) Decode(msg *models.NetworkMessage) error {
	if err := s.decoder.Decode(msg); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}
	return nil
}

// EncodeMessage encode directement un message
func EncodeMessage(msg *models.NetworkMessage) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}
	return data, nil
}

// DecodeMessage décode directement un message depuis bytes
func DecodeMessage(data []byte) (*models.NetworkMessage, error) {
	var msg models.NetworkMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	return &msg, nil
}
