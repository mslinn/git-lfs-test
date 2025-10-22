# LFS Test Server Setup

This document describes how to configure `lfs-test-server` on `gojira` for
automated testing without user authentication prompts.


## Overview

For automated testing, `lfs-test-server` must be configured with:

1. Admin interface enabled (to manage users)
2. A test user account created
3. Client credentials embedded in LFS URL

## Server Configuration

### 1. Start lfs-test-server with Admin Interface

Create the following startup script at `/opt/lfs-test-server/start-lfs-server.sh`.
The script overwrites `/opt/lfs-test/server/lfs-server.log` each time it runs.

```bash
#!/bin/bash
# Start lfs-test-server with admin interface enabled

export LFS_CONTENTPATH=/opt/lfs-test-server
export LFS_ADMINUSER=admin
export LFS_ADMINPASS=admin123

# Install lfs-test-server if not found
if [ ! `which lfs-test-server` ]; then
  go install github.com/git-lfs/lfs-test-server@latest
fi

export LFS_REPO=/opt/lfs-test-server
export LFS_PORT=8080
export SERVER=`uname -n`
export LOG_FILE=$LFS_REPO/lfs-server.log

cd "$LFS_REPO" | exit 1

# Kill any existing instance
pkill -f lfs-test-server

# Start server in background with verbose logging
nohup ~/go/bin/lfs-test-server -verbose -addr :$LFS_PORT > lfs-server.log 2>&1 &

echo "LFS Test Server started on port $LFS_PORT"
echo "Admin interface: http://$SERVER:$LFS_PORT/mgmt"
echo "Log file: $SERVER:$LOG_FILE"
echo "Tailing log file..."
tail -f "$LOG_FILE"
```

Make it executable:

```bash
chmod +x "$LFS_REPO/start-lfs-server.sh"
```

### 2. Start the Server

#### Manual Start

To start the server manually on gojira:

```bash
# On gojira directly
/opt/lfs-test-server/start-lfs-server.sh

# Or from a remote client
ssh gojira "/opt/lfs-test-server/start-lfs-server.sh"
```

This will:

- Kill any existing lfs-test-server instance
- Start a new instance with admin interface and verbose logging
- Display server status information

**Important:** The server runs in the background using `nohup` and continues
running even after you log out. You don't need to keep your SSH session open.

#### Automatic Start on Boot

To start the server automatically when gojira boots, add this line to crontab:

```bash
# On gojira, edit crontab
crontab -e

# Add this line:
@reboot sleep 60 && /opt/lfs-test-server/start-lfs-server.sh
```

The 60-second delay ensures the network is fully initialized before starting the server.

#### Restart the Server

If you need to restart the server (e.g., after configuration changes):

```bash
# From gojira directly
/opt/lfs-test-server/start-lfs-server.sh

# Or from a remote client
ssh gojira "/opt/lfs-test-server/start-lfs-server.sh"
```

The startup script kills any existing instance before starting a new one.


**Note:** After running these commands, the server continues running in the background even if you close your SSH session or log out.

### 3. Create Test User

The test user account is required for all LFS operations. Create it once via the admin interface:

```bash
curl -u admin:admin123 -X POST \
  -d "name=testuser&password=testpass" \
  http://gojira:8080/mgmt/add
```

Verify user was created:

```bash
curl -s -u admin:admin123 http://gojira:8080/mgmt/users | grep testuser
```

**Note:** The test user only needs to be created once. The user account is stored in the database (`/opt/lfs-test-server/lfs.db`) and persists across server restarts and reboots. You don't need to recreate the user after restarting the server.

## Client Configuration

For automated testing without credential prompts, embed credentials in the LFS URL in `.lfsconfig`:

```ini
[lfs]
	url = http://testuser:testpass@gojira:8080
```

This allows Git LFS to authenticate automatically without user interaction.

## Environment Variables

The following environment variables control lfs-test-server behavior:

- **LFS_CONTENTPATH**: Directory for LFS object storage (default: `./lfs-content`)
- **LFS_ADMINUSER**: Admin username for `/mgmt` interface (no default)
- **LFS_ADMINPASS**: Admin password for `/mgmt` interface (no default)
- **LFS_HOST**: Server listen address (default: `localhost:8080`)

## Security Note

The credentials above (`testuser`/`testpass`) are for **testing only** on internal networks. Do not use these credentials in production environments or expose the server to public networks.

## Verification

Test that the server accepts LFS operations:

```bash
# Create test repo
cd /tmp && rm -rf lfs-test && mkdir lfs-test && cd lfs-test
git init
git lfs install

# Configure LFS URL with credentials
git config -f .lfsconfig lfs.url "http://testuser:testpass@gojira:8080"

# Track and commit a test file
echo "test" > test.txt
git lfs track "*.txt"
git add .
git commit -m "test"

# Create bare repo and push
ssh gojira "rm -rf /tmp/test-bare.git && git init --bare /tmp/test-bare.git"
git remote add origin gojira:/tmp/test-bare.git
git push origin master

# Should complete without credential prompts
```

## Troubleshooting

### Server Logs

Monitor server activity:
```bash
ssh gojira "tail -f /opt/lfs-test-server/lfs-server.log"
```

### Check Server Status

```bash
ssh gojira "ps aux | grep lfs-test-server | grep -v grep"
curl -v http://gojira:8080/
```

### Verify Automatic Startup After Reboot

After rebooting gojira, verify the server started automatically:

```bash
# Wait at least 60 seconds after reboot, then check
ssh gojira "ps aux | grep lfs-test-server | grep -v grep"

# Check the log to see startup message
ssh gojira "tail -20 /opt/lfs-test-server/lfs-server.log"

# Verify server is responding
curl -v http://gojira:8080/

# Confirm test user still exists (should persist)
curl -s -u admin:admin123 http://gojira:8080/mgmt/users | grep testuser
```

If the server didn't start automatically, check the crontab:

```bash
ssh gojira "crontab -l | grep lfs-test-server"
```

### Reset Server

To start fresh with a new database:
```bash
ssh gojira "cd /opt/lfs-test-server && pkill lfs-test-server && mv lfs.db lfs.db.bak"
/opt/lfs-test-server/start-lfs-server.sh
```

Then recreate the test user as shown in step 3 above.

## Automated Test Integration

The `lfst-scenario` command automatically uses the embedded credentials URL for scenarios 6 and 7 (LFS Test Server scenarios). No manual credential configuration is needed when running these scenarios.
