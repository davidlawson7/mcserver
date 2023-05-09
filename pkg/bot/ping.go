package bot

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Response struct {
	Status int    `json:"statusCode,omitempty"`
	Body   string `json:"body,omitempty"`
}

type MCStartResponseBody struct {
	Message   string `json:"message"`
	IPAddress string `json:"ipAddress,omitempty"`
}

func ParseLambda(data []byte) (*Response, *MCStartResponseBody, error) {
	var payload Response
	var body MCStartResponseBody

	if err := json.Unmarshal(data, &payload); err != nil {
		// Handle better
		fmt.Println(err)
		return &payload, &body, err
	}

	if err := json.Unmarshal([]byte(payload.Body), &body); err != nil {
		// Handle better
		fmt.Println(err)
		return &payload, &body, err
	}

	return &payload, &body, nil
}

func NaturalizeMOTD(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = regexp.MustCompile("\u00a7[a-f0-9k-or]").ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	return s
}
