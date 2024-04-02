package main

import (
	"github.com/spf13/viper"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func loadEnv() {
	// Load from .env file if it exists
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found")
	}
	viper.Set("OPENAI_API_KEY", os.Getenv("OPENAI_API_KEY"))
}

func Test_generateBranchNameFromCommitMessage(t *testing.T) {
	type args struct {
		commitMessage string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test generateBranchNameFromCommitMessage",
			args: args{
				commitMessage: "Refactored the main.go file to include a more detailed implementation of the loop logic. This increases performance and gives us better error handling. Also amended the way we write files to disk, improving this and removing potential locks.",
			},
			want: "refactored-the-main-go-file-to-include-a-more-detailed-implementation-of-the-loop-logic-this-increases-performance-and-gives-us-better-error-handling-also-amended-the-way-we-write-files-to-disk-improving-this-and-removing-potential-locks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load from .env file if it exists
			loadEnv()
			if got := generateBranchNameFromCommitMessage(tt.args.commitMessage); got != tt.want {
				t.Errorf("generateBranchNameFromCommitMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
