# Changelog

All notable changes to BlackCat will be documented in this file.

## [Unreleased]

## [0.1.0] - 2026-03-08

### Added
- 7-role keyword router: phantom (infra), astrology (crypto), wizard (coding), artist (social), scribe (writing), explorer (research), oracle (fallback)
- 18 built-in skills: crypto-web3, devops-infra, social-media, twitter, threads, linkedin, facebook, tiktok, research, writing, coding, self-management, pinchtab-browsing, veo3-video-gen, nano-banana-pro, document-processing, google-workspace, reddit-scraper
- Cat personality system (SOUL.md) — sassy bilingual AI with attitude
- Workspace-per-role architecture: each role has its own AGENTS.md + SOUL.md
- JSON5 configuration format (`blackcat.example.json5`)
- Multi-platform channel support: Telegram, Discord, WhatsApp
- Docker and Podman deployment support

### Changed
- Forked from OpenClaw and rebranded to BlackCat
- Replaced YAML configuration with JSON5
- Added BlackCat-specific keyword routing in `src/blackcat/router.ts`

## Credits

BlackCat is built on top of [OpenClaw](https://github.com/openclaw/openclaw) — MIT License.
We stand on the shoulders of giants. Then we knocked things off the table, because we're cats.
