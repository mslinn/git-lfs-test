# Running Test Scenario 6

```shell
# Run scenario 6 (LFS Test Server - HTTP) with debug output
$ lfst-scenario -d 6

=== Executing Scenario 6: LFS Test Server - HTTP ===
Server: lfs-test-server via http
Work directory: /tmp/lfst

Created test run ID: 1

--- Step 1 ---
Initializing repository...
  ✓ Initialized in 15ms
Installing git-lfs...
  ✓ Installed git-lfs in 234ms
Configuring LFS tracking patterns...
  ✓ Tracked *.pdf in 12ms
  ✓ Tracked *.mov in 11ms
  ... (more patterns)
Copying initial test files (v1 - 1.3GB)...
Copying 7 test files to /tmp/lfst/repo1
  Copying pdf1.pdf (103.00 MB)
  Copying video1.m4v (116.00 MB)
  ... (more files)
  ✓ Copied 7 files
Computing checksums...
Stored 7 checksums
✓ Step 1 complete

--- Step 2 ---
Adding files to git...
  ✓ Added in 3241ms
Committing initial files...
  ✓ Committed in 567ms
Stored 7 checksums for step 2
✓ Step 2 complete

--- Step 3 ---
Updating files with v2 versions...
  Copying pdf1.pdf (205.00 MB)
  Copying video2.mov (398.00 MB)
  Copying video3.avi (272.00 MB)
  Copying zip1.zip (200.00 MB)
  ✓ Copied 4 files
Deleting files...
  Deleting video1.m4v
  Deleting video4.ogg
Renaming files...
  Renaming zip2.zip to zip2_renamed.zip
Adding changes to git...
  ✓ Added in 2850ms
Committing modifications...
  ✓ Committed in 421ms
Computing checksums after modifications...
Stored 6 checksums for step 3
✓ Step 3 complete

--- Step 4 ---
Cloning from /tmp/lfst/repo1 to /tmp/lfst/repo2...
  ✓ Cloned in 1234ms
Computing checksums in second clone...
Comparing checksums with step 3...
✓ Checksums match (6 files)
✓ Step 4 complete

--- Step 5 ---
Creating new file in second clone...
Adding new file to git...
  ✓ Added in 45ms
Committing new file...
  ✓ Committed in 123ms
Computing checksums after changes...
Stored 7 checksums for step 5
✓ Step 5 complete

--- Step 6 ---
Pulling changes from remote...
  (Skipping pull - remote not yet configured)
Computing checksums in first clone...
Stored 6 checksums for step 6
  Note: Checksum comparison with step 5 requires working pull
✓ Step 6 complete

--- Step 7 ---
Untracking patterns from LFS...
  ✓ Untracked *.pdf in 25ms
  ✓ Untracked *.mov in 18ms
  ✓ Untracked *.avi in 20ms
  ✓ Untracked *.ogg in 19ms
  ✓ Untracked *.m4v in 21ms
  ✓ Untracked *.zip in 17ms
Migrating files out of LFS...
  ✓ Migrated files in 5432ms
Adding .gitattributes changes...
  ✓ Added in 12ms
Committing LFS untrack...
  ✓ Committed in 98ms
Computing final checksums...
Stored 6 checksums for step 7
✓ Files successfully untracked from LFS
✓ Step 7 complete

✓ Scenario 6 completed successfully
  Run ID: 1
  View results: lfst-run show 1
```
