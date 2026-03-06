---
name: Facebook
tags: [social, facebook, meta, marketing]
requires:
  env: [FACEBOOK_PAGE_TOKEN]
---

# Facebook

Manage Facebook Pages and publish content using the official Facebook Graph API.

## Authentication

Requires a Facebook Page Access Token set as `FACEBOOK_PAGE_TOKEN` environment variable.

To get a Page Token:
1. Go to Meta Developer Console: https://developers.facebook.com
2. Create an app with `pages_manage_posts` and `pages_read_engagement` permissions
3. Generate a Page Access Token for your Page
4. Set: `FACEBOOK_PAGE_TOKEN=your_page_access_token`

You also need your Page ID. Find it at: `https://graph.facebook.com/me/accounts?access_token={TOKEN}`

## Capabilities

- Publish text posts and link posts to a Facebook Page
- Read Page posts and engagement metrics
- Moderate comments (hide/delete)
- Get Page info and insights

## How to Use

Use the `web` tool to make HTTP requests to the Facebook Graph API.

Base URL: `https://graph.facebook.com/v21.0`

### Get your pages
```
GET https://graph.facebook.com/v21.0/me/accounts?access_token={FACEBOOK_PAGE_TOKEN}
```

### Publish a text post
```
POST https://graph.facebook.com/v21.0/{PAGE_ID}/feed
  Content-Type: application/json
  Body: { "message": "Your post text here", "access_token": "{FACEBOOK_PAGE_TOKEN}" }
```

### Publish a link post
```
POST https://graph.facebook.com/v21.0/{PAGE_ID}/feed
  Body: {
    "message": "Check this out!",
    "link": "https://example.com",
    "access_token": "{FACEBOOK_PAGE_TOKEN}"
  }
```

### Read page posts
```
GET https://graph.facebook.com/v21.0/{PAGE_ID}/posts?fields=id,message,created_time,likes.summary(true)&access_token={FACEBOOK_PAGE_TOKEN}
```

### Get post comments
```
GET https://graph.facebook.com/v21.0/{POST_ID}/comments?access_token={FACEBOOK_PAGE_TOKEN}
```

## Notes

- Page Token != User Token — always use Page Token for Page operations
- Tokens can be short-lived (2h) or long-lived (60 days) — prefer long-lived
- Image posts require uploading to `/{PAGE_ID}/photos` endpoint first
- Video posts go through `/{PAGE_ID}/videos` with multipart upload
- Rate limits: 200 calls per hour per token
