# GitPilotAI 🚀

Welcome to **GitPilotAI**! This is not just another Git tool; it's your AI-powered assistant for handling Git operations with a sprinkle of fun and efficiency. 🤖✨

## What is GitPilotAI? 🤔

**GitPilotAI** is a command-line tool that brings a new spin to your Git experience. Crafted in Go, it leverages AI to automate several Git tasks, streamlining your workflow and adding a bit of joy to your day-to-day dev tasks.

### Key Features 🌟

- **AI-Generated Commit Messages**: Let AI take the wheel for your commit messages, based on your code changes.
- **Automated Staging**: Stages your changes intelligently, while smartly excluding binaries.
- **Flexible and Context-Aware**: Runs in any directory, adapting to your project's context.
- **Fun and Engaging CLI**: Git operations with a touch of cheerfulness!

## Getting Started 🚀

### Prerequisites

- Go (version 1.20 or newer).
- Git installed and configured on your machine.

### Installation

#### For Mac Users

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/GitPilotAI.git
   ```

2. Navigate to the directory:

   ```bash
   cd GitPilotAI
   ```

3. Build the binary:

   ```bash
   go build -o gitpilotai
   ```

4. Move the binary to a location in your PATH (e.g., `/usr/local/bin`):

   ```bash
   mv gitpilotai /usr/local/bin/
   ```

#### For Windows Users

1. Clone the repository in Git Bash or Command Prompt:

   ```bash
   git clone https://github.com/yourusername/GitPilotAI.git
   ```

2. Navigate to the directory:

   ```bash
   cd gitpilotai
   ```

3. Build the binary:

   ```bash
   go build -o gitpilotai
   ```

4. Add the directory to your system PATH, or move the `.exe` file to a directory already in your PATH.

## Building and Installing with Makefile 🛠️

GitPilotAI includes a Makefile for easy building and installation. This tool automates the compilation and installation process with just a few commands.

### Prerequisites for Using Makefile

- Ensure you have `make` installed on your system. Most Unix-like systems, including Linux and macOS, come with `make` pre-installed.
- You should have Go installed and properly set up to compile the source code.

### How to Use the Makefile

1. **Build the Binary**:
   - To compile the source code into a binary, simply run:
     ```sh
     make
     ```
   - This command executes the default target in the Makefile, which builds the `gitpilotai` binary.

2. **Install the Binary**:
   - To install the binary to your system (specifically to `/usr/bin/`), run:
     ```sh
     sudo make install
     ```
   - This command requires sudo privileges as it writes to a system directory.

### What Does the Makefile Do?

- **Build Target**: The `build` target compiles the Go source code and generates an executable named `gitpilotai`.
- **Install Target**: The `install` target moves the `gitpilotai` binary to `/usr/bin/`, making it accessible system-wide.

### Customization

- If you wish to change the installation path or modify the build process, you can edit the Makefile according to your needs.

---

### Note

- The Makefile is designed for Unix-like systems. If you're using Windows, you'll need a different method for building and installing the application.

### Usage

Just run GitPilotAI in your Git repository:

```bash
gitpilotai generate
```

This will:
- Automatically stage your changes (excluding binaries).
- Generate an AI-crafted commit message.
- Commit the changes.

### Customization

Dive into the code to tailor GitPilotAI to your preferences. You can adjust the AI model, staging logic, or enhance the CLI outputs.

## Upcoming Improvements 🚧

- **Pull Request Management**: Streamlining PR workflows.
- **Change Impact Assessment**: Insights on the impact of code changes.
- **Release Notes Generation**: Automate your release notes creation.

## Contributing 🤝

Contributions are welcome! Whether it's bug fixing, feature additions, or documentation improvements, your input is valuable.

## Feedback and Support 📢

Encounter an issue or have an idea? Feel free to open an issue or start a discussion. Your feedback fuels GitPilotAI's evolution!

---

Enjoy coding with GitPilotAI! Let's make Git both fun and efficient! 🎉💻