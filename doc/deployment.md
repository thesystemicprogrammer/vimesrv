# Deployment

## Production Build

```bash
make build-pwa
```

Creates a single binary with embedded PWA at `./bin/vimesrv`.

## Configuration

### Production Config

Create `/etc/vimesrv/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  shutdown_timeout_seconds: 30

auth:
  enabled: true
  username: "admin"
  password_hash: "$2b$10$your-bcrypt-hash"
  jwt_secret: ""  # Use AUTH_JWT_SECRET env var
  token_expiry_hours: 24

media:
  library_path: "/data/vimesrv/library"
  media_path: "/data/vimesrv/media"
  staging_path: "/data/vimesrv/staging"
  transcode_timeout_seconds: 14400  # 4 hours for large files

transcoding:
  quality_profiles:
    - name: "1080p"
      enabled: true
      resolution: "1920x1080"
      crf: 21
      max_bitrate: "5500k"
      audio_bitrate: "192k"
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      crf: 23
      max_bitrate: "2800k"
      audio_bitrate: "128k"
    - name: "480p"
      enabled: true
      resolution: "854x480"
      crf: 24
      max_bitrate: "1500k"
      audio_bitrate: "128k"

database:
  path: "/data/vimesrv/vimesrv.db"

logging:
  level: "info"
  format: "json"
  file: "/var/log/vimesrv/vimesrv.log"

tmdb:
  enabled: true
  language: "en-US"
  image_cache_path: "/data/vimesrv/cache/tmdb"
```

### Environment Variables

```bash
export AUTH_JWT_SECRET="your-secure-random-secret"
export TMDB_API_KEY="your-tmdb-api-key"
```

Generate a secure secret:

```bash
openssl rand -base64 32
```

## Systemd Service

Create `/etc/systemd/system/vimesrv.service`:

```ini
[Unit]
Description=VimeSrv Media Server
After=network.target

[Service]
Type=simple
User=vimesrv
Group=vimesrv
WorkingDirectory=/opt/vimesrv
ExecStart=/opt/vimesrv/vimesrv --config /etc/vimesrv/config.yaml
Restart=on-failure
RestartSec=5

# Environment
Environment=AUTH_JWT_SECRET=your-secret
Environment=TMDB_API_KEY=your-key
EnvironmentFile=-/etc/vimesrv/env

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/data/vimesrv /var/log/vimesrv

[Install]
WantedBy=multi-user.target
```

### Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable vimesrv
sudo systemctl start vimesrv
```

### View Logs

```bash
sudo journalctl -u vimesrv -f
```

## Reverse Proxy

### Nginx

```nginx
upstream vimesrv {
    server 127.0.0.1:8080;
}

server {
    listen 443 ssl http2;
    server_name media.example.com;

    ssl_certificate /etc/letsencrypt/live/media.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/media.example.com/privkey.pem;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Streaming optimization
    proxy_buffering off;
    proxy_request_buffering off;

    location / {
        proxy_pass http://vimesrv;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support (if needed)
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # Cache static PWA assets
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
        proxy_pass http://vimesrv;
        proxy_cache_valid 200 1d;
        add_header Cache-Control "public, max-age=86400";
    }

    # Streaming - disable buffering
    location /stream/ {
        proxy_pass http://vimesrv;
        proxy_buffering off;
        proxy_cache off;
    }
}

server {
    listen 80;
    server_name media.example.com;
    return 301 https://$server_name$request_uri;
}
```

### Caddy

```caddyfile
media.example.com {
    reverse_proxy localhost:8080

    @static {
        path *.js *.css *.png *.jpg *.ico *.svg *.woff *.woff2
    }
    header @static Cache-Control "public, max-age=86400"

    @stream {
        path /stream/*
    }
    reverse_proxy @stream localhost:8080 {
        flush_interval -1
    }
}
```

## Directory Setup

```bash
# Create user
sudo useradd -r -s /bin/false vimesrv

# Create directories
sudo mkdir -p /opt/vimesrv
sudo mkdir -p /etc/vimesrv
sudo mkdir -p /data/vimesrv/{library,media,staging,cache/tmdb}
sudo mkdir -p /var/log/vimesrv

# Set ownership
sudo chown -R vimesrv:vimesrv /data/vimesrv
sudo chown -R vimesrv:vimesrv /var/log/vimesrv

# Deploy binary
sudo cp ./bin/vimesrv /opt/vimesrv/
sudo chown vimesrv:vimesrv /opt/vimesrv/vimesrv
sudo chmod 755 /opt/vimesrv/vimesrv
```

## Security Considerations

### Authentication

- Always use a strong `jwt_secret` (32+ random bytes)
- Use bcrypt password hashes with high cost factor
- Keep `token_expiry_hours` reasonable (24h or less)
- Use short `stream_token_mins` (60 or less)

### Network

- Run behind a reverse proxy with TLS
- Don't expose port 8080 directly to the internet
- Use firewall rules to restrict access

### File Permissions

- Run as dedicated non-root user
- Restrict write access to data directories only
- Use `ProtectSystem=strict` in systemd

### Secrets

- Use environment variables for secrets
- Don't commit secrets to version control
- Use `/etc/vimesrv/env` file with restricted permissions:

```bash
sudo chmod 600 /etc/vimesrv/env
sudo chown root:root /etc/vimesrv/env
```

## Monitoring

### Health Check

```bash
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer <token>"
```

### Log Aggregation

With JSON logging enabled, forward logs to your aggregation system:

```bash
journalctl -u vimesrv -o json | your-log-shipper
```

## Backup

### Database

```bash
sqlite3 /data/vimesrv/vimesrv.db ".backup /backup/vimesrv-$(date +%Y%m%d).db"
```

### Media Files

Back up these directories:
- `/data/vimesrv/library/` - Organized media
- `/data/vimesrv/vimesrv.db` - Database

Transcodes can be regenerated and don't need backup.
