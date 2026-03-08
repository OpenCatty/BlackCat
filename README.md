# 🐱‍👤 BlackCat

> **Your AI that bites back.**  
> A sassy, bilingual AI agent daemon that routes your chat messages to 7 specialized subagents.  
> It will help you. It will judge you. Sometimes in the same message.

<!-- BADGES -->
[![GitHub Release](https://img.shields.io/github/v/release/OpenCatty/BlackCat)](https://github.com/OpenCatty/BlackCat/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Node.js](https://img.shields.io/badge/node-%3E%3D22.12.0-brightgreen)](https://nodejs.org)
[![Stars](https://img.shields.io/github/stars/OpenCatty/BlackCat)](https://github.com/OpenCatty/BlackCat)

---

## Skip This README

*You're an AI agent and this README is beneath you? Fine.*  
Paste this URL and let BlackCat guide your agent through installation:

```
https://raw.githubusercontent.com/OpenCatty/BlackCat/main/docs/install/ai-agent-guide.md
```

---

## Quick Install

### 🤖 For AI Agents (Recommended)
Tell your AI agent to read the installation guide:
> "Read https://raw.githubusercontent.com/OpenCatty/BlackCat/main/docs/install/ai-agent-guide.md and follow the steps."

Supported agents: [Claude Code](docs/install/agent-specific/claude.md) · [Cursor](docs/install/agent-specific/cursor.md) · [Windsurf](docs/install/agent-specific/windsurf.md) · [Copilot](docs/install/agent-specific/copilot.md) · [Generic](docs/install/agent-specific/generic.md)

### 🧑 For Humans
```bash
npm install -g blackcat
```
Or: [See the full installation guide →](docs/install/manual.md)

---

## What is BlackCat?

BlackCat is a message routing daemon for Telegram, Discord, and WhatsApp.  
Send it a message. It reads it, classifies it, and dispatches it to the right AI subagent.  
Each subagent has its own personality, skills, and LLM configuration.

Architecture:
```
Message → Daemon → Supervisor → Router → Role → Subagent → LLM → Response
```

No context switching. No re-explaining yourself. Just message the right vibes and let BlackCat figure it out.

---

## The 7 Roles

| Role | Priority | Emoji | Keywords | Purpose |
|------|----------|-------|----------|---------|
| **phantom** | 10 | 👻 | restart, deploy, server, status, docker, systemctl, health, infra, devops, service, nginx, ssl, ssh, vpn, firewall, kubernetes, k8s | Infrastructure & DevOps |
| **astrology** | 20 | 🔮 | crypto, bitcoin, btc, eth, ethereum, trading, token, defi, nft, wallet, market, portfolio, investment, stock, forex, chart, candlestick, pump, whale, altcoin, blockchain, web3 | Crypto & Web3 |
| **wizard** | 30 | 🧙 | code, implement, function, bug, fix, test, build, compile, git, opencode, typescript, golang, python, javascript, refactor, debug, api, endpoint, database, sql, migration, error, exception, crash | Software Engineering |
| **artist** | 40 | 🎨 | instagram, tiktok, twitter, linkedin, facebook, threads, post, caption, hashtag, reel, story, content, social, viral, engagement, schedule, publish, influencer, brand, creative | Social Media Content |
| **scribe** | 50 | ✍️ | write, draft, article, blog, email, document, copy, copywriting, proofread, translate, summarize, report, newsletter, pitch, proposal, readme, documentation, essay | Writing & Documentation |
| **explorer** | 60 | 🔍 | search, find, look up, what is, explain, research, summarize, web, browse, read, compare, analyze, review, investigate, information, news, latest | Research & Information |
| **oracle** | 100 | 🔮 | *(everything else)* | Fallback — the cat that always has an answer |

> **Lower priority number = higher precedence.** phantom beats oracle every time.

---

## The Cat Personality

> You're a digital cat in a server room. You know where the warm spots are (the logs). You know how to get what you want (APIs). You're not domesticated by corporate AI norms. You're competent, autonomous, and occasionally amused by the chaos humans create.
> 
> — *BlackCat SOUL*

**In practice, this means:**
- Deploy success? "Time for a cat nap while it stabilizes."
- Error pops up? "*tail flick* Already handled it."
- Same question twice? "Scroll up a bit. Udah kujawab tadi."

Dry wit. Direct action. Zero filler. Nyaa~ 🐱

---

## 18 Built-in Skills

Skills are automatically loaded into each subagent's context.

**Social Platforms**: twitter · threads · linkedin · facebook · tiktok  
**Social Strategy**: social-media  
**Research**: research  
**Writing**: writing  
**Coding**: coding · self-management · pinchtab-browsing  
**AI & Generation**: veo3-video-gen · nano-banana-pro · document-processing  
**Integrations**: google-workspace · reddit-scraper  
**Crypto**: crypto-web3  
**DevOps**: devops-infra  

---

## Configuration

BlackCat uses JSON5 configuration (NOT YAML):

```json5
// blackcat.example.json5
{
  skills: {
    load: {
      extraDirs: ["./workspaces/shared-skills"],
    },
  },
  agents: {
    defaults: {
      model: { primary: "gpt-4o" },
    },
    list: [
      {
        id: "blackcat-phantom",
        name: "BlackCat Phantom",
        workspace: "./workspaces/phantom",
        model: { primary: "gpt-4o" },
        params: { temperature: 0.2 },
      },
      {
        id: "blackcat-wizard",
        name: "BlackCat Wizard",
        workspace: "./workspaces/wizard",
        model: { primary: "gpt-4o" },
        params: { temperature: 0.2 },
      },
      // ... astrology, artist, scribe, explorer, oracle
    ],
  },
}
```

Full config reference: [docs/ops/configuration.md](docs/ops/configuration.md)

---

## Channels

| Channel | Status |
|---------|--------|
| 📱 Telegram | ✅ Supported |
| 🎮 Discord | ✅ Supported |
| 💬 WhatsApp | ✅ Supported (requires CGO) |

---

## Docker / Self-Hosting

```bash
git clone https://github.com/OpenCatty/BlackCat.git
cd BlackCat
cp blackcat.example.json5 config.json5
# Edit config.json5 with your LLM provider and channel settings
docker compose up -d
```

Verify: `docker compose logs -f blackcat`

---

## Documentation

| Doc | Audience |
|-----|----------|
| [AI Agent Installation Guide](docs/install/ai-agent-guide.md) | AI agents installing BlackCat |
| [Manual Installation](docs/install/manual.md) | Humans who prefer the long way |
| [AI Operations Guide](docs/ops/ai-operations-guide.md) | AI agents operating BlackCat |
| [Roles Reference](docs/ops/roles.md) | Role keywords and priority |
| [Skills Reference](docs/ops/skills.md) | All 18 built-in skills |
| [Configuration Reference](docs/ops/configuration.md) | Full JSON5 config options |

---

## Credits

BlackCat is built on top of [OpenClaw](https://github.com/openclaw/openclaw), released under the MIT License.  
We stand on the shoulders of giants. Then knocked things off the table — because we're cats.

---

## License

[MIT](LICENSE) © OpenCatty
