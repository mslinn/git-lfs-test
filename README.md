# Git LFS Test

Comprehensive testing framework for evaluating Git LFS server implementations.

This framework automates the evaluation of various Git LFS servers through a standardized 7-step test procedure, with full checksum verification and performance tracking.


## Overview

The Git LFS Test framework was developed as part of the [Git LFS evaluation series on mslinn.com](https://www.mslinn.com/git/5100-git-lfs.html). It provides automated testing for:

- **Bare Git repositories** (local and SSH)
- **LFS Test Server** (HTTP and GitHub)
- **Giftless** (local and SSH)
- **Rudolfs** (local and SSH)


## Features

- **Automated 7-step test scenarios** covering the complete Git LFS workflow
- **Remote test data access** via SSH/rsync (no need for local copies)
- **Checksum verification** at each step using CRC32
- **GitHub integration** for testing with real GitHub repositories
- **Configurable database** location (SQLite with WAL mode)
- **Performance tracking** with millisecond precision
- **Comprehensive unit tests** for all core functionality


## Installation

### Prerequisites

- Go 1.24.2 or later
- Git with Git LFS installed
- SSH access for remote test data (optional)
- GitHub CLI (`gh`) for GitHub scenarios (optional)

### Install from source

```shell
$ cd /mnt/f/work/git
$ git clone https://github.com/mslinn/git-lfs-test.git
$ cd git-lfs-test
$ make install
```

This installs all commands to `/usr/local/bin/`:

- `lfst` - Unified command (dispatches to individual tools)
- `lfst-scenario` - Execute complete 7-step test scenarios
- `lfst-checksum` - Compute and store checksums
- `lfst-import` - Import checksum JSON data
- `lfst-run` - Manage test run lifecycle
- `lfst-query` - Query and report on test data
- `lfst-config` - Manage configuration

You can use either the unified `lfst` command:
```shell
$ lfst config show
$ lfst scenario --list
```

Or the individual commands directly:
```shell
$ lfst-config show
$ lfst-scenario --list
```


## Quick Start

1. **Configure the test environment:**

First, set the `work` environment variable (required for the recommended test data location):

```shell
$ export work=/mnt/f/work  # or your preferred base directory
```

Then configure the framework:

```shell
$ lfst config init
Created config file: /home/mslinn/.lfs-test-config

$ lfst config set database /path/to/your/test.db
$ lfst config set remote_host your-server
$ lfst config set test_data $work/git/git_lfs_test_data
$ lfst config show
```

**Important:** If you use `$work/git/git_lfs_test_data` as your test data path, the `work` environment variable must be set, or commands will fail with an error.

2. **Set up test data:**

You need 2.4GB of test files. The test data location can be configured:
- In the config file: `lfst config set test_data $work/git/git_lfs_test_data` (recommended)
- Via environment variable: `export LFS_TEST_DATA=$work/git/git_lfs_test_data`
- For remote access via SSH: `export LFS_TEST_DATA=server:$work/git/git_lfs_test_data`

Recommended location: `$work/git/git_lfs_test_data` (requires `work` env var to be set)

3. **List available scenarios:**

```shell
$ lfst scenario --list
Available scenarios:

ID  Server             Protocol  Git Server  Description
--  ------             --------  ----------  -----------
1   bare               local     bare        Bare repo - local
2   bare               ssh       bare        Bare repo - SSH
6   lfs-test-server    http      bare        LFS Test Server - HTTP
7   lfs-test-server    http      github      LFS Test Server - HTTP/GitHub
...
```

4. **Run a test scenario:**

```shell
$ lfst scenario -d 6

=== Executing Scenario 6: LFS Test Server - HTTP ===
Server: lfs-test-server via http
Work directory: /tmp/lfst

Created test run ID: 1

--- Step 1 ---
Initializing repository...
  ✓ Initialized in 15ms
...
✓ Scenario 6 completed successfully
```

5. **View results:**

```shell
$ lfst run show 1
$ lfst query checksums --run-id 1 --step 1
$ lfst query stats --run-id 1
```


## Test Scenarios

The framework executes a standardized 7-step test procedure:

1. **Setup**: Create repository, install Git LFS, configure tracking patterns, copy initial test files (1.3GB)
2. **Initial Push**: Add and commit all files, verify checksums
3. **Modifications**: Update 4 files, delete 2 files, rename 1 file, commit changes
4. **Second Clone**: Clone repository to new location, verify checksums match
5. **Second Client Push**: Create new file in second clone, commit and push
6. **First Client Pull**: Pull changes from remote, verify checksums
7. **Untrack**: Remove files from LFS tracking, migrate back to regular Git

For detailed documentation, see [history/scenario1.md](history/scenario1.md).


## Configuration

### Configuration File

The framework uses `~/.lfs-test-config` (YAML format):

```yaml
database: /home/mslinn/lfs_eval/lfs-test.db
remote_host: gojira
auto_remote: true
test_data: $work/git/git_lfs_test_data
work_dir: /tmp/lfst
```

**Note:** The `test_data` and `work_dir` paths can use shell variable expansion. If using `$work/git/git_lfs_test_data`, you must set `export work=/your/base/path` in your shell environment, or commands will fail.

### Environment Variables

Environment variables override config file settings:

- `work` - Base directory for test data (required if using `$work/git/git_lfs_test_data` pattern)
- `LFS_TEST_DATA` - Location of test data directory (overrides `test_data` in config file; recommended: `$work/git/git_lfs_test_data`)
- `LFS_TEST_DB` - Database path (overrides `database` in config file)
- `LFS_TEST_CONFIG` - Path to config file (default: `~/.lfs-test-config`)
- `LFS_REMOTE_HOST` - Remote host for SSH operations (overrides `remote_host` in config file)
- `LFS_AUTO_REMOTE` - Enable auto-remote detection: `true`/`1` or `false`/`0` (overrides `auto_remote` in config file)
- `LFS_WORK_DIR` - Working directory for test execution (overrides `work_dir` in config file; default: `/tmp/lfst`)


### Command-line Flags

All commands accept a `--db` flag to override the database location:

```shell
$ lfst scenario --db $HOME/my-test.db 6
```


## Development

### Build

```shell
$ make build
```

### Run tests

```shell
$ go test ./...
```

### Run tests with coverage

```shell
$ go test -cover ./...
```

### Cancel tests

Cancel a specific test:

```shell
$ lfst scenario --cancel 1
```

Or cancel all running tests:

```shell
$ lfst scenario --cancel all
```

The lfst framework supports canceling tests with this mechanism:

  1. Each test run now stores its process ID (`pid`) when it starts
  2. Automatically adds the `pid` column to existing databases
  3. Implements and orderly shutdown procedure:

     - Sends `SIGTERM` to the process (graceful shutdown)
     - Waits 2 seconds
     - If process still running, sends `SIGKILL` (forceful)

  4. Removes working directories (`/tmp/lfst/repo1` and `/tmp/lfst/repo2`)
  5. Status Update: Marks run as `cancelled` in database with timestamp

Stale runs with processes that no longer exist are cancelled in the same way.

### Inspect repository details

View detailed repository contents for any test run:

```shell
$ lfst scenario --detail 1
```

This shows all files in both `repo1` and `repo2` with:
- **File name** - Relative path within the repository
- **Size** - Formatted as bytes, KB, MB, or GB
- **Storage type** - Where the file is stored:
  - `LFS (tracked)` - File tracked by Git LFS
  - `Git (regular)` - Regular Git object
  - `Untracked` - Not tracked by Git
  - `Ignored` - Matched by .gitignore

The output also includes a summary with total file count, total size, and counts for each storage type.

**Example output:**
```
Repository Details for Run 2
  Scenario: 1
  Status: completed
  Started: 2025-10-17 10:30:45

=== First Repository (repo1) ===
Location: /tmp/lfst/repo1

File                                                       Size  Storage
-------------------------------------------------- ------------  --------------------
.gitattributes                                             29 B  Git (regular)
README.md                                               1.23 KB  Git (regular)
pdf1.pdf                                              204.20 MB  Git (regular)
video2.mov                                            397.44 MB  Git (regular)
...

Summary:
  Total files: 7 (1.20 GB)
  LFS tracked: 0
  Git regular: 7
  Untracked:   0
  Ignored:     0
```

**Note:** The `--detail` option only works if the test repositories still exist in the work directory. If the test has been cancelled or the work directory has been cleaned up, the repositories will not be available for inspection.


## Architecture

The framework is organized into several packages:

- `pkg/checksum` - File checksumming with CRC32
- `pkg/config` - Configuration management
- `pkg/database` - SQLite database operations with WAL mode
- `pkg/git` - Git operations (clone, commit, push, pull)
- `pkg/scenario` - Test scenario execution logic
- `pkg/testdata` - Test file management with remote support
- `pkg/timing` - Command execution with timing


## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass (`go test ./...`)
5. Submit a pull request


## License

See the [articles on mslinn.com](https://www.mslinn.com/git/5100-git-lfs.html) for usage and license information.


## Support

For issues, questions, or feature requests, please open an issue on GitHub.


## Related Projects

- [Git LFS](https://git-lfs.github.com/) - Official Git Large File Storage extension
- [LFS Test Server](https://github.com/git-lfs/lfs-test-server) - Reference implementation
- [Giftless](https://github.com/datopian/giftless) - Python-based LFS server
- [Rudolfs](https://github.com/jasonwhite/rudolfs) - Rust-based LFS server with S3 backend
