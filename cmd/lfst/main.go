package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

var version = "dev" // Set by -ldflags during build

// Available subcommands
var subcommands = []struct {
	name        string
	description string
}{
	{"config", "Manage configuration"},
	{"scenario", "Execute complete test scenarios"},
	{"checksum", "Compute and verify checksums"},
	{"import", "Import checksum data"},
	{"run", "Manage test run lifecycle"},
	{"query", "Query and report on test data"},
	{"testdata", "Download Git LFS test data files"},
	{"create-eval-repo", "Create Git LFS evaluation repository"},
}

func main() {
	// Handle version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-V") {
		fmt.Printf("lfst version %s\n", version)
		os.Exit(0)
	}

	// Handle help flag
	if len(os.Args) == 1 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		printHelp()
		os.Exit(0)
	}

	// Get subcommand
	subcommand := os.Args[1]

	// Check if it's a valid subcommand
	validSubcommand := false
	for _, sc := range subcommands {
		if sc.name == subcommand {
			validSubcommand = true
			break
		}
	}

	if !validSubcommand {
		fmt.Fprintf(os.Stderr, "Error: unknown subcommand '%s'\n\n", subcommand)
		printUsage()
		os.Exit(1)
	}

	// Build the command name
	cmdName := "lfst-" + subcommand

	// Find the full path to the command
	cmdPath, err := exec.LookPath(cmdName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: command '%s' not found in PATH\n", cmdName)
		fmt.Fprintf(os.Stderr, "Make sure it is installed (try: sudo make install)\n")
		os.Exit(1)
	}

	// Prepare arguments (skip 'lfst' and the subcommand name)
	args := []string{filepath.Base(cmdPath)}
	if len(os.Args) > 2 {
		args = append(args, os.Args[2:]...)
	}

	// Execute the subcommand using execve (replaces current process)
	// This ensures the subcommand receives signals directly
	if err := syscall.Exec(cmdPath, args, os.Environ()); err != nil {
		// If exec fails, fall back to running as subprocess
		cmd := exec.Command(cmdPath, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "Error executing %s: %v\n", cmdName, err)
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lfst <command> [options]\n\n")
	fmt.Fprintf(os.Stderr, "Available commands:\n")
	for _, sc := range subcommands {
		fmt.Fprintf(os.Stderr, "  %-12s %s\n", sc.name, sc.description)
	}
	fmt.Fprintf(os.Stderr, "\nRun 'lfst <command> --help' for more information on a command.\n")
}

func printHelp() {
	fmt.Printf("lfst - Git LFS Test Framework\n\n")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Printf("DESCRIPTION:\n")
	fmt.Printf("  Comprehensive testing framework for evaluating Git LFS server implementations.\n")
	fmt.Printf("  This is a unified command that dispatches to the individual lfst-* tools.\n\n")

	fmt.Printf("USAGE:\n")
	fmt.Printf("  lfst <command> [options]\n\n")

	fmt.Printf("AVAILABLE COMMANDS:\n")
	for _, sc := range subcommands {
		fmt.Printf("  %-12s %s\n", sc.name, sc.description)
	}

	fmt.Printf("\nGLOBAL OPTIONS:\n")
	fmt.Printf("  -h, --help       Show this help message\n")
	fmt.Printf("  -V, --version    Show version\n\n")

	fmt.Printf("EXAMPLES:\n")
	fmt.Printf("  # Show configuration\n")
	fmt.Printf("  lfst config show\n\n")

	fmt.Printf("  # List available scenarios\n")
	fmt.Printf("  lfst scenario --list\n\n")

	fmt.Printf("  # Run scenario 6 with debug output\n")
	fmt.Printf("  lfst scenario -d 6\n\n")

	fmt.Printf("  # Compute checksums for a directory\n")
	fmt.Printf("  lfst checksum --skip-db --dir /path/to/repo\n\n")

	fmt.Printf("  # Query database statistics\n")
	fmt.Printf("  lfst query stats\n\n")

	fmt.Printf("GETTING STARTED:\n")
	fmt.Printf("  1. Set up configuration:\n")
	fmt.Printf("       lfst config init\n")
	fmt.Printf("       lfst config set test_data $work/git/git_lfs_test_data\n\n")

	fmt.Printf("  2. Download test data:\n")
	fmt.Printf("       lfst testdata\n\n")

	fmt.Printf("  3. List available scenarios:\n")
	fmt.Printf("       lfst scenario --list\n\n")

	fmt.Printf("  4. Run a test scenario:\n")
	fmt.Printf("       lfst scenario 6\n\n")

	fmt.Printf("  5. Create evaluation repository (optional):\n")
	fmt.Printf("       lfst create-eval-repo 3\n\n")

	fmt.Printf("For detailed help on any command:\n")
	fmt.Printf("  lfst <command> --help\n\n")

	fmt.Printf("Documentation: https://www.mslinn.com/git/5100-git-lfs.html\n")
}
