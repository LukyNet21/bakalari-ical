# Bakalari iCal

Convert your Bakalari school timetable into an iCalendar feed that works with Google Calendar, Apple Calendar, Outlook, and other calendar apps.

## Prerequisites

- Docker and Docker Compose

## Quick Start

### 1. Download compose file
```bash
mkdir bakalari-ical && cd bakalari-ical
wget https://raw.githubusercontent.com/LukyNet21/bakalari-ical/main/compose.yaml
```

Or with curl:
```bash
mkdir bakalari-ical && cd bakalari-ical
curl -O https://raw.githubusercontent.com/LukyNet21/bakalari-ical/main/compose.yaml
```

### 2. Generate encryption key
```bash
export ENCRYPTION_KEY=$(docker run --rm ghcr.io/lukynet21/bakalari-ical:dev /newkey | tail -1)
echo "ENCRYPTION_KEY=$ENCRYPTION_KEY" > .env
```

### 3. Create config file (empty)
```bash
echo '{"calendars":[]}' > config.json
```

### 4. Run configuration wizard
```bash
docker run -it --rm \
  -e ENCRYPTION_KEY="$ENCRYPTION_KEY" \
  -v $(pwd)/config.json:/config.json \
  ghcr.io/lukynet21/bakalari-ical:dev /newcal
```

Follow the prompts to add your calendar credentials.

### 5. Start the service
```bash
docker compose up -d
```

The service is now running at `http://localhost:8080`

## Adding Multiple Calendars

Run the configuration wizard again to add more calendars:

```bash
source .env
docker run -it --rm \
  -e ENCRYPTION_KEY="$ENCRYPTION_KEY" \
  -v $(pwd)/config.json:/config.json \
  ghcr.io/lukynet21/bakalari-ical:dev /newcal
```

## Using Your Calendar

Your calendar feed is available at:
```
http://localhost:8080/calendar.ics?token=<token>
```

Add this URL to your calendar app (Google Calendar, Apple Calendar, Outlook, etc.).

## Config File

The `config.json` file stores your encrypted credentials:

```json
{
  "calendars": [
    {
      "name": "My Timetable",
      "weeks_past": 2,
      "weeks_future": 6,
      "base_url": "https://school.bakalari.cz",
      "username": "your_username",
      "enc_password": "encrypted_password",
      "token": "unique_token"
    }
  ]
}
```

## Security

- Passwords are encrypted with ChaCha20-Poly1305
- Keep `ENCRYPTION_KEY` safe (store in `.env`)
- Don't commit `.env` or `config.json` to version control
- Use HTTPS in production
