---
name: LinkedIn
tags: [social, linkedin, professional, marketing]
requires:
  bins: [python3]
  env: [LINKEDIN_LI_AT, LINKEDIN_JSESSIONID]
---

# LinkedIn

Read LinkedIn profiles, feed, and messages using the unofficial `linkedin-api` Python library.

## Installation

```bash
pip install linkedin-api
```

## Authentication

LinkedIn uses session cookies for authentication. Get these from your browser:
1. Log in to linkedin.com
2. Open DevTools → Application → Cookies → www.linkedin.com
3. Copy the values of `li_at` and `JSESSIONID`
4. Set as environment variables:
   - `LINKEDIN_LI_AT` — the `li_at` cookie value
   - `LINKEDIN_JSESSIONID` — the `JSESSIONID` cookie value

> **Warning**: `linkedin-api` uses unofficial API endpoints.
> This violates LinkedIn's Terms of Service. Use responsibly and sparingly.

## Capabilities

- Search people and companies
- View profiles
- Read feed posts
- Access messages (inbox)

## How to Use

Use the `exec` tool to run Python scripts.

### Search for people
```python
python3 -c "
from linkedin_api import Linkedin
import os, json
api = Linkedin(os.environ['LINKEDIN_LI_AT'], os.environ['LINKEDIN_JSESSIONID'], authenticate=False)
results = api.search_people('software engineer', limit=5)
print(json.dumps(results, indent=2))
"
```

### Get a profile
```python
python3 -c "
from linkedin_api import Linkedin
import os, json
api = Linkedin(os.environ['LINKEDIN_LI_AT'], os.environ['LINKEDIN_JSESSIONID'], authenticate=False)
profile = api.get_profile('public-profile-url-id')
print(json.dumps(profile, indent=2))
"
```

### Read feed
```python
python3 -c "
from linkedin_api import Linkedin
import os, json
api = Linkedin(os.environ['LINKEDIN_LI_AT'], os.environ['LINKEDIN_JSESSIONID'], authenticate=False)
feed = api.get_feed_posts(limit=10)
print(json.dumps(feed, indent=2))
"
```

## Notes

- Use `authenticate=False` when passing cookies directly (not username/password)
- Session cookies expire periodically — refresh from browser when API calls fail
- Rate limit cautiously — LinkedIn actively blocks scrapers
- Advanced features (messaging, posting) may require additional permissions
