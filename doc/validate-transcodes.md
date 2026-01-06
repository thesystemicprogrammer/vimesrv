# Validate Transcodes

A comprehensive validation and correction tool for transcoded media files that checks integrity, consistency, and audio/video synchronization.

## Overview

The `validate-transcodes.go` script scans all transcoded media and performs the following validations:

| Check | Description |
|-------|-------------|
| Database/Filesystem sync | Verify transcode records in DB have corresponding files on disk |
| Filesystem/Database sync | Verify transcode directories on disk have corresponding DB records |
| Init segment exists | Each video/audio transcode has `init.mp4` |
| Segment files exist | At least one `.m4s` chunk file exists |
| segments.json exists | Timing metadata file present |
| segments.json valid | No negative or excessive durations (>30s) |
| Segment count matches | Number of `.m4s` files matches segments.json entries |
| Audio/Video sync drift | Audio total duration within threshold of video |
| Subtitle files exist | VTT files present for subtitle transcodes |

## Usage

```bash
go run scripts/validate-transcodes.go [options] <database-path> <media-root-path>
```

### Options

| Option | Description |
|--------|-------------|
| `--dry-run` | Report issues without making any changes |
| `--fix-segments` | Regenerate corrupted or missing segments.json files |
| `--fix-orphaned-dirs` | Delete transcode directories with no database record |
| `--fix-orphaned-records` | Delete database records with no files on disk |
| `--create-transcode-jobs` | Create transcode jobs for media needing audio re-transcode |
| `--drift-threshold <ms>` | Audio/video drift threshold in milliseconds (default: 200) |
| `--no-probe` | Trust existing segments.json instead of probing with ffprobe |
| `--verbose` | Show detailed output for each media file |
| `--json` | Output results as JSON for programmatic use |
| `-h, --help` | Show help message |

## Examples

### Basic validation

Scan all media and report issues:

```bash
go run scripts/validate-transcodes.go ./data/vimesrv.db /mnt/media
```

### Quick validation (no ffprobe)

Trust existing segments.json files instead of probing each segment:

```bash
go run scripts/validate-transcodes.go --no-probe ./data/vimesrv.db /mnt/media
```

### Verbose output

Show detailed information for all media files, not just failures:

```bash
go run scripts/validate-transcodes.go --verbose ./data/vimesrv.db /mnt/media
```

### Fix corrupted segments.json

Regenerate any missing or corrupted segments.json files:

```bash
go run scripts/validate-transcodes.go --fix-segments ./data/vimesrv.db /mnt/media
```

### Custom drift threshold

Use a stricter 100ms threshold for audio sync validation:

```bash
go run scripts/validate-transcodes.go --drift-threshold 100 ./data/vimesrv.db /mnt/media
```

### Clean up orphaned data

Remove transcode directories and database records that are out of sync:

```bash
go run scripts/validate-transcodes.go --fix-orphaned-dirs --fix-orphaned-records ./data/vimesrv.db /mnt/media
```

### Queue audio re-transcoding

Create transcode jobs for media with audio sync issues:

```bash
go run scripts/validate-transcodes.go --create-transcode-jobs ./data/vimesrv.db /mnt/media
```

### Dry run

Preview what changes would be made without actually modifying anything:

```bash
go run scripts/validate-transcodes.go --dry-run --fix-segments --create-transcode-jobs ./data/vimesrv.db /mnt/media
```

### JSON output

Get machine-readable output for scripting:

```bash
go run scripts/validate-transcodes.go --json ./data/vimesrv.db /mnt/media
```

## Output

### Console Output

```
Validating transcodes...
Database: ./data/vimesrv.db
Media root: /mnt/media
Drift threshold: 200ms
Mode: Probing segments with ffprobe

[PASS]  Citizen_Kane_1941.mkv
        Video: 7164616ms, Audio: 7164629ms, Drift: 13ms

[FAIL]  Dune_2021.mkv
        Video: 9324982ms, Audio: 9354005ms, Drift: 29023ms
        * Audio/video drift exceeds threshold (29023ms > 200ms)
        -> Re-transcode audio tracks

[PASS]  Inception_2010.mkv
        Video: 8127453ms, Audio: 8127701ms, Drift: 248ms

================================================================================
Summary:
  Total media files: 10
  Passed: 9
  Warnings: 0
  Failures: 1

Issues found:
  - 1 media file(s) with audio sync drift > 200ms

Media requiring audio re-transcode:
  - Dune_2021.mkv (89d3c76f-29fa-55a9-2837-5e6fed5dea25)

Run with --create-transcode-jobs to queue audio re-transcoding
```

### JSON Output

```json
{
  "total_media": 10,
  "passed": 9,
  "warnings": 0,
  "failures": 1,
  "orphaned_dirs": 0,
  "orphaned_records": 0,
  "drift_issues": 1,
  "missing_segments": 0,
  "corrupted_segments": 0,
  "results": [
    {
      "media_id": "89d3c76f-29fa-55a9-2837-5e6fed5dea25",
      "filename": "Dune_2021.mkv",
      "status": "FAIL",
      "video_duration_ms": 9324982,
      "audio_duration_ms": 9354005,
      "drift_ms": 29023,
      "issues": ["Audio/video drift exceeds threshold (29023ms > 200ms)"],
      "actions": ["Re-transcode audio tracks"]
    }
  ],
  "media_needing_retranscode": ["89d3c76f-29fa-55a9-2837-5e6fed5dea25"]
}
```

## Audio/Video Sync Drift

### What is drift?

Audio/video drift occurs when the total duration of audio segments differs from the total duration of video segments. Over time, small differences in segment timing accumulate, causing audio to gradually become out of sync with video.

### Causes

1. **Source file issues**: Some source files have pre-existing timestamp problems
2. **Transcoding artifacts**: Audio and video are transcoded separately with different segmentation timing

### Threshold

The default drift threshold is **200ms**, which is generally imperceptible for most content. You can adjust this based on your needs:

- **100ms**: Strict, for content where sync is critical (e.g., dialogue-heavy)
- **200ms**: Default, good balance for most content
- **500ms**: Lenient, for content where minor drift is acceptable
- **5000ms**: Very lenient, only flags severe issues

### Fixing drift

When drift is detected, the solution is to re-transcode the **audio tracks only** (video is fine). The updated audio transcoding pipeline includes timestamp correction to prevent drift.

1. Run with `--create-transcode-jobs` to queue re-transcoding
2. Or manually trigger audio re-transcode from the web UI

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All validations passed |
| 1 | One or more validations failed |

## Performance

- **With ffprobe** (default): Accurate but slow, probes each segment file
- **With `--no-probe`**: Fast, trusts existing segments.json files

For routine checks, use `--no-probe`. For thorough validation or after suspected corruption, run without `--no-probe`.

## Related Scripts

- `scripts/regenerate-segments.go` - Regenerates segments.json files with correct sorting (fixes a specific bug with segment ordering)
