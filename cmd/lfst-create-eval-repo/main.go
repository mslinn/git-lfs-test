package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mslinn/git-lfs-test/pkg/git"
	"github.com/mslinn/git-lfs-test/pkg/testdata"
	"github.com/spf13/pflag"
)

var version = "dev" // Set by -ldflags during build

func main() {
	// Define flags
	var (
		showVersion bool
		showHelp    bool
		debug       bool
		force       bool
		workDir     string
	)

	pflag.BoolVarP(&showVersion, "version", "V", false, "Show version and exit")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show this help message")
	pflag.BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	pflag.BoolVarP(&force, "force", "f", false, "Force recreation if repository already exists")
	pflag.StringVar(&workDir, "work", "", "Work directory (default: from $work environment variable)")

	pflag.Parse()

	// Handle version
	if showVersion {
		fmt.Printf("lfst-create-eval-repo version %s\n", version)
		os.Exit(0)
	}

	// Handle help
	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Get scenario number
	args := pflag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: Please provide the scenario number.\n\n")
		printUsage()
		os.Exit(1)
	}

	var scenarioNum int
	if _, err := fmt.Sscanf(args[0], "%d", &scenarioNum); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid scenario number '%s'\n", args[0])
		os.Exit(1)
	}

	// Validate scenario number
	if scenarioNum < 1 {
		fmt.Fprintf(os.Stderr, "Error: Invalid scenario number must be at least 3 ('%d' was provided).\n", scenarioNum)
		os.Exit(1)
	}
	if scenarioNum < 3 {
		fmt.Fprintf(os.Stderr, "Error: Scenarios 1 and 2 are for bare git repositories; use newBareRepo instead.\n")
		os.Exit(1)
	}
	if scenarioNum > 9 {
		fmt.Fprintf(os.Stderr, "Error: Invalid scenario number must be less than 10 ('%d' was provided).\n", scenarioNum)
		os.Exit(1)
	}

	// Check dependencies
	if err := checkDependencies(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Determine work directory
	if workDir == "" {
		workDir = os.Getenv("work")
		if workDir == "" {
			fmt.Fprintf(os.Stderr, "Error: the \"work\" environment variable is undefined and --work flag not provided.\n")
			os.Exit(1)
		}
	}

	// Create repository
	if err := createEvalRepo(scenarioNum, workDir, force, debug); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("All done.")
}

func createEvalRepo(scenarioNum int, workDir string, force, debug bool) error {
	scenarioName := fmt.Sprintf("scenario%d", scenarioNum)
	repoDir := filepath.Join(workDir, "git", scenarioName)
	lfsDir := repoDir + ".lfs"

	if debug {
		fmt.Printf("Creating evaluation repository for %s\n", scenarioName)
		fmt.Printf("  Repository: %s\n", repoDir)
		fmt.Printf("  LFS dir: %s\n", lfsDir)
	}

	// Check if directory already exists
	if _, err := os.Stat(repoDir); err == nil {
		fmt.Fprintf(os.Stderr, "Error: the directory '%s' already exists.\n", repoDir)
		return fmt.Errorf("directory already exists")
	}

	// Create directories
	fmt.Printf("Creating '%s'\n", repoDir)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}
	if err := os.MkdirAll(lfsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LFS directory: %w", err)
	}

	// Initialize git repository
	fmt.Println("Initializing the repository on this computer.")
	ctx := &git.Context{
		Debug:      debug,
		StepNumber: 0,
		WorkDir:    repoDir,
	}

	if err := ctx.InitRepo(repoDir, false); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Install git-lfs
	if err := ctx.LFSInstall(repoDir); err != nil {
		return fmt.Errorf("failed to install git-lfs: %w", err)
	}

	// Check if GitHub repository exists
	repoName := scenarioName
	if err := checkGitHubRepo(repoName, force, debug); err != nil {
		return err
	}

	// Create GitHub repository
	fmt.Printf("Creating private repository '%s' on GitHub\n", repoName)
	if err := createGitHubRepo(repoName, repoDir, debug); err != nil {
		return fmt.Errorf("failed to create GitHub repository: %w", err)
	}

	// Populate repository with test data
	if err := populateRepo(repoDir, scenarioNum, debug); err != nil {
		return fmt.Errorf("failed to populate repository: %w", err)
	}

	return nil
}

func checkGitHubRepo(repoName string, force, debug bool) error {
	// Get current user
	userCmd := exec.Command("gh", "api", "user", "-q", ".login")
	userOutput, err := userCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get GitHub user: %w", err)
	}
	user := string(userOutput)
	user = user[:len(user)-1] // trim newline

	// Check if repo exists
	fullRepoName := fmt.Sprintf("%s/%s", user, repoName)
	checkCmd := exec.Command("gh", "repo", "view", fullRepoName)
	err = checkCmd.Run()

	if err == nil {
		// Repository exists
		if force {
			fmt.Printf("Recreating the '%s' repository on GitHub\n", repoName)
			deleteCmd := exec.Command("gh", "repo", "delete", fullRepoName, "--yes")
			if err := deleteCmd.Run(); err != nil {
				return fmt.Errorf("failed to delete existing repository: %w", err)
			}
		} else {
			return fmt.Errorf("a repository called '%s' already exists in your GitHub account and the -f option was not specified", repoName)
		}
	}

	return nil
}

func createGitHubRepo(repoName, repoDir string, debug bool) error {
	// Create private repository
	cmd := exec.Command("gh", "repo", "create", repoName, "--private", "--source="+repoDir, "--remote=origin")
	if debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh repo create failed: %w\nOutput: %s", err, string(output))
	}

	if debug {
		fmt.Printf("✓ Created GitHub repository: %s\n", repoName)
	}

	return nil
}

func populateRepo(repoDir string, scenarioNum int, debug bool) error {
	fmt.Println("Populating repository with test data")

	// Create README.md
	readmePath := filepath.Join(repoDir, "README.md")
	readmeContent := fmt.Sprintf("# Scenario %d\nThis is a normal file.\n", scenarioNum)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	// Find test data directory
	testDataPath, err := testdata.GetTestDataPath()
	if err != nil {
		return fmt.Errorf("test data not found: %w\n\nPlease run 'lfst-testdata' first to download test data", err)
	}

	if debug {
		fmt.Printf("Copying test data from %s\n", testDataPath)
	}

	// Copy test data using rsync
	cmd := exec.Command("rsync", "-at", "--progress", testDataPath+"/", repoDir+"/")
	if debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy test data: %w", err)
	}

	if debug {
		fmt.Println("✓ Test data copied successfully")
	}

	return nil
}

func checkDependencies() error {
	// Check for git
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is required but not found in PATH")
	}

	// Check for git-lfs
	if _, err := exec.LookPath("git-lfs"); err != nil {
		return fmt.Errorf("git-lfs is required but not found in PATH")
	}

	// Check for gh (GitHub CLI)
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh (GitHub CLI) is required but not found in PATH\nInstall with: sudo apt install gh")
	}

	// Check for rsync
	if _, err := exec.LookPath("rsync"); err != nil {
		return fmt.Errorf("rsync is required but not found in PATH")
	}

	return nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lfst-create-eval-repo [OPTIONS] SCENARIO_NUMBER\n")
	fmt.Fprintf(os.Stderr, "Try 'lfst-create-eval-repo --help' for more information.\n")
}

func printHelp() {
	fmt.Printf("lfst-create-eval-repo - Create Git LFS evaluation repository\n\n")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Printf("DESCRIPTION:\n")
	fmt.Printf("  Creates a standard git repository for testing Git LFS implementations.\n")
	fmt.Printf("  This script creates a new Git repository, and an empty clone of the new\n")
	fmt.Printf("  repository on GitHub. The local copy is then populated with test data.\n\n")

	fmt.Printf("  This script uses test data from the configured test data directory,\n")
	fmt.Printf("  which must exist. See lfst-testdata command to download test data.\n\n")

	fmt.Printf("  Scenarios 1 and 2 exercise bare git repositories, created by newBareRepo.\n")
	fmt.Printf("  This command only supports scenarios 3-9.\n\n")

	fmt.Printf("USAGE:\n")
	fmt.Printf("  lfst-create-eval-repo [OPTIONS] SCENARIO_NUMBER\n\n")

	fmt.Printf("ARGUMENTS:\n")
	fmt.Printf("  SCENARIO_NUMBER    Scenario number (3-9)\n\n")

	fmt.Printf("OPTIONS:\n")
	fmt.Printf("  -h, --help         Show this help message\n")
	fmt.Printf("  -V, --version      Show version\n")
	fmt.Printf("  -d, --debug        Enable debug output\n")
	fmt.Printf("  -f, --force        Force recreation if repository already exists\n")
	fmt.Printf("  --work PATH        Work directory (default: $work environment variable)\n\n")

	fmt.Printf("EXAMPLES:\n")
	fmt.Printf("  # Create evaluation repository for scenario 3\n")
	fmt.Printf("  lfst-create-eval-repo 3\n\n")

	fmt.Printf("  # Force recreate scenario 5 repository\n")
	fmt.Printf("  lfst-create-eval-repo --force 5\n\n")

	fmt.Printf("  # Create with custom work directory\n")
	fmt.Printf("  lfst-create-eval-repo --work /tmp/lfs-work 4\n\n")

	fmt.Printf("DEPENDENCIES:\n")
	fmt.Printf("  - git\n")
	fmt.Printf("  - git-lfs\n")
	fmt.Printf("  - gh (GitHub CLI)\n")
	fmt.Printf("  - rsync\n\n")

	fmt.Printf("DOCUMENTATION:\n")
	fmt.Printf("  https://www.mslinn.com/git/5600-git-lfs-evaluation.html\n\n")
}
