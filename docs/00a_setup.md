# Setup and Test Execution

Clients automatically detect they're remote and SSH their data to `gojira`.
All commands work from anywhere with SSH access.


## One-Time Setup

### On Git LFS Server (gojira)

We cannot do this yet because no release has been published to date:

```shell
go install github.com/mslinn/git-lfs-test/cmd/...@latest
```

Do this instead for now:

```shell
$ cd /work/git/git-lfs-test
$ make install
```

This installs all commands to `/usr/local/bin/`:

- `lfst-checksum`  - Compute and store checksums
- `lfst-config`    - Manage configuration
- `lfst-eval-repo` - Create a standard test repository
- `lfst-import`    - Import checksum JSON data
- `lfst-query`     - Query and report on test data
- `lfst-run`       - Manage test run lifecycle
- `lfst-scenario`  - Execute complete 7-step test scenarios
- `lfst-testdata`  - Create test data

The database is created automatically at first use at
`gojira:/home/mslinn/lfs_eval/lfs-test.db`

### On Clients (Bear and Camille)

Includes the same installation as for the server:

```shell
$ cd /mnt/f/work/git/git-lfs-test  # or appropriate path
$ make install
```

Configure `~/.lfs-test-config`:

```shell
$ cat > ~/.lfs-test-config <<EOF
auto_remote: true
database: /home/mslinn/lfs_eval/lfs-test.db
remote_host: gojira
test_data: $work/git/git_lfs_test_data
EOF
```

**Important:** The `test_data` path uses shell variable expansion. You must set `export work=/your/base/path` in your shell environment, or commands will fail.


#### GitHub Scenario Setup

All GitHub-based scenarios, for example Scenario 7, create a private GitHub
repository using the `gh` CLI.

Install and authenticate with GitHub CLI:

```shell
# Install gh (if not already installed)
$ sudo apt install gh

# Authenticate with GitHub
$ gh auth login
```


### Environment Variables

The test commands use several environment variables for configuration:

**Required for recommended configuration:**

- `work` - Base directory for test data (required if using `$work/git/git_lfs_test_data` pattern)


**Configuration (can also be set in config file):**

- `LFS_TEST_DATA` - Location of test data directory (overrides `test_data` in config file; recommended: `$work/git/git_lfs_test_data`)


**Optional (override config file):**

- `LFS_TEST_DB` - Database path (overrides `database` in config file)
- `LFS_TEST_CONFIG` - Path to config file (default: `~/.lfs-test-config`)
- `LFS_REMOTE_HOST` - Remote host for SSH operations (overrides `remote_host` in config file)
- `LFS_AUTO_REMOTE` - Enable auto-remote detection: `true`/`1` or `false`/`0` (overrides `auto_remote` in config file)


**Setting environment variables:**

```shell
# Required: Add to ~/.bashrc or ~/.zshrc
export work=/mnt/f/work  # or your preferred base directory

# Recommended: Use $work variable for test data
export LFS_TEST_DATA=$work/git/git_lfs_test_data

# Optional overrides
export LFS_TEST_DB=/path/to/custom/database.db
export LFS_REMOTE_HOST=myserver
```

For more information on environment variables and directory organization, see:
https://www.mslinn.com/git/5600-git-lfs-evaluation.html


### Test Data Requirements

The test scenarios require **2.4GB of real large files** (103M-398M each).
These must be available on the machine running `lfst-scenario`.

Check if test data exists:

```shell
$ ls -lh $LFS_TEST_DATA/v1/
total 1.3G
-rw-r--r-- 1 mslinn mslinn 103M Jan 23  2025 pdf1.pdf
-rw-r--r-- 1 mslinn mslinn 116M Jan 23  2025 video1.m4v
-rw-r--r-- 1 mslinn mslinn 238M Jan 23  2025 video2.mov
-rw-r--r-- 1 mslinn mslinn 150M Jan 23  2025 video3.avi
-rw-r--r-- 1 mslinn mslinn 188M Jan 23  2025 video4.ogg
-rw-r--r-- 1 mslinn mslinn 308M Jan 23  2025 zip1.zip
-rw-r--r-- 1 mslinn mslinn 154M Jan 23  2025 zip2.zip
```

If not present, the test data can be downloaded using the [`git_lfs_test_data` script](https://www.mslinn.com/git/5600-git-lfs-evaluation.html#git_lfs_test_data).
