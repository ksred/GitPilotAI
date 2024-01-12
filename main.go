package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const ApiModel = "gpt-3.5-turbo"

/*
curl https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
  "model": "gpt-3.5-turbo",
  "messages": [
    {
      "role": "user",
      "content": ""
    }
  ],
  "temperature": 1,
  "max_tokens": 256,
  "top_p": 1,
  "frequency_penalty": 0,
  "presence_penalty": 0
}'
*/

type GPTRequest struct {
	Messages  []GPTMessage `json:"messages"`
	MaxTokens int          `json:"max_tokens"`
	Model     string       `json:"model"`
}

type GPTMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

/*
"choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Productivity is achieved when actions lead a company towards its goal, while actions that do not help achieve the goal are not productive."
      },
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
*/

type GPTResponse struct {
	Choices []GPTChoice `json:"choices"`
	Error   interface{} `json:"error"`
}

type GPTChoice struct {
	Message GPTMessage `json:"message"`
}

func main() {
	fmt.Println("Diff Generator running...")
	// load environment variables
	loadEnv()

	// First ask if the user wants to add all files or if they have already added them
	fmt.Printf("Have you already added all files? (y/n): ")
	var answer string
	_, err := fmt.Scanln(&answer)
	if err != nil {
		return
	}
	if answer == "n" {
		// Add all files
		out, _ := exec.Command("git", "add", ".").Output()
		fmt.Printf("Added all files: %s\n", out)
	}

	// Check if there are any changes in git status
	out, _ := exec.Command("git", "status").Output()
	status := string(out)
	fmt.Printf("Git status: %s\n", status)
	// If no changes, exit. Check if the string contains "nothing to commit, working tree clean\n"
	if strings.Contains(status, "nothing to commit, working tree clean") {
		fmt.Printf("No changes detected\n")
		return
	}

	// Get diff of changes
	out, _ = exec.Command("git", "diff", "HEAD~1..HEAD").Output()
	diff := string(out)
	fmt.Printf("Diff: %s\n", diff)

	if diff == "" {
		fmt.Printf("No changes detected\n")
		return
	}

	// Generate commit message
	commitMessage, err := GenerateDiff(diff)
	if err != nil {
		log.Fatalf("error generating commit message: %v", err)
	}

	fmt.Printf("Commit message: %s\n", commitMessage)

	//Commit changes
	out, _ = exec.Command("git", "commit", "-m", commitMessage).Output()
	fmt.Printf("Changes committed: %s\n", out)

	// Push changes
	out, _ = exec.Command("git", "push").Output()
	fmt.Printf("Changes pushed: %s\n", out)
}

func GenerateDiff(diff string) (string, error) {
	// Build prompt with diff as context
	prompt := fmt.Sprintf("Generate a git commit message based on this diff:\n\n%s\n\nCommit message:", diff)

	requestBody, err := json.Marshal(GPTRequest{
		Model:     ApiModel,
		Messages:  []GPTMessage{{Role: "user", Content: prompt}},
		MaxTokens: 200, // Adjust based on your needs
	})
	if err != nil {
		return "", fmt.Errorf("error marshaling request body: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to OpenAI: %v", err)
	}
	defer resp.Body.Close()

	var apiResp GPTResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("error decoding API response: %v", err)
	}

	fmt.Printf("API Response: %v\n", apiResp)

	if apiResp.Error != nil {
		return "", fmt.Errorf("error from API: %s", apiResp.Error)
	}

	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}

	return "", nil
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}
