# Job Overview Page Specification

## Overview
The Jobs page displays currently running and historical jobs with real-time progress updates via WebSocket.

## Access Control
- Accessible by users with `admin` or `manager` roles
- Protected by `managerGuard`
- Route: `/jobs`

## Features

### Filter Tabs
- All | Running | Queued | Completed | Failed

### Job Sorting (by status priority)
1. Running (priority 0)
2. Queued/Pending (priority 1)
3. Failed/Dead (priority 2)
4. Completed/Succeeded (priority 3)

Within same status, sorted by `created_at` descending (newest first).

### Job Status Mapping
| API Status | UI Label |
|------------|----------|
| `running` | Running |
| `queued` | Pending/Queued |
| `dead` | Failed |
| `succeeded` | Completed |

## Job Card Display

### Common Elements
- Job type icon (color-coded)
- Job type label
- Status badge
- Job ID

### Job-Specific Details
| Job Type | Detail Displayed |
|----------|------------------|
| `enrich_metadata` | Filename (e.g., "dune_2021.mp4") |
| `fetch_translations` | Full language name (e.g., "German", "Deutsch") |
| `transcode_video` / `transcode_audio` | Transcode ID + progress message when running |
| `scan_library` | No additional details |

### Timestamps
- **Queued jobs**: Show "Created: {date}"
- **Running jobs**: Show "Started: {date}"
- **Completed/Failed jobs**: Show "Started: {date}", "Ended: {date}", "Duration: {duration}"

Date format is locale-aware (uses `Intl.DateTimeFormat`):
- English: `Jan 3, 2026, 2:32 PM`
- German: `03.01.2026, 14:32`

Duration format: `2m 45s` or `1h 15m 30s`

### Progress Display
- **Running transcode jobs**: Progress bar with percentage, FPS, speed, bitrate
- **Running non-transcode jobs**: Indeterminate spinner with "Processing..."
- **Failed jobs**: Error message in red box

## UI Mockups

### Running Transcode Job
```
┌─────────────────────────────────────────────────────────────────┐
│ 🎬 Video Transcode                              ● RUNNING       │
│ Transcoding video - inception_2010.mp4                          │
│ ████████████████████░░░░░░░░░░  67%                             │
│ FPS: 24.5 | Speed: 1.2x | Bitrate: 4500 kbps                    │
│ Started: Jan 3, 2026, 2:32 PM                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Running Non-Transcode Job
```
┌─────────────────────────────────────────────────────────────────┐
│ 📝 Metadata Enrichment                          ● RUNNING       │
│ File: dune_2021.mp4                                             │
│ ◐ Processing...  (indeterminate spinner)                        │
│ Started: Jan 3, 2026, 2:35 PM                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Queued Job
```
┌─────────────────────────────────────────────────────────────────┐
│ 🌐 Fetch Translations                           ○ QUEUED        │
│ Language: German                                                │
│ Created: Jan 3, 2026, 2:35 PM                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Failed Job
```
┌─────────────────────────────────────────────────────────────────┐
│ ✗ Audio Transcode                               ✗ FAILED        │
│ Transcode ID: abc123                                            │
│ Attempts: 3/3                                                   │
│ Started: Jan 3, 2026, 1:20 PM                                   │
│ Ended: Jan 3, 2026, 1:25 PM | Duration: 5m 33s                  │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ Error: ffmpeg exited with code 1                            │ │
│ └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Completed Job
```
┌─────────────────────────────────────────────────────────────────┐
│ ✓ Library Scan                                  ✓ COMPLETED     │
│ Started: Jan 3, 2026, 12:00 PM                                  │
│ Ended: Jan 3, 2026, 12:02 PM | Duration: 2m 45s                 │
└─────────────────────────────────────────────────────────────────┘
```

## WebSocket Events

### Subscribed Events
- `job_started`: Job begins processing
- `job_progress`: Transcode progress update (percentage, fps, speed, bitrate, message)
- `job_completed`: Job finished successfully
- `job_failed`: Job failed with error
- `job_retrying`: Job being retried

### Progress Payload (for transcode jobs)
```typescript
interface JobProgressPayload {
  job_id: number;
  job_type: string;
  frame?: number;
  fps?: number;
  bitrate?: string;
  time?: string;
  speed?: string;
  percentage?: number;
  message?: string;  // Contains "Transcoding video - filename.mp4"
}
```

## Job Payload Structures

```go
// Transcode jobs
TranscodeJobPayload { TranscodeID string `json:"transcode_id"` }

// Enrich metadata
EnrichMetadataJobPayload { 
    MediaID  string `json:"media_id"`
    Filename string `json:"filename"`
}

// Fetch translations
FetchTranslationsJobPayload { Language string `json:"language"` }

// Scan library - no payload
```

## Implementation Notes

### Backend
- `EnrichMetadataJobPayload` includes `Filename` field for UI display
- Filename is set when job is enqueued in `scan_library.go`

### Frontend
- Uses `Intl.DisplayNames` for language code to name conversion
- Uses `toLocaleString()` with current language for date formatting
- Duration calculated from `started_at` and `finished_at` timestamps
