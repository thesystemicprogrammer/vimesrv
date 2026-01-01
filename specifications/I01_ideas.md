# Media Management API (Optional)

## Endpoints:

- GET /api/v1/media - List all media
- GET /api/v1/media/:id - Get single media
- PATCH /api/v1/media/:id - Update metadata
- DELETE /api/v1/media/:id - Delete media
  Priority: Low (system works without it)

# Web Player Integration (Optional)

## Tasks:

- Update player.html to auto-detect media
- Add media library browser UI
- Add quality selector
- Add audio/subtitle track selector
  Priority: Medium (improves UX)

# Live Transcoding Progress (Optional)

## Tasks:

- WebSocket endpoint for real-time progress
- Progress bar in web UI
- Transcode job cancellation
  Priority: Low (nice to have)
