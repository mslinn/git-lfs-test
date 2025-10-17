package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mslinn/git-lfs-test/pkg/config"
	"github.com/mslinn/git-lfs-test/pkg/database"
	"github.com/mslinn/git-lfs-test/pkg/scenario"
	"github.com/mslinn/git-lfs-test/pkg/timing"
	"github.com/spf13/pflag"
)

var version = "dev" // Set by -ldflags during build

// Predefined scenarios based on gitScenarios.html
var scenarios = map[int]*scenario.Scenario{
	1:  {ID: 1, Name: "Bare repo - local", ServerType: "bare", Protocol: "local", GitServer: "bare"},
	2:  {ID: 2, Name: "Bare repo - SSH", ServerType: "bare", Protocol: "ssh", GitServer: "bare"},
	6:  {ID: 6, Name: "LFS Test Server - HTTP", ServerType: "lfs-test-server", Protocol: "http", GitServer: "bare", ServerURL: "http://gojira:8080"},
	7:  {ID: 7, Name: "LFS Test Server - HTTP/GitHub", ServerType: "lfs-test-server", Protocol: "http", GitServer: "github", ServerURL: "http://gojira:8080", RepoName: "mslinn/lfs-eval-test"},
	8:  {ID: 8, Name: "Giftless - local", ServerType: "giftless", Protocol: "local", GitServer: "bare"},
	9:  {ID: 9, Name: "Giftless - SSH", ServerType: "giftless", Protocol: "ssh", GitServer: "bare"},
	13: {ID: 13, Name: "Rudolfs - local", ServerType: "rudolfs", Protocol: "local", GitServer: "bare"},
	14: {ID: 14, Name: "Rudolfs - SSH", ServerType: "rudolfs", Protocol: "ssh", GitServer: "bare"},
}

func main() {
	// Define flags
	var (
		showVersion bool
		showHelp    bool
		debug       bool
		force       bool
		dbPath      string
		workDir     string
		listOnly    bool
		cancelArg   string
	)

	pflag.BoolVarP(&showVersion, "version", "V", false, "Show version and exit")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show this help message")
	pflag.BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	pflag.BoolVarP(&debug, "verbose", "v", false, "Enable verbose output (alias for --debug)")
	pflag.BoolVarP(&force, "force", "f", false, "Force recreation of existing repositories")
	pflag.StringVar(&dbPath, "db", "", "Path to SQLite database (default from config)")
	pflag.StringVar(&workDir, "work-dir", "", "Working directory for test execution (default from config)")
	pflag.BoolVar(&listOnly, "list", false, "List available scenarios and exit")
	pflag.StringVar(&cancelArg, "cancel", "", "Cancel a running test: run ID or 'all'")
	var detailArg string
	pflag.StringVar(&detailArg, "detail", "", "Show detailed repository contents for a run ID")

	pflag.Parse()

	// Handle version
	if showVersion {
		fmt.Printf("lfst-scenario version %s\n", version)
		os.Exit(0)
	}

	// Handle help
	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Handle list
	if listOnly {
		listScenarios()
		os.Exit(0)
	}

	// Load configuration early for defaults
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Use config values if not overridden
	if dbPath == "" {
		dbPath = cfg.GetDatabasePath()
	}
	if workDir == "" {
		workDir = cfg.GetWorkDir()
	}

	// Handle cancel
	if cancelArg != "" {
		handleCancel(cancelArg, dbPath, workDir)
		os.Exit(0)
	}

	// Handle detail
	if detailArg != "" {
		handleDetail(detailArg, dbPath, workDir)
		os.Exit(0)
	}

	// Get scenario ID
	args := pflag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: scenario ID required\n\n")
		printUsage()
		os.Exit(1)
	}

	scenarioID, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid scenario ID '%s'\n", args[0])
		os.Exit(1)
	}

	// Get scenario
	scen, ok := scenarios[scenarioID]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: scenario %d not found (use --list to see available scenarios)\n", scenarioID)
		os.Exit(1)
	}

	// Validate database (creates directory if needed)
	if err := cfg.ValidateDatabase(); err != nil {
		fmt.Fprintf(os.Stderr, "Error validating database: %v\n", err)
		os.Exit(1)
	}

	// Open database
	db, err := database.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create and run scenario
	runner := scenario.NewRunner(scen, db, workDir, debug, force)
	if err := runner.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Scenario %d completed successfully\n", scenarioID)
	fmt.Printf("  Run ID: %d\n", runner.RunID)
	fmt.Printf("  View results: lfst-run show %d\n", runner.RunID)
}

func handleDetail(detailArg, dbPath, workDir string) {
	// Parse run ID
	runID, err := strconv.ParseInt(detailArg, 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid run ID '%s'\n", detailArg)
		os.Exit(1)
	}

	// Open database
	db, err := database.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Get run info
	run, err := db.GetTestRun(runID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: run %d not found\n", runID)
		os.Exit(1)
	}

	fmt.Printf("Repository Details for Run %d\n", runID)
	fmt.Printf("  Scenario: %d\n", run.ScenarioID)
	fmt.Printf("  Status: %s\n", run.Status)
	fmt.Printf("  Started: %s\n", run.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// Check if repositories exist
	repo1Dir := filepath.Join(workDir, "repo1")
	repo2Dir := filepath.Join(workDir, "repo2")

	repos := []struct {
		name string
		path string
	}{
		{"First Repository (repo1)", repo1Dir},
		{"Second Repository (repo2)", repo2Dir},
	}

	for _, repo := range repos {
		if _, err := os.Stat(repo.path); os.IsNotExist(err) {
			fmt.Printf("%s: Not found (may have been cleaned up)\n", repo.name)
			fmt.Println()
			continue
		}

		fmt.Printf("=== %s ===\n", repo.name)
		fmt.Printf("Location: %s\n\n", repo.path)

		// Show repository details
		if err := showRepositoryDetails(repo.path); err != nil {
			fmt.Printf("Error: %v\n\n", err)
		}
	}
}

func showRepositoryDetails(repoDir string) error {
	// Get LFS tracked files
	lfsResult := timing.Run("git", []string{"-C", repoDir, "lfs", "ls-files", "-n"}, nil)
	lfsFiles := make(map[string]bool)
	if lfsResult.Error == nil && lfsResult.ExitCode == 0 {
		scanner := bufio.NewScanner(strings.NewReader(lfsResult.Stdout))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				lfsFiles[line] = true
			}
		}
	}

	// Get git status to find untracked and ignored files
	statusResult := timing.Run("git", []string{"-C", repoDir, "status", "--porcelain", "--ignored"}, nil)
	untrackedFiles := make(map[string]bool)
	ignoredFiles := make(map[string]bool)
	if statusResult.Error == nil && statusResult.ExitCode == 0 {
		scanner := bufio.NewScanner(strings.NewReader(statusResult.Stdout))
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) < 3 {
				continue
			}
			status := line[0:2]
			fileName := strings.TrimSpace(line[3:])

			if strings.HasPrefix(status, "?") {
				untrackedFiles[fileName] = true
			} else if strings.HasPrefix(status, "!") {
				ignoredFiles[fileName] = true
			}
		}
	}

	// Get all files in the repository (excluding .git)
	type FileInfo struct {
		Name    string
		Size    int64
		Storage string
	}
	var files []FileInfo

	err := filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(repoDir, path)
		if err != nil {
			return err
		}

		// Determine storage type
		storage := "Git (regular)"
		if lfsFiles[relPath] {
			storage = "LFS (tracked)"
		} else if untrackedFiles[relPath] {
			storage = "Untracked"
		} else if ignoredFiles[relPath] {
			storage = "Ignored"
		}

		files = append(files, FileInfo{
			Name:    relPath,
			Size:    info.Size(),
			Storage: storage,
		})

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Print file listing
	fmt.Printf("%-50s %12s  %s\n", "File", "Size", "Storage")
	fmt.Printf("%-50s %12s  %s\n", strings.Repeat("-", 50), strings.Repeat("-", 12), strings.Repeat("-", 20))

	totalSize := int64(0)
	lfsCount := 0
	gitCount := 0
	untrackedCount := 0
	ignoredCount := 0

	for _, f := range files {
		// Format size
		sizeStr := formatSize(f.Size)
		fmt.Printf("%-50s %12s  %s\n", f.Name, sizeStr, f.Storage)

		totalSize += f.Size
		switch f.Storage {
		case "LFS (tracked)":
			lfsCount++
		case "Git (regular)":
			gitCount++
		case "Untracked":
			untrackedCount++
		case "Ignored":
			ignoredCount++
		}
	}

	fmt.Println()
	fmt.Printf("Summary:\n")
	fmt.Printf("  Total files: %d (%s)\n", len(files), formatSize(totalSize))
	fmt.Printf("  LFS tracked: %d\n", lfsCount)
	fmt.Printf("  Git regular: %d\n", gitCount)
	fmt.Printf("  Untracked:   %d\n", untrackedCount)
	fmt.Printf("  Ignored:     %d\n", ignoredCount)
	fmt.Println()

	return nil
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func handleCancel(cancelArg, dbPath, workDir string) {
	// Open database
	db, err := database.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Get runs to cancel
	var runsToCanccel []*database.TestRun

	if cancelArg == "all" {
		// Get all running tests
		allRuns, err := db.GetAllTestRuns()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting test runs: %v\n", err)
			os.Exit(1)
		}

		for _, run := range allRuns {
			if run.Status == "running" {
				runsToCanccel = append(runsToCanccel, run)
			}
		}

		if len(runsToCanccel) == 0 {
			fmt.Println("No running tests to cancel")
			return
		}
	} else {
		// Parse run ID
		runID, err := strconv.ParseInt(cancelArg, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid run ID '%s'\n", cancelArg)
			os.Exit(1)
		}

		// Get specific run
		run, err := db.GetTestRun(runID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: run %d not found\n", runID)
			os.Exit(1)
		}

		if run.Status != "running" {
			fmt.Printf("Run %d is not running (status: %s)\n", runID, run.Status)
			return
		}

		runsToCanccel = append(runsToCanccel, run)
	}

	// Cancel each run
	for _, run := range runsToCanccel {
		fmt.Printf("Cancelling run %d (PID %d)...\n", run.ID, run.PID)

		// Try to terminate the process
		if run.PID > 0 {
			process, err := os.FindProcess(run.PID)
			if err == nil {
				// Send SIGTERM for graceful shutdown
				err = process.Signal(syscall.SIGTERM)
				if err == nil {
					fmt.Printf("  Sent SIGTERM to process %d\n", run.PID)

					// Wait a bit for graceful shutdown
					time.Sleep(2 * time.Second)

					// Check if process is still running
					err = process.Signal(syscall.Signal(0))
					if err == nil {
						// Process still running, send SIGKILL
						process.Kill()
						fmt.Printf("  Sent SIGKILL to process %d\n", run.PID)
					}
				} else {
					fmt.Printf("  Process %d not found (may have already exited)\n", run.PID)
				}
			}
		}

		// Clean up working directories
		repo1Dir := filepath.Join(workDir, "repo1")
		repo2Dir := filepath.Join(workDir, "repo2")

		if _, err := os.Stat(repo1Dir); err == nil {
			if err := os.RemoveAll(repo1Dir); err != nil {
				fmt.Printf("  Warning: failed to remove %s: %v\n", repo1Dir, err)
			} else {
				fmt.Printf("  Removed %s\n", repo1Dir)
			}
		}

		if _, err := os.Stat(repo2Dir); err == nil {
			if err := os.RemoveAll(repo2Dir); err != nil {
				fmt.Printf("  Warning: failed to remove %s: %v\n", repo2Dir, err)
			} else {
				fmt.Printf("  Removed %s\n", repo2Dir)
			}
		}

		// Mark run as cancelled in database
		run.Status = "cancelled"
		run.PID = 0
		completedNow := time.Now()
		run.CompletedAt = &completedNow
		run.Notes += " | Cancelled by user"

		if err := db.UpdateTestRun(run); err != nil {
			fmt.Printf("  Warning: failed to update run status: %v\n", err)
		} else {
			fmt.Printf("  ✓ Run %d marked as cancelled\n", run.ID)
		}
	}

	fmt.Printf("\nCancelled %d test run(s)\n", len(runsToCanccel))
}

func listScenarios() {
	fmt.Println("Available scenarios:")
	fmt.Println()
	fmt.Println("ID  Server             Protocol  Git Server  Description")
	fmt.Println("--  ------             --------  ----------  -----------")

	// Print in order
	ids := []int{1, 2, 6, 7, 8, 9, 13, 14}
	for _, id := range ids {
		scen := scenarios[id]
		fmt.Printf("%-3d %-18s %-9s %-11s %s\n",
			scen.ID,
			scen.ServerType,
			scen.Protocol,
			scen.GitServer,
			scen.Name,
		)
	}

	fmt.Println()
	fmt.Println("Note: Only scenarios 1, 2, 6-9, and 13-14 are currently implemented.")
	fmt.Println("      Additional scenarios require specific server configurations.")
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lfst-scenario [OPTIONS] SCENARIO_ID\n\n")
	fmt.Fprintf(os.Stderr, "Run a complete Git LFS test scenario (all 7 steps)\n\n")
	pflag.PrintDefaults()
}

func printHelp() {
	fmt.Printf("lfst-scenario - Execute complete Git LFS test scenarios\n\n")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Printf("DESCRIPTION:\n")
	fmt.Printf("  Executes a complete 7-step Git LFS evaluation scenario:\n")
	fmt.Printf("    1. Setup repository, configure LFS, copy initial files (~1.3GB)\n")
	fmt.Printf("    2. Add, commit, and push with timing measurements\n")
	fmt.Printf("    3. Modify, delete, and rename files\n")
	fmt.Printf("    4. Clone to second machine and verify checksums\n")
	fmt.Printf("    5. Make changes on second machine\n")
	fmt.Printf("    6. Pull changes back to first machine\n")
	fmt.Printf("    7. Untrack files from LFS\n\n")

	fmt.Printf("USAGE:\n")
	fmt.Printf("  lfst-scenario [OPTIONS] SCENARIO_ID\n\n")

	fmt.Printf("OPTIONS:\n")
	pflag.PrintDefaults()

	fmt.Printf("\nEXAMPLES:\n")
	fmt.Printf("  # List available scenarios\n")
	fmt.Printf("  lfst-scenario --list\n\n")

	fmt.Printf("  # Run scenario 6 (LFS Test Server - HTTP)\n")
	fmt.Printf("  lfst-scenario 6\n\n")

	fmt.Printf("  # Run with debug output\n")
	fmt.Printf("  lfst-scenario -d 6\n\n")

	fmt.Printf("  # Use custom work directory\n")
	fmt.Printf("  lfst-scenario --work-dir /mnt/o/lfs_test 6\n\n")

	fmt.Printf("NOTES:\n")
	fmt.Printf("  - Requires ~2.4GB of test data (set LFS_TEST_DATA environment variable)\n")
	fmt.Printf("  - Work directory should have at least 5GB free space\n")
	fmt.Printf("  - For remote scenarios, requires passwordless SSH to gojira\n")
	fmt.Printf("  - Each run creates a test_run record in the database\n")
	fmt.Printf("  - All operations are timed with millisecond precision\n")
	fmt.Printf("  - Checksums are computed and stored for each step\n\n")
}
