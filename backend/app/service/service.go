package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/PatiharnKam/AiLaw/config"
	"github.com/google/generative-ai-go/genai"
)

type MessageService struct {
	cfg    *config.Config
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewService(cfg *config.Config, client *genai.Client, model *genai.GenerativeModel) *MessageService {
	return &MessageService{
		cfg:    cfg,
		client: client,
		model:  model,
	}

}

func (s *MessageService) GetMessageService(ctx context.Context, req GetMessageRequest) (*GetMessageResponse, error) {
	// mockMessage := `asdasdasdfsdagasdgadsgasdg
	//             	asdasdasdfsdagasdgadsgasdg
	// 				asdasdasdfsdagasdgadsgasdg
	// 				asdasdasdfsdagasdgadsgasdg
	// 				asdasdasdfsdagasdgadsgasdg
	// 				asdasdasdfsdagasdgadsgasdg
	// 				asdasdasdfsdagasdgadsgasdg
	// 				asdasdasdfsdagasdgadsgasdg
	// 				asdasdasdfsdagasdgadsgasdg`
	// if req.UserMessage != "" {
	// 	return &GetMessageResponse{Message: mockMessage}, nil
	// }
	// return nil, nil
	prompt := genai.Text(req.UserMessage)

	// เรียก Gemini API
	resp, err := s.model.GenerateContent(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// ดึง response text
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response generated")
	}

	// แปลง response เป็น string
	responseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		responseText += fmt.Sprintf("%v", part)
	}

	return &GetMessageResponse{Message: responseText}, nil
}

func (s *MessageService) GetMessageModelService(ctx context.Context, req GetMessageModelRequest) (*GetMessageModelResponse, error) {
	client := &http.Client{}

	// data := map[string]interface{}{
	//     "input": {"prompt":"Your prompt"}
	// }
	data := map[string]interface{}{
		"input": map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"role":    req.Input.Message[0].Role,
					"content": req.Input.Message[0].Content, // Assuming your GetMessageModelRequest has Message field
				},
			},
		},
	}
	fmt.Println(req.Input.Message[0].Content)
	fmt.Println(req.Input.Message[0].Role)
	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return nil, err
	}
	buffer := bytes.NewBuffer(jsonData)

	// var bodyReader *bytes.Reader
	// reqBody, err := json.Marshal(req)
	// if err != nil {
	// 	return nil, fmt.Errorf("marshal error: %s", err)
	// }
	// bodyReader = bytes.NewReader(reqBody)

	// jsonData, err := json.Unmarshal(req)
	// if err != nil {
	// 	fmt.Printf("Error marshaling JSON: %v\n", err)
	// 	return
	// }

	reqHttp, err := http.NewRequest("POST", s.cfg.APIkey.ModelURL, buffer)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return nil, err
	}
	reqHttp.Header.Set("Content-Type", "application/json")
	reqHttp.Header.Set("Authorization", "Bearer "+s.cfg.APIkey.ModelAPIkey)

	resp, err := client.Do(reqHttp)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Unmarshal the response into the struct
	var response GetMessageModelResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}
	fmt.Println(response)

	return &response, nil
}
