package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	rootCmd.AddCommand(branchCmd)
	cobra.OnInitialize(initConfig)
	if err := rootCmd.Execute(); err != nil {
		color.Red("%v", err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.AutomaticEnv()
	if viper.GetString("OPENAI_API_KEY") == "" {
		color.Red("OPENAI_API_KEY environment variable not set")
		return
	}
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate commit messages based on git diff",
	Run: func(cmd *cobra.Command, args []string) {
		if !hasGitChanges() {
			color.Yellow("Please make some changes to the repository and try again.")
			return
		}

		if err := stageFiles(); err != nil {
			color.Red("Error staging files: %v", err)
			return
		}

		diff := getGitDiff()
		if diff == "" {
			color.Yellow("No diff found.")
			return
		}

		commitMessage, err := GenerateDiff(diff)
		if err != nil {
			color.Red("Error generating commit message: %v", err)
			return
		}

		color.Green("Generated commit message: %s\n", commitMessage)

		if err := commitChanges(commitMessage); err != nil {
			color.Red("Error committing changes: %v", err)
			return
		}

		currentBranch, err := detectCurrentBranch()
		if err != nil {
			color.Red("Error detecting current branch: %v", err)
			return
		}

		if err := pushChanges(currentBranch); err != nil {
			color.Red("Error pushing changes: %v", err)
			return
		}
	},
}

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Generate a new branch and commit messages based on git diff",
	Run: func(cmd *cobra.Command, args []string) {
		currentBranch, err := detectCurrentBranch()
		if err != nil {
			color.Red("Error detecting current branch: %v", err)
			return
		}
		if currentBranch != "main" && currentBranch != "master" {
			color.Red("You must be on the main branch to create a new branch.")
			return
		}
		// Add files
		if !hasGitChanges() {
			color.Yellow("No changes detected in the Git repository.")
			return
		}

		if err := stageFiles(); err != nil {
			color.Red("Error staging files: %v", err)
			return
		}

		diff := getGitDiff()
		if diff == "" {
			color.Yellow("No diff found.")
			return
		}
		// Generate commit message
		commitMessage, err := GenerateDiff(diff)
		if err != nil {
			color.Red("Error generating commit message: %v", err)
			return
		}
		// Create branch
		branchName := generateBranchNameFromCommitMessage(commitMessage)
		// Switch to new branch
		err = checkoutNewBranchLocally(branchName)
		if err != nil {
			color.Red("Error checking out new branch: %v", err)
			return
		}
		// Commit changes
		if err := commitChanges(commitMessage); err != nil {
			color.Red("Error committing changes: %v", err)
			return
		}

		if err := pushChanges(branchName); err != nil {
			color.Red("Error pushing changes: %v", err)
			return
		}
	},
}

func hasGitChanges() bool {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		color.Red("Error checking git status: %v", err)
		return false
	}
	return len(out) > 0
}

func getGitDiff() string {
	stagedDiffCmd := exec.Command("git", "diff", "--staged")
	stagedDiff, err := stagedDiffCmd.Output()
	if err != nil {
		color.Red("Error getting staged git diff: %v", err)
		return ""
	}

	unstagedDiffCmd := exec.Command("git", "diff")
	unstagedDiff, err := unstagedDiffCmd.Output()
	if err != nil {
		color.Red("Error getting unstaged git diff: %v", err)
		return ""
	}

	totalDiff := strings.TrimSpace(string(stagedDiff)) + "\n" + strings.TrimSpace(string(unstagedDiff))
	return totalDiff
}

func GenerateDiff(diff string) (string, error) {
	prompt := fmt.Sprintf("Generate a git commit message based on the output of a diff command. The commit message should be a detailed but brief overview of the changes. Return only the text for the commit message. \n\nHere is the diff output:\n\n%s\n\nCommit message:", diff)
	commitMessage, err := makeOpenAPIRequestFromPrompt(prompt)
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
	color.Green("Commit message: %s\n", commitMessage)
	return nil
}

func pushChanges(branchName string) error {
	// Check if the branch exists locally
	cmd := exec.Command("git", "rev-parse", "--verify", branchName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("branch %s does not exist locally", branchName)
	}

	// Check if the branch exists remotely
	cmd = exec.Command("git", "ls-remote", "--exit-code", "--heads", "origin", branchName)
	if err := cmd.Run(); err == nil {
		// Branch exists remotely, perform a normal push
		cmd = exec.Command("git", "push", "origin", branchName)
	} else {
		// Branch does not exist remotely, set the upstream and push
		cmd = exec.Command("git", "push", "--set-upstream", "origin", branchName)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s, %v", out, err)
	}

	color.Green("Changes pushed successfully to branch %s.\n", branchName)
	return nil
}

func stageFiles() error {
	// Using "git add ." to add all changes
	cmd := exec.Command("git", "add", ".")

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error staging files: %s, %v", out, err)
	}

	color.Green("Files staged successfully.")
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
	branch, err := makeOpenAPIRequestFromPrompt(prompt)
	if err != nil {
		color.Red("Error generating branch name: %v", err)
		return ""
	}
	return branch
}

func makeOpenAPIRequestFromPrompt(prompt string) (string, error) {
	requestBody, err := json.Marshal(GPTRequest{
		Model:     ApiModel,
		Messages:  []GPTMessage{{Role: "user", Content: prompt}},
		MaxTokens: 1000,
	})
	if err != nil {
		return "", fmt.Errorf("error marshaling request body: %v", err)
	}

	response, err := makeOpenAIRequest(requestBody)
	if err != nil {
		return "", err
	}

	return response, nil
}

func checkoutNewBranchLocally(branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error checking out new branch: %s, %v", out, err)
	}
	color.Green("Switched to new branch %s\n", branchName)
	return nil
}
