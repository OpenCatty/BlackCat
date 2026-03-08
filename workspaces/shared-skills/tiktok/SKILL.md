---
name: TikTok
tags: [social, tiktok, video, marketing]
requires:
  env: [TIKTOK_ACCESS_TOKEN]
---

# TikTok

Manage TikTok videos and view basic performance stats using the TikTok Content API.

## Authentication

Requires a TikTok API access token set as `TIKTOK_ACCESS_TOKEN` environment variable.

To get an access token:
1. Register at TikTok Developer Portal: https://developers.tiktok.com
2. Create an app with `video.list` and `video.upload` scopes
3. Complete OAuth 2.0 authorization flow
4. Set: `TIKTOK_ACCESS_TOKEN=your_access_token`

## Capabilities

- List your videos with metadata
- Query video details (views, likes, comments, shares)
- Upload new videos (direct post or upload)
- Update video settings (privacy, description, hashtags)

> **Note**: Advanced analytics (demographics, audience breakdowns, engagement rates by segment)
> are only available in TikTok Creator Portal — not accessible via the Content API.

## How to Use

Use the `web` tool to make HTTP requests to the TikTok API.

Base URL: `https://open.tiktokapis.com/v2`
Authorization: `Bearer $TIKTOK_ACCESS_TOKEN`

### List your videos
```
POST https://open.tiktokapis.com/v2/video/list/
  Authorization: Bearer {TOKEN}
  Content-Type: application/json
  Body: {
    "max_count": 20,
    "fields": ["id", "title", "video_description", "duration", "cover_image_url",
               "share_url", "view_count", "like_count", "comment_count", "share_count",
               "create_time"]
  }
```

### Get video details
```
POST https://open.tiktokapis.com/v2/video/query/
  Authorization: Bearer {TOKEN}
  Content-Type: application/json
  Body: {
    "filters": { "video_ids": ["<video_id>"] },
    "fields": ["id", "title", "view_count", "like_count", "comment_count", "share_count", "duration"]
  }
```

### Initialize video upload
```
POST https://open.tiktokapis.com/v2/post/publish/video/init/
  Authorization: Bearer {TOKEN}
  Content-Type: application/json
  Body: {
    "post_info": {
      "title": "Video caption #hashtag",
      "privacy_level": "PUBLIC_TO_EVERYONE",
      "disable_comment": false
    },
    "source_info": {
      "source": "FILE_UPLOAD",
      "video_size": <bytes>,
      "chunk_size": <bytes>,
      "total_chunk_count": 1
    }
  }
```
Returns `publish_id` and `upload_url` for chunked upload.

## Notes

- Video uploads use chunked transfer — initialize → upload chunks → publish
- Privacy levels: `PUBLIC_TO_EVERYONE`, `MUTUAL_FOLLOW_FRIENDS`, `SELF_ONLY`
- Max video duration: 10 minutes
- Supported formats: MP4, WebM, MOV
- Access tokens expire — implement refresh using your app's `client_key` and `client_secret`
