---
name: Twitter (X)
tags: [social, twitter, x, marketing]
requires:
  bins: [bird]
  env: [TWITTER_AUTH_TOKEN]
---

# Twitter (X)

Read and post on X (formerly Twitter) using the `bird` CLI tool.

## Installation

```bash
npm install -g @steipete/bird
```

## Authentication

Set `TWITTER_AUTH_TOKEN` to your X session cookie token from browser DevTools.

> **Warning**: `bird` uses cookie-based authentication against X's internal API.
> This is against X's Terms of Service and may result in account suspension.
> Use at your own risk. Read operations are lower risk than write operations.

## Capabilities

- Read timeline, tweets, and threads
- Search tweets and users
- Post tweets (use sparingly — ToS risk)
- View bookmarks and trending topics

## How to Use

Use the `exec` tool to run `bird` commands.

### Check authenticated user
```bash
bird whoami
```

### Read home timeline
```bash
bird timeline --count 10
```

### Search tweets
```bash
bird search "query here" --count 20
```

### Read a specific tweet/thread
```bash
bird tweet <tweet_url_or_id>
```

### Post a tweet
```bash
bird tweet post "Your tweet text here"
```

### View trending topics
```bash
bird trending
```

## Notes

- All commands output JSON by default — use `--plain` for readable text
- `TWITTER_AUTH_TOKEN` is the `auth_token` cookie from x.com (browser DevTools → Application → Cookies)
- Tokens can be revoked by X at any time
- Prefer read operations over write operations to reduce suspension risk
