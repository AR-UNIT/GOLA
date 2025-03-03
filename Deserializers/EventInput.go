package Deserializers

import (
	"GOLA/commons"
	"encoding/json"
	"fmt"
)

// DeserializeEventInput takes a JSON byte slice and returns an EventInputModel.
func DeserializeEventInput(data []byte) (*commons.EventInputModel, error) {
	var input commons.EventInputModel
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to deserialize event input: %w", err)
	}
	return &input, nil
}
