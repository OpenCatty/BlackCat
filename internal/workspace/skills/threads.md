---
name: Threads
tags: [social, threads, meta, marketing]
requires:
  env: [THREADS_ACCESS_TOKEN]
---

# Threads

Post and read content on Meta Threads using the official Threads Graph API.

## Authentication

Requires a Threads API access token set as `THREADS_ACCESS_TOKEN` environment variable.
Get a token from Meta Developer Console: https://developers.facebook.com/docs/threads

## Capabilities

- Publish text posts, single image posts, and carousel posts (up to 10 images)
- Read your own posts and profile information
- Reply to threads

## How to Use

Use the `web` tool to make HTTP requests to the Threads Graph API.

Base URL: `https://graph.threads.net/v1.0`
Authorization: `Bearer $THREADS_ACCESS_TOKEN`

### Get profile info
```
GET https://graph.threads.net/v1.0/me?fields=id,username,name,biography&access_token={TOKEN}
```

### Publish a text post
Step 1 — Create media container:
```
POST https://graph.threads.net/v1.0/me/threads
  media_type=TEXT
  text=Your post content here
  access_token={TOKEN}
```
Returns `{ "id": "<container_id>" }`

Step 2 — Publish:
```
POST https://graph.threads.net/v1.0/me/threads_publish
  creation_id=<container_id>
  access_token={TOKEN}
```

### List your posts
```
GET https://graph.threads.net/v1.0/me/threads?fields=id,text,timestamp,permalink&access_token={TOKEN}
```

## Notes

- Text posts: max 500 characters
- Image posts: provide `image_url` (publicly accessible URL)
- Carousel: create individual IMAGE containers first, then combine with `media_type=CAROUSEL`
- Rate limits: 250 API calls per hour per token
- Always use HTTPS URLs for image/video assets
