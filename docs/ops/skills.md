# BlackCat Skills Reference

Skills are reusable capabilities that subagents can invoke. They are stored as Markdown files with YAML frontmatter and loaded dynamically at runtime.

## How Skills Work

- Skills are located in `workspaces/shared-skills/<skill-name>/SKILL.md`
- Only subdirectories with `SKILL.md` files are loaded
- Flat `.md` files in `shared-skills/` root are ignored
- Skills auto-load via `skills.load.extraDirs` in config
- No daemon restart needed after adding skills

## The 18 Built-in Skills

### Social Platforms

| Skill | Description | Requirements |
|-------|-------------|--------------|
| **twitter** | Twitter/X posting, threads, engagement | — |
| **threads** | Threads (Meta) posting and management | — |
| **linkedin** | LinkedIn professional content | — |
| **facebook** | Facebook page management | — |
| **tiktok** | TikTok content and captions | — |

### Social Strategy

| Skill | Description | Requirements |
|-------|-------------|--------------|
| **social-media** | General social media strategy and scheduling | — |

### Research

| Skill | Description | Requirements |
|-------|-------------|--------------|
| **research** | Web research, information gathering | — |

### Writing

| Skill | Description | Requirements |
|-------|-------------|--------------|
| **writing** | Writing best practices, documentation | — |

### Coding

| Skill | Description | Requirements |
|-------|-------------|--------------|
| **coding** | Coding best practices, code quality | — |
| **self-management** | BlackCat CLI management commands | — |
| **pinchtab-browsing** | Web browsing via PinchTab API | `BLACKCAT_PINCHTAB_ENABLED=true` |

### AI & Generation

| Skill | Description | Requirements |
|-------|-------------|--------------|
| **veo3-video-gen** | Google Veo 3 video generation | uv, ffmpeg, `GEMINI_API_KEY` |
| **nano-banana-pro** | Gemini image generation | uv, `GEMINI_API_KEY` |
| **document-processing** | PDF/DOCX/XLSX extraction | python3 |

### Integrations

| Skill | Description | Requirements |
|-------|-------------|--------------|
| **google-workspace** | Google Drive/Gmail/Calendar/Sheets | gws CLI |
| **reddit-scraper** | Reddit posts/comments scraping | python3 |

### Crypto & Finance

| Skill | Description | Requirements |
|-------|-------------|--------------|
| **crypto-web3** | Crypto trading, DeFi, Web3 operations | — |

### DevOps

| Skill | Description | Requirements |
|-------|-------------|--------------|
| **devops-infra** | Infrastructure management, Docker, k8s | — |

---

## Skill Format

Each skill is a Markdown file with YAML frontmatter:

```markdown
---
name: Skill Display Name
version: v1.0.0
tags: [tag1, tag2]
requires:
  bins: [python3, ffmpeg]    # Required binaries
  env: [API_KEY, BASE_URL]   # Required environment variables
---

# Skill Content

Instructions, examples, and context for the LLM to use this skill.
```

## Adding a New Skill

1. **Create the directory and file:**
   ```bash
   mkdir -p workspaces/shared-skills/my-new-skill
   touch workspaces/shared-skills/my-new-skill/SKILL.md
   ```

2. **Write the SKILL.md with frontmatter:**
   ```markdown
   ---
   name: My New Skill
   version: v1.0.0
   tags: [api, integration]
   requires:
     bins: [curl]
     env: [MY_API_KEY]
   ---

   # My New Skill

   Instructions for using this skill...
   ```

3. **Verify it loads:**
   - No restart needed
   - Skills are loaded fresh on each subagent creation
   - Check `blackcat doctor` or logs if issues occur

## Skill Requirements Reference

### Binaries
- `python3` — For Python-based skills (document-processing, reddit-scraper)
- `uv` — Fast Python package manager (veo3-video-gen, nano-banana-pro)
- `ffmpeg` — Video processing (veo3-video-gen)
- `gws` — Google Workspace CLI (google-workspace)

### Environment Variables
- `BLACKCAT_PINCHTAB_ENABLED=true` — Enable PinchTab browsing
- `BLACKCAT_PINCHTAB_BASE_URL` — PinchTab API endpoint
- `BLACKCAT_PINCHTAB_TOKEN` — PinchTab authentication
- `GEMINI_API_KEY` — Google Gemini API (veo3-video-gen, nano-banana-pro)
