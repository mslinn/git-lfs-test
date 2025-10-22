# Troubleshooting

## GitHub repository creation fails

```shell
Error: gh CLI not available - install with: sudo apt install gh
```

**Solution:** Install and authenticate with GitHub CLI:

```shell
$ sudo apt install gh
$ gh auth login
```

## Test data not found

```shell
Error: test data directory not found (searched: [/mnt/f/work/git/git_lfs_test_data /work/git/git_lfs_test_data ...])
```

**Solution:** Set the test data location in your config file or via environment variable:

```shell
# First, set the work environment variable (required)
export work=/mnt/f/work  # or your preferred base directory

# Option 1: Set in config file (recommended)
$ lfst-config set test_data $work/git/git_lfs_test_data

# Option 2: Set environment variable (add to ~/.bashrc or ~/.zshrc)
export LFS_TEST_DATA=$work/git/git_lfs_test_data
lfst-scenario 6
```

**Important:** If using `$work/git/git_lfs_test_data`, the `work` environment variable must be set, or commands will fail with an error.

The commands search for test data in these locations (highest priority first):

1. `$LFS_TEST_DATA` environment variable
2. `test_data` setting in `~/.lfs-test-config`
3. Hardcoded fallbacks: `/mnt/f/work/git/git_lfs_test_data`,
  `/work/git/git_lfs_test_data`, `/home/mslinn/git_lfs_test_data`

## Remote mode not working

If checksums aren't being sent to gojira:

```shell
# Force local mode for testing
lfst-checksum --local --run-id 1 --step 1 --dir /path/to/repo

# Force remote mode with specific host
lfst-checksum --remote gojira --run-id 1 --step 1 --dir /path/to/repo

# Check SSH connectivity
ssh gojira "which lfst-import"
```

## Database location

Commands search for the database in this order (highest priority first):

1. `--db` command-line flag
2. `$LFS_TEST_DB` environment variable
3. `database` setting in `~/.lfs-test-config` file
4. `$LFS_TEST_CONFIG` environment variable (alternate config file path)
5. Default: `/home/mslinn/lfs_eval/lfs-test.db`

**Example:**

```shell
# Use environment variable
export LFS_TEST_DB=/path/to/my-test.db

# Or use command-line flag
lfst-scenario --db /path/to/my-test.db 6
```
