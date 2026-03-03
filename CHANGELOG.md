# Changelog

All notable changes to BlackCat will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Changed
- Reorganized project structure: moved all business logic packages into `internal/`
- Removed redundant `docs/`, `extensions/`, `examples/`, `test/` directories

## [1.0.0] - 2026-03-03

### Added
- Complete UX overhaul: daemon commands, channel management, onboard wizard
- One-line install scripts for Linux/macOS and Windows
- Non-root deployment (user-level systemd/launchd/schtasks)
- Comprehensive VitePress documentation website
- Agent self-knowledge via skills system
- GoReleaser configuration for automated releases
