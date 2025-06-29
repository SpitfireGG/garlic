package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"net/http"
	"net/http/httputil"

	"github.com/charmbracelet/glamour"
	"github.com/spitfiregg/garlic/internal/config"
	"google.golang.org/genai"
)

type LogTransport struct {
	RoundTripper http.RoundTripper
}

func (lt *LogTransport) RoundTrip(req *http.Request) (*http.Response, error) {

	Df, err := os.Create("gemini_req_dump.log")
	if err != nil {
		log.Printf("Error creating gemini_req_dump.log: %v", err)
		return lt.RoundTripper.RoundTrip(req)
	}

	requestDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Printf("Error dump request: %v", err)
	}

	httpRequestDumpFile, err := os.OpenFile("http_req_dump_gemini.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Error http dump request failed: %v", err)
	} else {
		httpRequestDumpFile.WriteString(fmt.Sprintf("--- %s Request to %s ---\n\n", req.Method, req.URL.String()))
		httpRequestDumpFile.Write(requestDump)
		httpRequestDumpFile.WriteString("\n--- End Request ---\n\n")
	}

	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			log.Printf("Error reading request body for logging: %v", err)
		} else {
			_, writeErr := Df.Write(bodyBytes)
			if writeErr != nil {
				log.Printf("Error writing to gemini_req_dump: %v", writeErr)
			}

			_, writeErr = Df.WriteString("\n")
			if writeErr != nil {
				log.Printf("Error writing newline to gemini_req_dump.log: %v", writeErr)
			}
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return lt.RoundTripper.RoundTrip(req)
}

// initialize a new Gemini client
func NewGeminiClient(apiKey string) (*genai.Client, error) {
	ctx := context.Background()

	httpClient := &http.Client{
		Transport: &LogTransport{RoundTripper: http.DefaultTransport},
		Timeout:   0,
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key was not found, please check again if the API Key exists")
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:     apiKey,
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: httpClient,
	})
	if err != nil {
		return nil, fmt.Errorf("Could not initialize a Gemini client")
	}
	return client, nil
}

func GenerateContent(api, prompt string) (string, error) {

	ctx := context.Background()
	client, _ := NewGeminiClient(api)
	model := NewDefaultAppConfig(api).GeminiDefault.GeminiConfig.Model
	cfg := GenerateContentConfigFromGeminiConfig(&config.NewDefaultAppConfig().GeminiDefault)

	res, err := client.Models.GenerateContent(ctx, model, genai.Text(prompt), cfg)
	if err != nil {
		return "", fmt.Errorf("error calling GenerateContent: %w", err)
	}
	defer client.ClientConfig().HTTPClient.CloseIdleConnections()

	if len(res.Candidates) == 0 {
		// candiates are the differenct responses the LLM redponds with
		// res.PromptFeedback is recieved when any violation prompt is sent to the LLM is found, eg: pornographic or hacking questions or something
		if res.PromptFeedback != nil && len(res.PromptFeedback.BlockReason) > 0 {
			return "", fmt.Errorf("no response candidate found, bot was blocked due to violation: %v", res.PromptFeedback.BlockReason)
		}
		return "", fmt.Errorf("no response candidate found, issue with the model or something")
	}

	var botResponse strings.Builder
	for _, part := range res.Candidates[0].Content.Parts {
		botResponse.WriteString(part.Text)
	}

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(120),
	)

	renderedResponse, err := r.Render(botResponse.String())

	return renderedResponse, nil
}
