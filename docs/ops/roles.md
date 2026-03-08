# BlackCat Roles Reference

BlackCat uses a priority-based role router. Each role maps to a subagent with specific capabilities.

**How it works:**
- When a message arrives, the router scans the text for role keywords
- Roles are checked in priority order (lower number = higher precedence)
- The first role with a matching keyword wins
- If no keywords match, `oracle` (priority 100) handles the message as fallback

**Location:** Roles are defined in `src/blackcat/router.ts` in the `DEFAULT_ROLES` array.

## Role: phantom (priority 10)

**Purpose:** Infrastructure & DevOps

**Keywords:**
```
restart, deploy, server, status, docker, systemctl, health, infra, devops,
service, nginx, ssl, ssh, vpn, firewall, kubernetes, k8s
```

**Workspace:** `workspaces/phantom/`

**Triggers on:** Messages about servers, deployment, containers, networking, system administration

**Example triggers:**
- "restart the server"
- "deploy to production"
- "docker container is down"
- "check nginx config"
- "kubernetes pod stuck"

---

## Role: astrology (priority 20)

**Purpose:** Crypto & Web3

**Keywords:**
```
crypto, bitcoin, btc, eth, ethereum, trading, token, defi, nft, wallet, market,
portfolio, investment, stock, forex, chart, candlestick, pump, whale, altcoin,
blockchain, web3
```

**Workspace:** `workspaces/astrology/`

**Triggers on:** Messages about cryptocurrency, trading, DeFi, NFTs, blockchain

**Example triggers:**
- "what's bitcoin doing today"
- "analyze this trading chart"
- "check my portfolio"
- "explain defi protocols"
- "is this a good time to buy eth"

---

## Role: wizard (priority 30)

**Purpose:** Software Engineering

**Keywords:**
```
code, implement, function, bug, fix, test, build, compile, git, opencode,
typescript, golang, python, javascript, refactor, debug, api, endpoint,
database, sql, migration, error, exception, crash
```

**Workspace:** `workspaces/wizard/`

**Triggers on:** Messages about coding, debugging, testing, building, databases

**Example triggers:**
- "fix this typescript error"
- "write a python function"
- "debug this API endpoint"
- "help with git rebase"
- "refactor this code"

---

## Role: artist (priority 40)

**Purpose:** Social Media Content

**Keywords:**
```
instagram, tiktok, twitter, linkedin, facebook, threads, post, caption,
hashtag, reel, story, content, social, viral, engagement, schedule, publish,
influencer, brand, creative
```

**Workspace:** `workspaces/artist/`

**Triggers on:** Messages about social media posts, content creation, scheduling

**Example triggers:**
- "write an instagram caption"
- "create a viral tiktok script"
- "schedule a linkedin post"
- "hashtag strategy for reels"
- "social media content calendar"

---

## Role: scribe (priority 50)

**Purpose:** Writing & Documentation

**Keywords:**
```
write, draft, article, blog, email, document, copy, copywriting, proofread,
translate, summarize, report, newsletter, pitch, proposal, readme,
documentation, essay
```

**Workspace:** `workspaces/scribe/`

**Triggers on:** Messages about writing, editing, documentation, translation

**Example triggers:**
- "draft an email to the team"
- "write a blog post about AI"
- "summarize this document"
- "proofread my proposal"
- "help with my readme"

---

## Role: explorer (priority 60)

**Purpose:** Research & Information

**Keywords:**
```
search, find, look up, what is, explain, research, summarize, web, browse,
read, compare, analyze, review, investigate, information, news, latest
```

**Workspace:** `workspaces/explorer/`

**Triggers on:** Messages about research, explanations, web searches, comparisons

**Example triggers:**
- "what is kubernetes"
- "search for the latest news"
- "compare these two products"
- "research this company"
- "explain how blockchain works"

---

## Role: oracle (priority 100)

**Purpose:** Fallback — general assistant for everything else

**Keywords:** *(empty)*

**Workspace:** `workspaces/oracle/`

**Triggers on:** Any message that does not match other role keywords

**Behavior:** This is the catch-all role. If no keywords from phantom, astrology, wizard, artist, scribe, or explorer match the message, oracle handles it. It has the lowest precedence (highest priority number).

---

## Priority System Explained

| Priority | Order | Role |
|----------|-------|------|
| 10 | 1st | phantom |
| 20 | 2nd | astrology |
| 30 | 3rd | wizard |
| 40 | 4th | artist |
| 50 | 5th | scribe |
| 60 | 6th | explorer |
| 100 | Last | oracle |

**Key points:**
- Lower priority number = higher precedence
- Roles are sorted by priority, then checked in order
- First match wins
- oracle with empty keywords acts as the fallback

**Example:** A message saying "deploy this code to the server" contains keywords from both `phantom` (deploy, server) and `wizard` (code). phantom has priority 10 vs wizard's 30, so phantom wins.
