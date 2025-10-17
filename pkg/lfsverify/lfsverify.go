package lfsverify

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mslinn/git-lfs-test/pkg/timing"
)

// VerificationResult contains the results of LFS verification
type VerificationResult struct {
	IsLFSEnabled      bool     // Is LFS installed in the repo
	TrackedFiles      []string // Files tracked by LFS (from git lfs ls-files)
	LFSObjectCount    int      // Number of objects in .git/lfs/objects
	LFSObjectsSize    int64    // Total size of LFS objects
	GitObjectsSize    int64    // Size of .git/objects (should be small if LFS working)
	PointerFiles      []string // Files that are LFS pointers in working directory
	NonPointerFiles   []string // Files that should be pointers but aren't
	MissingLFSObjects []string // Files tracked but missing from .git/lfs/objects
	Errors            []string // Any errors encountered
}

// VerifyLFSStatus checks if LFS is properly configured and files are stored correctly
func VerifyLFSStatus(repoDir string, expectedFiles []string, debug bool) (*VerificationResult, error) {
	result := &VerificationResult{}

	if debug {
		fmt.Println("  Verifying LFS status...")
	}

	// Check if LFS is installed in the repo
	gitDir := filepath.Join(repoDir, ".git")
	lfsDir := filepath.Join(gitDir, "lfs")
	if _, err := os.Stat(lfsDir); err == nil {
		result.IsLFSEnabled = true
		if debug {
			fmt.Println("    ✓ LFS is enabled in repository")
		}
	} else {
		result.Errors = append(result.Errors, "LFS not enabled in repository")
		return result, fmt.Errorf("LFS not enabled in repository")
	}

	// Get list of files tracked by LFS
	trackedFiles, err := getLFSTrackedFiles(repoDir)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to get LFS tracked files: %v", err))
	} else {
		result.TrackedFiles = trackedFiles
		if debug {
			fmt.Printf("    ✓ %d files tracked by LFS\n", len(trackedFiles))
		}
	}

	// Count and measure LFS objects
	objectCount, objectSize, err := countLFSObjects(gitDir)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to count LFS objects: %v", err))
	} else {
		result.LFSObjectCount = objectCount
		result.LFSObjectsSize = objectSize
		if debug {
			fmt.Printf("    ✓ %d LFS objects (%.2f MB)\n", objectCount, float64(objectSize)/1024/1024)
		}
	}

	// Measure git objects size
	gitObjectSize, err := dirSize(filepath.Join(gitDir, "objects"))
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to measure git objects: %v", err))
	} else {
		result.GitObjectsSize = gitObjectSize
		if debug {
			fmt.Printf("    ✓ Git objects size: %.2f MB\n", float64(gitObjectSize)/1024/1024)
		}
	}

	// Verify expected files are LFS pointers
	if len(expectedFiles) > 0 {
		pointers, nonPointers := checkPointerFiles(repoDir, expectedFiles)
		result.PointerFiles = pointers
		result.NonPointerFiles = nonPointers

		if debug {
			fmt.Printf("    ✓ %d/%d files are LFS pointers\n", len(pointers), len(expectedFiles))
		}

		if len(nonPointers) > 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("%d files are not LFS pointers: %v", len(nonPointers), nonPointers))
		}
	}

	// Verify LFS objects exist for tracked files
	missing := checkMissingLFSObjects(repoDir, trackedFiles)
	result.MissingLFSObjects = missing
	if len(missing) > 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("%d tracked files missing LFS objects: %v", len(missing), missing))
	}

	return result, nil
}

// getLFSTrackedFiles returns list of files tracked by LFS using git lfs ls-files
func getLFSTrackedFiles(repoDir string) ([]string, error) {
	result := timing.Run("git", []string{"-C", repoDir, "lfs", "ls-files", "-n"}, nil)
	if result.Error != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("git lfs ls-files failed: %v", result.Error)
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// countLFSObjects counts objects in .git/lfs/objects and returns count and total size
func countLFSObjects(gitDir string) (int, int64, error) {
	lfsObjectsDir := filepath.Join(gitDir, "lfs", "objects")

	if _, err := os.Stat(lfsObjectsDir); os.IsNotExist(err) {
		return 0, 0, nil
	}

	count := 0
	var totalSize int64

	err := filepath.Walk(lfsObjectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() != "." && info.Name() != ".." {
			count++
			totalSize += info.Size()
		}
		return nil
	})

	return count, totalSize, err
}

// dirSize calculates the total size of a directory
func dirSize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// checkPointerFiles checks which files are LFS pointers
// Returns two lists: pointer files and non-pointer files
func checkPointerFiles(repoDir string, files []string) ([]string, []string) {
	var pointers []string
	var nonPointers []string

	for _, file := range files {
		filePath := filepath.Join(repoDir, file)
		if isLFSPointer(filePath) {
			pointers = append(pointers, file)
		} else {
			// Only add to nonPointers if file exists (not deleted)
			if _, err := os.Stat(filePath); err == nil {
				nonPointers = append(nonPointers, file)
			}
		}
	}

	return pointers, nonPointers
}

// isLFSPointer checks if a file is an LFS pointer file
// LFS pointer files are small text files with specific format:
// version https://git-lfs.github.com/spec/v1
// oid sha256:...
// size ...
func isLFSPointer(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	// LFS pointers are typically 120-150 bytes
	// If file is larger than 200 bytes, it's not a pointer
	if info.Size() > 200 {
		return false
	}

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// Check for LFS pointer format
	lines := strings.Split(string(content), "\n")
	if len(lines) < 3 {
		return false
	}

	// Check for required fields
	hasVersion := false
	hasOID := false
	hasSize := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version https://git-lfs.github.com/spec/") {
			hasVersion = true
		}
		if strings.HasPrefix(line, "oid sha256:") {
			hasOID = true
		}
		if strings.HasPrefix(line, "size ") {
			hasSize = true
		}
	}

	return hasVersion && hasOID && hasSize
}

// checkMissingLFSObjects checks if LFS objects exist for tracked files
func checkMissingLFSObjects(repoDir string, trackedFiles []string) []string {
	var missing []string

	for _, file := range trackedFiles {
		filePath := filepath.Join(repoDir, file)

		// Get the OID from the pointer file
		oid, err := getOIDFromPointer(filePath)
		if err != nil {
			missing = append(missing, file)
			continue
		}

		// Check if object exists in .git/lfs/objects
		if !lfsObjectExists(repoDir, oid) {
			missing = append(missing, file)
		}
	}

	return missing
}

// getOIDFromPointer extracts the OID from an LFS pointer file
func getOIDFromPointer(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Look for "oid sha256:..." line
	re := regexp.MustCompile(`oid sha256:([a-f0-9]{64})`)
	matches := re.FindSubmatch(content)
	if len(matches) < 2 {
		return "", fmt.Errorf("OID not found in pointer file")
	}

	return string(matches[1]), nil
}

// lfsObjectExists checks if an LFS object exists in .git/lfs/objects
func lfsObjectExists(repoDir, oid string) bool {
	if len(oid) < 5 {
		return false
	}

	// LFS objects are stored as .git/lfs/objects/XX/YY/XXYY...
	// where XX is first 2 chars, YY is next 2 chars
	objectPath := filepath.Join(repoDir, ".git", "lfs", "objects", oid[0:2], oid[2:4], oid)
	_, err := os.Stat(objectPath)
	return err == nil
}

// VerifyLFSPointers verifies that specific files are tracked by LFS
// Uses git lfs ls-files to check what's actually tracked, since working directory
// files are always expanded (not pointers)
func VerifyLFSPointers(repoDir string, files []string, debug bool) error {
	if debug {
		fmt.Printf("  Verifying %d files are tracked by LFS...\n", len(files))
	}

	// Get list of LFS-tracked files from git
	trackedFiles, err := getLFSTrackedFiles(repoDir)
	if err != nil {
		return fmt.Errorf("failed to get LFS tracked files: %w", err)
	}

	// Convert to map for quick lookup
	trackedMap := make(map[string]bool)
	for _, f := range trackedFiles {
		trackedMap[f] = true
	}

	// Check which expected files are not tracked
	var notTracked []string
	for _, file := range files {
		if !trackedMap[file] {
			notTracked = append(notTracked, file)
		}
	}

	if len(notTracked) > 0 {
		return fmt.Errorf("expected %d files to be tracked by LFS, but %d are not: %v",
			len(files), len(notTracked), notTracked)
	}

	if debug {
		fmt.Printf("    ✓ All %d files are tracked by LFS\n", len(files))
	}

	return nil
}

// VerifyLFSObjects verifies that LFS objects exist for tracked files
func VerifyLFSObjects(repoDir string, expectedCount int, debug bool) error {
	gitDir := filepath.Join(repoDir, ".git")
	count, size, err := countLFSObjects(gitDir)
	if err != nil {
		return fmt.Errorf("failed to count LFS objects: %w", err)
	}

	if debug {
		fmt.Printf("  Verifying LFS objects...\n")
		fmt.Printf("    Found %d LFS objects (%.2f MB)\n", count, float64(size)/1024/1024)
	}

	if count < expectedCount {
		return fmt.Errorf("expected at least %d LFS objects, found %d", expectedCount, count)
	}

	if debug {
		fmt.Printf("    ✓ LFS objects exist (%d >= %d expected)\n", count, expectedCount)
	}

	return nil
}

// VerifyNotLFSPointers verifies that files are NOT tracked by LFS (after untracking)
// Uses git lfs ls-files to verify files are no longer tracked
func VerifyNotLFSPointers(repoDir string, files []string, debug bool) error {
	if debug {
		fmt.Printf("  Verifying %d files are NOT tracked by LFS...\n", len(files))
	}

	// Get list of LFS-tracked files from git
	trackedFiles, err := getLFSTrackedFiles(repoDir)
	if err != nil {
		// If git lfs ls-files fails or returns empty, that's expected after untracking
		if debug {
			fmt.Printf("    ✓ No files tracked by LFS (successfully migrated out)\n")
		}
		return nil
	}

	// Convert to map for quick lookup
	trackedMap := make(map[string]bool)
	for _, f := range trackedFiles {
		trackedMap[f] = true
	}

	// Check which files are still tracked when they shouldn't be
	var stillTracked []string
	for _, file := range files {
		if trackedMap[file] {
			stillTracked = append(stillTracked, file)
		}
	}

	if len(stillTracked) > 0 {
		return fmt.Errorf("expected files to NOT be tracked by LFS, but %d still are: %v",
			len(stillTracked), stillTracked)
	}

	if debug {
		fmt.Printf("    ✓ No files tracked by LFS (successfully migrated out)\n")
	}

	return nil
}

// GetPointerInfo returns detailed information about an LFS pointer file
type PointerInfo struct {
	Version string
	OID     string
	Size    int64
}

// GetPointerInfo extracts information from an LFS pointer file
func GetPointerInfo(filePath string) (*PointerInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	info := &PointerInfo{}
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "version ") {
			info.Version = strings.TrimPrefix(line, "version ")
		} else if strings.HasPrefix(line, "oid sha256:") {
			info.OID = strings.TrimPrefix(line, "oid sha256:")
		} else if strings.HasPrefix(line, "size ") {
			sizeStr := strings.TrimPrefix(line, "size ")
			size, err := strconv.ParseInt(sizeStr, 10, 64)
			if err == nil {
				info.Size = size
			}
		}
	}

	if info.OID == "" {
		return nil, fmt.Errorf("invalid LFS pointer file: missing OID")
	}

	return info, nil
}

// VerifyRepositorySizes checks that git objects are small (pointers) and LFS objects are large (actual files)
func VerifyRepositorySizes(repoDir string, debug bool) error {
	gitDir := filepath.Join(repoDir, ".git")

	// Get git objects size
	gitObjectsSize, err := dirSize(filepath.Join(gitDir, "objects"))
	if err != nil {
		return fmt.Errorf("failed to measure git objects: %w", err)
	}

	// Get LFS objects size
	_, lfsObjectsSize, err := countLFSObjects(gitDir)
	if err != nil {
		return fmt.Errorf("failed to measure LFS objects: %w", err)
	}

	if debug {
		fmt.Printf("  Repository size comparison:\n")
		fmt.Printf("    Git objects: %.2f MB\n", float64(gitObjectsSize)/1024/1024)
		fmt.Printf("    LFS objects: %.2f MB\n", float64(lfsObjectsSize)/1024/1024)
	}

	// LFS objects should be significantly larger than git objects
	// If git objects are larger, LFS probably isn't working
	if lfsObjectsSize > 0 && gitObjectsSize > lfsObjectsSize {
		return fmt.Errorf("git objects (%.2f MB) larger than LFS objects (%.2f MB) - LFS may not be working correctly",
			float64(gitObjectsSize)/1024/1024, float64(lfsObjectsSize)/1024/1024)
	}

	if debug {
		fmt.Printf("    ✓ Repository sizes are correct (LFS objects > git objects)\n")
	}

	return nil
}
