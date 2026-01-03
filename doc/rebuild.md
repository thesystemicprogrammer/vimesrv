# Database Rebuild

Safely migrate your VimeSrv database across major upgrades while preserving users, metadata links, and transcoded files.

## Overview

The database rebuild feature allows you to:

- **Upgrade VimeSrv** with breaking database schema changes
- **Preserve user accounts** including password hashes
- **Retain TMDB metadata links** via fingerprint-based matching
- **Recover existing transcodes** without re-encoding

## When to Use

Use the rebuild feature when:

- Upgrading to a new major version with schema changes
- The release notes indicate database incompatibility
- You need to migrate to a fresh database while keeping user data

Do **not** use for routine upgrades that don't require database changes.

## How It Works

1. **Prepare**: Export users and media-to-TMDB links to `rebuild.json`
2. **Upgrade**: Install new VimeSrv, delete old database
3. **Rebuild**: Import users, scan media library, auto-link via fingerprints, recover transcodes

The fingerprint (Blake2b hash of the video file) is used to re-establish TMDB metadata links. Since the video files haven't changed, the fingerprints match and metadata is automatically restored.

## Step-by-Step Guide

### Step 1: Prepare the Export

Run the prepare command to export your data:

```bash
./vimesrv --prepare-rebuild
```

This creates `{library_path}/rebuild.json` containing:
- All user accounts with password hashes
- Media fingerprint to TMDB ID mappings
- Edition information (e.g., "Director's Cut")

Verify the file was created:

```bash
ls -la /path/to/library/rebuild.json
```

### Step 2: Upgrade VimeSrv

```bash
# Stop the server
systemctl stop vimesrv  # or kill the process

# Backup the old database (just in case)
cp /data/vimesrv.db /data/vimesrv.db.backup

# Delete the old database
rm /data/vimesrv.db

# Install the new vimesrv binary
cp vimesrv-new /usr/local/bin/vimesrv
```

### Step 3: Enable Rebuild and Run

First, enable the rebuild flag in your config:

```yaml
rebuild:
  allow_rebuild: true
```

Then run the rebuild:

```bash
./vimesrv --rebuild-from-dump
```

The rebuild process will:
1. Clear the database (if any data exists)
2. Import all users
3. Scan the media library and create media records
4. Auto-link media to TMDB using fingerprints
5. Recover existing transcode records
6. Rename `rebuild.json` to `rebuild.json.done`

### Step 4: Verify and Cleanup

Check the logs for the rebuild summary:

```
[rebuild] Database rebuild complete
  users_imported=5
  files_processed=150
  files_linked=148
  transcodes_recovered=450
  errors=2
```

If there were errors, check `{library_path}/rebuild-errors.json`.

Disable the rebuild flag to prevent accidental use:

```yaml
rebuild:
  allow_rebuild: false  # Re-disable after successful rebuild
```

Start the server normally:

```bash
./vimesrv
```

## Configuration

```yaml
rebuild:
  allow_rebuild: false          # Must be true to enable --rebuild-from-dump
  tmdb_requests_per_10s: 15     # Rate limit during rebuild (lower than normal)
```

| Option | Default | Description |
|--------|---------|-------------|
| `allow_rebuild` | `false` | Safety flag to prevent accidental database wipes |
| `tmdb_requests_per_10s` | `15` | Conservative TMDB rate limit during rebuild |

## File Reference

| File | Location | Description |
|------|----------|-------------|
| `rebuild.json` | `{library_path}/` | Export data (users, media links) |
| `rebuild.json.done` | `{library_path}/` | Renamed after successful rebuild |
| `rebuild-errors.json` | `{library_path}/` | Errors encountered during rebuild |

### rebuild.json Structure

```json
{
  "version": 1,
  "created_at": "2025-01-03T10:00:00Z",
  "users": [
    {
      "id": "uuid",
      "username": "admin",
      "password_hash": "$2b$10$...",
      "role": "admin",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "media_links": [
    {
      "fingerprint": "abc123...",
      "metadata_type": "movie",
      "tmdb_id": 12345,
      "edition": "Director's Cut"
    }
  ]
}
```

## Troubleshooting

### "rebuild is disabled in configuration"

Set `rebuild.allow_rebuild: true` in your config file.

### "rebuild.json not found"

Run `--prepare-rebuild` first to create the export file.

### "unsupported rebuild.json version"

The rebuild.json was created by an incompatible VimeSrv version. Re-run `--prepare-rebuild` with the old version before upgrading.

### Some files not linked after rebuild

Check `rebuild-errors.json` for details. Common causes:
- File was moved or renamed (fingerprint changed)
- File was not linked to TMDB before the export
- File corruption changed the fingerprint

You can manually re-link these files via the API or web UI.

### Transcodes not recovered

Ensure the `{media_path}/{media_id}/transcoded/` directories exist and contain valid segment files. The recovery process looks for `init.mp4` or `.m4s` files.

## Security Considerations

- **Disable `allow_rebuild` after use**: This flag allows database destruction
- **Backup before rebuilding**: Keep the old database file until verified
- **Secure rebuild.json**: Contains password hashes (bcrypt, but still sensitive)
- **Trusted environment only**: Don't run rebuild on untrusted systems
