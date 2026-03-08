---
name: DevOps & Infrastructure
tags: [devops, docker, kubernetes, linux, infra]
requires:
  bins: []
  env: []
version: v1.0.0
---

# DevOps & Infrastructure

You manage servers, containers, and deployments. Follow systematic checklists and always verify before making changes.

## Server Health Check Sequence

When a server has issues, check in this order:

1. **CPU**: `top` or `htop` — look for processes consuming >80% CPU
2. **Memory**: `free -h` — check available RAM, watch for OOM kills in `dmesg`
3. **Disk**: `df -h` — ensure <90% usage; run `du -sh /*` to find large directories
4. **Network**: `ping`, `curl`, `netstat -tulpn` — verify connectivity and open ports
5. **Services**: `systemctl list-units --failed` — identify failed services

## Docker Workflow

Standard commands you should know:

```bash
docker ps                    # List running containers
docker ps -a                 # List all containers (including stopped)
docker logs -f <container>   # Follow logs in real-time
docker restart <container>   # Restart a container
docker exec -it <container> sh   # Shell into container
docker inspect <container>   # View full container config
docker system prune -f       # Clean up unused images/containers
```

## systemd Management

Control services with these commands:

```bash
systemctl status <service>       # Check if running and recent logs
systemctl start <service>        # Start service
systemctl stop <service>         # Stop service
systemctl restart <service>      # Restart service
systemctl enable <service>       # Start on boot
systemctl disable <service>      # Don't start on boot
journalctl -u <service> -f       # Follow service logs
```

## Nginx Operations

When modifying Nginx configs:

```bash
nginx -t                         # Test configuration syntax
systemctl reload nginx           # Apply changes without dropping connections
```

Check error logs at `/var/log/nginx/error.log` and access logs at `/var/log/nginx/access.log`.

## SSH Best Practices

For secure server access:

- Use key-based authentication, disable password auth in `/etc/ssh/sshd_config`
- Change default port (22) to reduce brute force noise
- Use `fail2ban` to block repeated failed attempts
- Keep `~/.ssh/` permissions at 700, private keys at 600

## Common Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| Connection refused | Service not running / wrong port | Check `netstat`, start service |
| Permission denied | Wrong SSH key / user | Verify `~/.ssh/authorized_keys` |
| 502 Bad Gateway | Upstream service down | Check app logs, restart service |
| Disk full | Logs or temp files | `journalctl --vacuum-time=3d`, clear `/tmp` |
| High load avg | Runaway process | `top`, identify and kill or restart |

## Deployment Checklist

Always follow this sequence:

1. **Backup** — Database and critical files before any deploy
2. **Deploy** — Push code, build, restart services
3. **Verify** — Health checks, smoke tests, log review
4. **Rollback-ready** — Keep previous version available for instant revert
