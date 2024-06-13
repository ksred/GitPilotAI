package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const ApiModel = "gpt-4o" // 128k context window

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
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(branchCmd)
	cobra.OnInitialize(initConfig)
	if err := rootCmd.Execute(); err != nil {
		color.Red(err.Error())
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetConfigType("env")
	viper.SetConfigName(".gitpilotai")
	viper.AddConfigPath("$HOME")

	// Attempt to read the configuration file
	configErr := viper.ReadInConfig()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		color.Red("Error finding home directory: %v", err)
		os.Exit(1)
	}
	envFilePath := homeDir + "/.gitpilotai.env"

	viper.AutomaticEnv() // Enable reading of environment variables
	// Check if the OPENAI_API_KEY environment variable is set
	envVar := viper.GetString("OPENAI_API_KEY")

	if configErr != nil { // Config file not found
		if envVar == "" {
			// Environment variable not set, create or open the config file and write the variable
			envFile, err := os.OpenFile(envFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				color.Red("Error opening or creating .gitpilotai.env file: %v", err)
				os.Exit(1)
			}
			defer envFile.Close()

			if _, err := envFile.WriteString("OPENAI_API_KEY=\n"); err != nil {
				color.Red("Error writing to .gitpilotai.env file: %v", err)
				os.Exit(1)
			}
			color.Red("OPENAI_API_KEY environment variable not set and was added to .gitpilotai.env. Please set it.")
			os.Exit(1)
		} else {
			// Environment variable is set but config file does not exist, write it to the file
			envFile, err := os.OpenFile(envFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				color.Red("Error opening or creating .gitpilotai.env file: %v", err)
				os.Exit(1)
			}
			defer envFile.Close()

			// Remove all contents from the file
			if err := envFile.Truncate(0); err != nil {
				color.Red("Error truncating .gitpilotai.env file: %v", err)
				return
			}
			if _, err := envFile.Seek(0, 0); err != nil {
				color.Red("Error seeking in .gitpilotai.env file: %v", err)
				return
			}

			if _, err := envFile.WriteString("OPENAI_API_KEY=\n"); err != nil {
				color.Red("Error writing to .gitpilotai.env file: %v", err)
				return
			}

			viper.Set("OPENAI_API_KEY", envVar)
		}
	} else {
		// Config file exists, load the variable from the file
		if envVar == "" {
			// If the environment variable is still not set, load it from the file
			envFile, err := os.Open(envFilePath)
			if err != nil {
				color.Red("Error opening .gitpilotai.env file: %v", err)
				os.Exit(1)
			}
			defer envFile.Close()

			scanner := bufio.NewScanner(envFile)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "OPENAI_API_KEY=") {
					envVar = strings.TrimPrefix(line, "OPENAI_API_KEY=")
					viper.Set("OPENAI_API_KEY", envVar)
					break
				}
			}

			if err := scanner.Err(); err != nil {
				color.Red("Error reading .gitpilotai.env file: %v", err)
				os.Exit(1)
			}
		} else {
			envFile, err := os.OpenFile(envFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				color.Red("Error opening or creating .gitpilotai.env file: %v", err)
				os.Exit(1)
			}
			defer envFile.Close()

			// Remove all contents from the file
			if err := envFile.Truncate(0); err != nil {
				color.Red("Error truncating .gitpilotai.env file: %v", err)
				return
			}
			if _, err := envFile.Seek(0, 0); err != nil {
				color.Red("Error seeking in .gitpilotai.env file: %v", err)
				return
			}

			_, err = envFile.WriteString("OPENAI_API_KEY=" + envVar + "\n")
			if err != nil {
				color.Red("Error writing to .gitpilotai.env file: %v", err)
				os.Exit(1)
			}

			viper.Set("OPENAI_API_KEY", envVar)
		}
	}
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure the API key",
	Run: func(cmd *cobra.Command, args []string) {
		color.Green("Config initialized.")
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate commit messages based on git diff",
	Run: func(cmd *cobra.Command, args []string) {
		if !hasGitChanges() {
			color.Yellow("No changes detected in the Git repository.")
			return
		}

		if err := stageFiles(); err != nil {
			color.Red("Error staging files: %v", err)
			os.Exit(1)
		}

		diff := getGitDiff()
		if diff == "" {
			color.Yellow("No diff found.")
			return
		}

		commitMessage, err := GenerateDiff(diff)
		if err != nil {
			color.Red("Error generating commit message: %v", err)
			os.Exit(1)
		}

		color.Cyan("%s\n", commitMessage)

		if err := commitChanges(commitMessage); err != nil {
			color.Red("Error committing changes: %v", err)
			os.Exit(1)
		}

		currentBranch, err := detectCurrentBranch()
		if err != nil {
			color.Red("Error detecting current branch: %v", err)
			os.Exit(1)
		}

		if err := pushChanges(currentBranch); err != nil {
			color.Red("Error pushing changes: %v", err)
			os.Exit(1)
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
			os.Exit(1)
		}
		if currentBranch != "main" && currentBranch != "master" {
			color.Red("You must be on the main branch to create a new branch.")
			os.Exit(1)
		}
		// Add files
		if !hasGitChanges() {
			color.Yellow("No changes detected in the Git repository.")
			return
		}

		if err := stageFiles(); err != nil {
			color.Red("Error staging files: %v", err)
			os.Exit(1)
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
			os.Exit(1)
		}
		// Create branch
		branchName := generateBranchNameFromCommitMessage(commitMessage)
		// Switch to new branch
		err = checkoutNewBranchLocally(branchName)
		if err != nil {
			color.Red("Error checking out new branch: %v", err)
			os.Exit(1)
		}
		// Commit changes
		if err := commitChanges(commitMessage); err != nil {
			color.Red("Error committing changes: %v", err)
			os.Exit(1)
		}

		if err := pushChanges(branchName); err != nil {
			color.Red("Error pushing changes: %v", err)
			os.Exit(1)
		}
	},
}

func hasGitChanges() bool {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		color.Red("Error checking git status: %v", err)
		os.Exit(1)
	}
	return len(out) > 0
}

func getGitDiff() string {
	stagedDiffCmd := exec.Command("git", "diff", "--staged")
	stagedDiff, err := stagedDiffCmd.Output()
	if err != nil {
		color.Red("Error getting staged git diff: %v", err)
		os.Exit(1)
	}

	unstagedDiffCmd := exec.Command("git", "diff")
	unstagedDiff, err := unstagedDiffCmd.Output()
	if err != nil {
		color.Red("Error getting unstaged git diff: %v", err)
		os.Exit(1)
	}

	totalDiff := strings.TrimSpace(string(stagedDiff)) + "\n" + strings.TrimSpace(string(unstagedDiff))
	return totalDiff
}

func GenerateDiff(diff string) (string, error) {
	prompt := fmt.Sprintf("Generate a git commit message based on the output of a diff command. The commit message should be a detailed but brief overview of the changes. Return only the text for the commit message. \n\nHere is the diff output:\n\n%s\n\nCommit message:", diff)
	commitMessage, err := makeOpenAPIRequestFromPrompt(prompt)
	if err != nil {
		color.Red("Error generating commit message: %v", err)
		os.Exit(1)
	}

	// Strip commitMessage of any quotes
	commitMessage = strings.ReplaceAll(commitMessage, "\"", "")

	return commitMessage, nil
}

func makeOpenAIRequest(body []byte) (string, error) {
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		color.Red("Error creating request: %v", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+viper.GetString("OPENAI_API_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		color.Red("Error sending request to OpenAI: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var apiResp GPTResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		color.Red("Error decoding API response: %v", err)
		os.Exit(1)
	}

	if apiResp.Error != nil {
		color.Red("Error from API: %s", apiResp.Error)
		os.Exit(1)
	}

	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}

	color.Red("No response from API")
	os.Exit(1)
	return "", nil
}

func commitChanges(commitMessage string) error {
	cmd := exec.Command("git", "commit", "-m", commitMessage)
	if out, err := cmd.CombinedOutput(); err != nil {
		color.Red("git commit failed: %s, %v", out, err)
		os.Exit(1)
	}
	color.Green("Changes committed successfully.")
	return nil
}

func pushChanges(branchName string) error {
	// Check if the branch exists locally
	cmd := exec.Command("git", "rev-parse", "--verify", branchName)
	if err := cmd.Run(); err != nil {
		color.Red("branch %s does not exist locally", branchName)
		os.Exit(1)
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
		color.Red("git push failed: %s, %v", out, err)
		os.Exit(1)
	}

	color.Green("Changes pushed successfully to branch %s.\n", branchName)
	return nil
}

func stageFiles() error {
	// Using "git add ." to add all changes
	cmd := exec.Command("git", "add", ".")

	if out, err := cmd.CombinedOutput(); err != nil {
		color.Red("error staging files: %s, %v", out, err)
		os.Exit(1)
	}

	color.Green("Files staged successfully.")
	return nil
}

func detectCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		color.Red("error detecting current branch: %v", err)
		os.Exit(1)
	}
	return strings.TrimSpace(string(out)), nil
}

func generateBranchNameFromCommitMessage(commitMessage string) string {
	prompt := fmt.Sprintf("Generate a branch name from a commit message. The branch name should be in a valid format, e.g. branch-name-of-feature. Here is the commit message:%s", commitMessage)
	branch, err := makeOpenAPIRequestFromPrompt(prompt)
	if err != nil {
		color.Red("Error generating branch name: %v", err)
		os.Exit(1)
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
		color.Red("error marshaling request body: %v", err)
		os.Exit(1)
	}

	response, err := makeOpenAIRequest(requestBody)
	if err != nil {
		color.Red("Error generating branch name: %v", err)
		os.Exit(1)
	}

	return response, nil
}

func checkoutNewBranchLocally(branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		color.Red("error checking out new branch: %s, %v", out, err)
		os.Exit(1)
	}
	color.Green("Switched to new branch %s\n", branchName)
	return nil
}
