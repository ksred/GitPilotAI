package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const ApiModel = "gpt-4-turbo-preview" // 128k context window

type GPTRequest struct {
	Messages  []GPTMessage `json:"messages"`
	MaxTokens int          `json:"max_tokens"`
	Model     string       `json:"model"`
}

type GPTMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GPTResponse struct {
	Choices []GPTChoice `json:"choices"`
	Error   interface{} `json:"error"`
}

type GPTChoice struct {
	Message GPTMessage `json:"message"`
}

func main() {
	// Initialize Cobra
	var rootCmd = &cobra.Command{Use: "gitpilotai"}
	rootCmd.AddCommand(generateCmd)
	cobra.OnInitialize(initConfig)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.AutomaticEnv()
	if viper.GetString("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate commit messages based on git diff",
	Run: func(cmd *cobra.Command, args []string) {
		if !hasGitChanges() {
			fmt.Println("No changes detected in the Git repository.")
			return
		}

		if err := stageFiles(); err != nil {
			log.Fatalf("Error staging files: %v", err)
		}

		diff := getGitDiff()
		if diff == "" {
			fmt.Println("No diff found.")
			return
		}

		commitMessage, err := GenerateDiff(diff)
		if err != nil {
			log.Fatalf("Error generating commit message: %v", err)
		}

		fmt.Printf("Generated commit message: %s\n", commitMessage)

		if err := commitChanges(commitMessage); err != nil {
			log.Fatalf("Error committing changes: %v", err)
		}

		if err := pushChanges(); err != nil {
			log.Fatalf("Error pushing changes: %v", err)
		}
	},
}

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Generate a new branch and commit messages based on git diff",
	Run: func(cmd *cobra.Command, args []string) {
		currentBranch, err := detectCurrentBranch()
		if err != nil {
			log.Fatalf("Error detecting current branch: %v", err)
		}
		if currentBranch != "main" && currentBranch != "master" {
			log.Fatalf("You must be on the main branch to create a new branch.")
		}

	},
}

func hasGitChanges() bool {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		log.Fatalf("Error checking git status: %v", err)
	}
	return len(out) > 0
}

func getGitDiff() string {
	stagedDiffCmd := exec.Command("git", "diff", "--staged")
	stagedDiff, err := stagedDiffCmd.Output()
	if err != nil {
		log.Printf("Error getting staged git diff: %v", err)
	}

	unstagedDiffCmd := exec.Command("git", "diff")
	unstagedDiff, err := unstagedDiffCmd.Output()
	if err != nil {
		log.Printf("Error getting unstaged git diff: %v", err)
	}

	totalDiff := strings.TrimSpace(string(stagedDiff)) + "\n" + strings.TrimSpace(string(unstagedDiff))
	return totalDiff
}

func GenerateDiff(diff string) (string, error) {
	prompt := fmt.Sprintf("Generate a git commit message based on the output of a diff command. The commit message should be a detailed but brief overview of the changes. Return only the text for the commit message. \n\nHere is the diff output:\n\n%s\n\nCommit message:", diff)

	requestBody, err := json.Marshal(GPTRequest{
		Model:     ApiModel,
		Messages:  []GPTMessage{{Role: "user", Content: prompt}},
		MaxTokens: 1000,
	})
	if err != nil {
		return "", fmt.Errorf("error marshaling request body: %v", err)
	}

	commitMessage, err := makeOpenAIRequest(requestBody)
	if err != nil {
		return "", err
	}

	// Strip commitMessage of any quotes
	commitMessage = strings.ReplaceAll(commitMessage, "\"", "")

	return commitMessage, nil
}

func makeOpenAIRequest(body []byte) (string, error) {
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+viper.GetString("OPENAI_API_KEY"))

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

	if apiResp.Error != nil {
		return "", fmt.Errorf("error from API: %s", apiResp.Error)
	}

	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from API")
}

func commitChanges(commitMessage string) error {
	cmd := exec.Command("git", "commit", "-m", commitMessage)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s, %v", out, err)
	}
	fmt.Println("Changes committed successfully.")
	return nil
}

func pushChanges() error {
	cmd := exec.Command("git", "push")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s, %v", out, err)
	}
	fmt.Println("Changes pushed successfully.")
	return nil
}

func stageFiles() error {
	// Using "git add ." to add all changes
	cmd := exec.Command("git", "add", ".")

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error staging files: %s, %v", out, err)
	}

	fmt.Println("Files staged successfully.")
	return nil
}

func detectCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error detecting current branch: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func generateBranchNameFromCommitMessage(commitMessage string) string {
	prompt := fmt.Sprintf("Generate a branch name from a commit message. The branch name should be in a valid format, e.g. branch-name-of-feature. Here is the commit message:%s", commitMessage)
	// TODO implement branch name generation
	return prompt
}
