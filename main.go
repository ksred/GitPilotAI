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
const Version = "0.0.1"

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

const CommitMessagePrompt = `
Generate a git commit message based on the output of a diff command. 
The commit message should be a detailed but brief overview of the changes. 
Return only the text for the commit message. 

Here is the diff output:
%s
`

const AdditionalCommitMessagePrompt = `
The following extra context has been added by the user. Take it into account when generating the commit message.
This information *must* be included in the commit message, aligned to the overall diff changes in the commit.

Here is the additional context:
%s

---
`

const BranchNamePrompt = `
Generate a branch name from a commit message. The branch name should be in a valid format, e.g. feat/branch-name-of-feature.
It should be prefixed with "feat/", "fix/", "perf/", "docs/", "style/", "refactor/", "test/", "chore/" or "bug/".
Here is the commit message:
%s

Only respond with the branch name, nothing else.
`

func main() {
	// Initialize Cobra
	var rootCmd = &cobra.Command{Use: "gitpilotai"}
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(branchCmd)
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(versionCmd)
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

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("GitPilotAI Version: %s\n", Version)
	},
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
		gitMessage := ""
		if len(args) == 1 && args[0] == "m" {
			color.Blue("Enter your git message (press Enter to finish):")
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					break
				}
				gitMessage += line + "\n"
			}
			gitMessage = strings.TrimSpace(gitMessage)
			color.Blue("Received git message: %s", gitMessage)
		} else if len(args) > 0 {
			gitMessage = strings.Join(args, " ")
			color.Blue("Received git message: %s", gitMessage)
		}

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

		commitMessage, err := GenerateDiff(diff, gitMessage)
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
		gitMessage := ""
		if len(args) > 0 {
			gitMessage = strings.Join(args, " ")
			color.Blue("Received git message: %s", gitMessage)
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
		commitMessage, err := GenerateDiff(diff, gitMessage)
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

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Open a pull request",
	Run: func(cmd *cobra.Command, args []string) {
		if err := openPullRequest(); err != nil {
			color.Red("Error opening pull request: %v", err)
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

// getGitDiff retrieves the staged and unstaged git diff.
func getGitDiff() string {
	// Retrieve staged diff
	stagedDiffCmd := exec.Command("git", "diff", "--staged")
	stagedDiff, err := stagedDiffCmd.Output()
	if err != nil {
		color.Red("Error getting staged git diff: %v", err)
		os.Exit(1)
	}

	// Retrieve unstaged diff
	unstagedDiffCmd := exec.Command("git", "diff")
	unstagedDiff, err := unstagedDiffCmd.Output()
	if err != nil {
		color.Red("Error getting unstaged git diff: %v", err)
		os.Exit(1)
	}

	// Combine staged and unstaged diff
	totalDiff := strings.TrimSpace(string(stagedDiff)) + "\n" + strings.TrimSpace(string(unstagedDiff))
	return totalDiff
}

func GenerateDiff(diff string, gitMessage string) (string, error) {
	prompt := fmt.Sprintf(CommitMessagePrompt, diff)
	if gitMessage != "" {
		prompt += fmt.Sprintf(AdditionalCommitMessagePrompt, gitMessage)
	}
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
	// List the files to be staged
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		color.Red("Error listing files: %v", err)
		os.Exit(1)
	}
	fmt.Println("Files to be staged:")
	fmt.Println(string(out))

	// Ask for confirmation
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Are you sure you want to stage these changes? (y/n): ")
	confirmation, _ := reader.ReadString('\n')
	confirmation = strings.TrimSpace(confirmation)

	if confirmation != "y" && confirmation != "Y" {
		color.Yellow("Staging canceled.")
		return nil
	}

	// Stage the files
	cmd = exec.Command("git", "add", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		color.Red("Error staging files: %s, %v", out, err)
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
	prompt := fmt.Sprintf(BranchNamePrompt, commitMessage)
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

func openPullRequest() error {
	repoURL, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		color.Red("error retrieving repository URL: %v", err)
		os.Exit(1)
	}
	repoURLStr := strings.TrimSpace(string(repoURL))
	repoURLStr = strings.TrimSuffix(repoURLStr, ".git")
	repoURLStr = strings.Replace(repoURLStr, "git@", "https://", 1)

	currentBranchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	currentBranchOutput, err := currentBranchCmd.Output()
	if err != nil {
		color.Red("error retrieving current branch name: %v", err)
		os.Exit(1)
	}
	branchName := strings.TrimSpace(string(currentBranchOutput))

	prURL := fmt.Sprintf("%s/compare/%s", repoURLStr, branchName)
	cmd := exec.Command("open", prURL)
	if out, err := cmd.CombinedOutput(); err != nil {
		color.Red("error opening pull request: %s, %v", out, err)
		os.Exit(1)
	}
	return nil
}
