# Running a Test Scenario

## Automated Execution (Recommended)

The `lfst-scenario` command automates the entire 7-step test procedure. All steps are fully implemented, including GitHub repository creation (using `gh` CLI), `.lfsconfig` file generation, evaluation README generation, and automatic cleanup on failure. Use the `-f` flag to force recreation of existing repositories.

**Repository naming:** The test creates two local repository clones to simulate multiple clients:

- `repo1` - The first repository clone (created in Step 1, used in Steps 1-3 and 6-7)
- `repo2` - The second repository clone (created in Step 4, used in Steps 4-5)

Both repositories are created in a temporary work directory (e.g., `/tmp/lfst/`).

```shell
# List available scenarios
$ lfst-scenario --list
Available scenarios:

ID  Server             Protocol  Git Server  Description
--  ------             --------  ----------  -----------
1   bare               local     bare        Bare repo - local
2   bare               ssh       bare        Bare repo - SSH
6   lfs-test-server    http      bare        LFS Test Server - HTTP
7   lfs-test-server    http      github      LFS Test Server - HTTP/GitHub
8   giftless           local     bare        Giftless - local
9   giftless           ssh       bare        Giftless - SSH
13  rudolfs            local     bare        Rudolfs - local
14  rudolfs            ssh       bare        Rudolfs - SSH
```

## What Each Step Does

1. **Setup**:
   - Create local Git repository
   - Configure Git user (LFS Test `<test@example.com>`)
   - Create GitHub repository (for scenarios with GitHub Git server) using `gh` CLI
   - Add GitHub remote as `origin`
   - Install `git-lfs` hooks
   - Create `.lfsconfig` with LFS server URL (if applicable)
   - Configure LFS tracking patterns, e.g. `\*.pdf`, `\*.mov`, `\*.avi`, `\*.ogg`, `\*.m4v`, `\*.zip`
   - Generate evaluation `README.md` with scenario details
   - Copy 1.3GB test files from `v1/`
   - Compute and store checksums
2. **Initial Push**: Add all files to Git, commit with message "Initial commit with LFS files", verify checksums match step 1
3. **Modifications**:
   - Update 4 files with v2 versions (`pdf1.pdf`, `video2.mov`, `video3.avi`, `zip1.zip`)
   - Delete 2 files (`video1.m4v`, `video4.ogg`)
   - Rename 1 file (`zip2.zip` → `zip2_renamed.zip`)
   - Add all changes, commit with message "Update, delete, and rename files (v2)", compute checksums
4. **Second Clone**: Clone repository to repo2 directory, compute checksums, compare with step 3 (must match)
5. **Second Client Push**: Create README.md in second clone, add and commit with message "Add README from second client", compute checksums
6. **First Client Pull**: Pull changes from remote (when configured), compute checksums in first clone (should match step 5 after successful pull)
7. **Untrack**: Untrack all patterns from LFS, run git lfs migrate export, commit .gitattributes changes, compute final checksums (files now in regular git)


## Viewing Results

### Show test run details

From anywhere (clients or server):

```shell
$ lfst-run show 1
Test Run 1:
  Scenario ID:  6
  Server Type:  lfs-test-server
  Protocol:     http
  Git Server:   bare
  Status:       completed
  Started:      2025-10-16 15:30:00
  Completed:    2025-10-16 15:35:23
  Duration:     323.45s
  Notes:        Automated execution of scenario 6 | All steps completed successfully
```

### View checksums for a specific step

```shell
$ lfst-query checksums --run-id 1 --step 1
Checksums for run 1, step 1:

CRC32             Size        Path
-----             ----        ----
a1b2c3d4          103.00 MB   pdf1.pdf
e5f6g7h8          116.00 MB   video1.m4v
... (more checksums)
```

### Compare checksums between steps

```shell
$ lfst-query compare --run-id 1 --from 1 --to 2
Changes from step 1 to step 2:

  No differences found
```

### View statistics

```shell
$ lfst-query stats --run-id 1
Test Run 1 Statistics:

  Scenario:     6
  Server:       lfs-test-server
  Protocol:     http
  Status:       completed

  Checksums per step:
    Step 1: 7 checksums
    Step 2: 7 checksums

  Operations per step:
    Step 1: 9 operations (avg 156.3ms)
    Step 2: 2 operations (avg 1904.0ms)
```

### List all runs

```shell
$ lfst-run list --status completed
ID  Scenario  Server           Protocol  Git   Status     Started   Duration  Notes
--  --------  ------           --------  ---   ------     -------   --------  -----
1   6         lfs-test-server  http      bare  completed  15:30:00  323.5s    Automated execution...
```


## Manual Execution (Advanced)

For debugging or custom workflows, you can run individual commands:

```shell
# Create a test run manually
$ lfst-run create --scenario 6 --server lfs-test-server --protocol http
Created test run ID: 2

# Compute and store checksums for a specific directory and step
$ lfst-checksum --run-id 2 --step 1 --dir /path/to/repo
Computed 7 checksums
✓ Stored 7 checksums on gojira for step 1

# Mark run as completed
$ lfst-run complete 2 --notes "Manual test completed"
✓ Test run 2 marked as completed (45.23s)
```


## Running GitHub Scenarios

The `lfst-scenario` command runs a test scenario.

```shell
# First run - creates new GitHub repository
$ lfst-scenario 7
```

The command will:

1. Create private GitHub repository (e.g., `mslinn/lfs-eval-test`)
2. Add it as 'origin' remote
3. Configure `.lfsconfig` with LFS server URL
4. Execute all 7 test steps

The `-f` flag deletes and recreates the GitHub repository.

```shell
# Subsequent runs - use force flag to recreate repository
$ lfst-scenario -f 7
```
