# Deploy Directory

This directory supports the `make deploy` workflow and contains systemd service files and configuration templates for deploying the blackcat application.

## Setup Instructions

### Step 1: Create your deploy configuration
Copy `deploy.env.example` to `deploy.env`:
```bash
cp deploy/deploy.env.example deploy/deploy.env
```

### Step 2: Fill in your VM details
Edit `deploy/deploy.env` and update with your actual values:
- `DEPLOY_HOST` — your VM's IP address or hostname (e.g., 35.198.216.167)
- `DEPLOY_USER` — SSH username for the VM
- `DEPLOY_SSH_KEY` — path to your SSH private key
- `DEPLOY_HOME` — home directory on the VM
- `DEPLOY_WORKDIR` — working directory path on the VM
- `DEPLOY_CONFIG_PATH` — path to blackcat.yaml on the VM
- `VAULT_PASSPHRASE` — vault encryption passphrase (keep this secret!)

### Step 3: Deploy
Run the deployment from the project root:
```bash
make deploy
```

## Files in this directory

| File | Purpose |
|------|---------|
| `deploy.env.example` | Template with all required variables and comments. Copy to `deploy.env` and fill in your values. |
| `.gitignore` | Ensures `deploy.env` (which contains secrets) is never committed to the repository. |
| `blackcat.service` | Systemd service file for the blackcat daemon. |
| `opencode.service` | Systemd service file for the opencode service. |

## Security Notice

⚠️ **CRITICAL:** The `deploy.env` file contains sensitive information (SSH keys, passphrases, etc.) and **MUST NEVER** be committed to version control. It is gitignored and should only exist locally on your development machine.

Never share your `deploy.env` file. If you need to rotate secrets, delete `deploy.env` and create a new one from `deploy.env.example`.
