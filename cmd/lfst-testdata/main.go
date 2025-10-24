package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mslinn/git-lfs-test/pkg/config"
	"github.com/mslinn/git-lfs-test/pkg/download"
	"github.com/spf13/pflag"
)

var version = "dev" // Set by -ldflags during build

// Step represents a test data step directory
type Step struct {
	Name      string
	Downloads []download.FileDownload
	Readme    string
	GitIgnore string
}

func main() {
	// Define flags
	var (
		showVersion bool
		showHelp    bool
		debug       bool
		destPath    string
	)

	pflag.BoolVarP(&showVersion, "version", "V", false, "Show version and exit")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show this help message")
	pflag.BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	pflag.StringVar(&destPath, "dest", "", "Destination directory (default: from config or $work/git/git_lfs_test_data)")

	pflag.Parse()

	// Handle version
	if showVersion {
		fmt.Printf("lfst-testdata version %s\n", version)
		os.Exit(0)
	}

	// Handle help
	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Check dependencies
	if err := checkDependencies(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Determine destination directory
	if destPath == "" {
		cfg, err := config.Load()
		if err == nil && cfg.TestDataPath != "" {
			destPath = cfg.GetTestDataPath()
		} else if workDir := os.Getenv("work"); workDir != "" {
			destPath = filepath.Join(workDir, "git", "git_lfs_test_data")
		} else {
			fmt.Fprintf(os.Stderr, "Error: destination directory not specified.\n")
			fmt.Fprintf(os.Stderr, "Use --dest flag, set LFS_TEST_DATA environment variable,\n")
			fmt.Fprintf(os.Stderr, "or define test_data in ~/.lfs-test-config\n")
			os.Exit(1)
		}
	}

	if debug {
		fmt.Printf("Destination directory: %s\n", destPath)
	}

	// Define test data steps
	steps := []Step{
		{
			Name: "step1",
			GitIgnore: `.cksum_output
`,
			Readme: "This is README.md for step 1\n",
			Downloads: []download.FileDownload{
				{
					URL:      "https://download.blender.org/peach/bigbuckbunny_movies/BigBuckBunny_640x360.m4v",
					FileName: "video1.m4v",
				},
				{
					URL:      "https://download.blender.org/peach/bigbuckbunny_movies/big_buck_bunny_480p_h264.mov",
					FileName: "video2.mov",
				},
				{
					URL:      "https://download.blender.org/peach/bigbuckbunny_movies/big_buck_bunny_480p_stereo.avi",
					FileName: "video3.avi",
				},
				{
					URL:      "https://download.blender.org/peach/bigbuckbunny_movies/big_buck_bunny_720p_stereo.ogg",
					FileName: "video4.ogg",
				},
				{
					URL:      "https://mattmahoney.net/dc/enwik9.zip",
					FileName: "zip1.zip",
				},
				{
					URL:      "https://www.gutenberg.org/cache/epub/feeds/rdf-files.tar.zip",
					FileName: "zip2.zip",
				},
				{
					URL:      "https://files.testfile.org/PDF/100MB-TESTFILE.ORG.pdf",
					FileName: "pdf1.pdf",
				},
			},
		},
		{
			Name:   "step2",
			Readme: "This is README.md for step 2\n",
			Downloads: []download.FileDownload{
				{
					URL:      "http://ipv4.download.thinkbroadband.com/200MB.zip",
					FileName: "zip1.zip",
				},
				{
					URL:      "https://download.blender.org/peach/bigbuckbunny_movies/big_buck_bunny_720p_h264.mov",
					FileName: "video2.mov",
				},
				{
					URL:      "https://download.blender.org/peach/bigbuckbunny_movies/big_buck_bunny_720p_stereo.avi",
					FileName: "video3.avi",
				},
				{
					URL:      "https://files.testfile.org/PDF/200MB-TESTFILE.ORG.pdf",
					FileName: "pdf1.pdf",
				},
			},
		},
		{
			Name:      "step3",
			Readme:    "This is README.md for step 3\n",
			Downloads: []download.FileDownload{},
		},
	}

	// Download files for each step
	for _, step := range steps {
		stepDir := filepath.Join(destPath, step.Name)

		if debug {
			fmt.Printf("\nProcessing %s/\n", step.Name)
		}

		// Create step directory
		if err := os.MkdirAll(stepDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", stepDir, err)
			os.Exit(1)
		}

		// Write .gitignore if specified
		if step.GitIgnore != "" {
			gitignorePath := filepath.Join(stepDir, ".gitignore")
			if err := os.WriteFile(gitignorePath, []byte(step.GitIgnore), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing .gitignore: %v\n", err)
				os.Exit(1)
			}
		}

		// Write README.md
		readmePath := filepath.Join(stepDir, "README.md")
		if err := os.WriteFile(readmePath, []byte(step.Readme), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing README.md: %v\n", err)
			os.Exit(1)
		}

		// Download files
		for _, dl := range step.Downloads {
			destFile := filepath.Join(stepDir, dl.FileName)
			_, err := download.DownloadFile(dl.URL, destFile, debug)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", dl.FileName, err)
				os.Exit(1)
			}
		}
	}

	// Print summary
	fmt.Printf("\nDownload complete.\n")
	fmt.Printf("Here are lists of the initial set of downloaded files and sizes,\n")
	fmt.Printf("ordered by name. The last line in each listing shows the total\n")
	fmt.Printf("size of the files.\n\n")
	fmt.Printf("Some files might be deleted by each step; those are not shown here.\n\n")

	// Show disk usage for each step
	for _, step := range steps {
		stepDir := filepath.Join(destPath, step.Name)
		fmt.Printf("\n%s:\n", step.Name)
		showDiskUsage(stepDir)
	}
}

func printHelp() {
	fmt.Printf("lfst-testdata - Download Git LFS test data files\n\n")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Printf("DESCRIPTION:\n")
	fmt.Printf("  Downloads test data files for Git LFS evaluation. Creates three step\n")
	fmt.Printf("  directories (step1, step2, step3) with test files of various sizes.\n")
	fmt.Printf("  Files are downloaded from Big Buck Bunny, Project Gutenberg, and other\n")
	fmt.Printf("  public sources. Total download size is approximately 2.5 GB.\n\n")

	fmt.Printf("USAGE:\n")
	fmt.Printf("  lfst-testdata [OPTIONS]\n\n")

	fmt.Printf("OPTIONS:\n")
	fmt.Printf("  -h, --help         Show this help message\n")
	fmt.Printf("  -V, --version      Show version\n")
	fmt.Printf("  -d, --debug        Enable debug output\n")
	fmt.Printf("  --dest PATH        Destination directory (default: from config)\n\n")

	fmt.Printf("CONFIGURATION:\n")
	fmt.Printf("  The destination directory is determined in this order:\n")
	fmt.Printf("  1. --dest flag\n")
	fmt.Printf("  2. test_data from ~/.lfs-test-config\n")
	fmt.Printf("  3. $work/git/git_lfs_test_data (if $work is set)\n\n")

	fmt.Printf("EXAMPLES:\n")
	fmt.Printf("  # Download to default location\n")
	fmt.Printf("  lfst-testdata\n\n")

	fmt.Printf("  # Download to specific directory\n")
	fmt.Printf("  lfst-testdata --dest /tmp/lfs-test-data\n\n")

	fmt.Printf("  # Download with debug output\n")
	fmt.Printf("  lfst-testdata --debug\n\n")

	fmt.Printf("DOCUMENTATION:\n")
	fmt.Printf("  https://www.mslinn.com/git/5600-git-lfs-evaluation.html#git_lfs_test_data\n\n")
}

func checkDependencies() error {
	// Check for curl (used as fallback in download package)
	if _, err := exec.LookPath("curl"); err != nil {
		return fmt.Errorf("curl is required but not found in PATH")
	}
	return nil
}

func showDiskUsage(dir string) {
	// Use du command to show disk usage (similar to original bash script)
	cmd := exec.Command("du", "-ah", dir)
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running du: %v\n", err)
		return
	}

	// Filter output to remove leading "./" from paths
	lines := string(output)
	fmt.Print(lines)
}
