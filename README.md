# Domaindex

Self-hosted domain manager, all of your countless domains (*sure, you will finish that side-project*) in one place. (Domain + Index = Domaindex)

![Domaindex](art/01-light.png#gh-light-mode-only)
![Domaindex](art/01-dark.png#gh-dark-mode-only)

## Features

- Add domains, bulk import from list
- Automatically fetches WHOIS information, incl. expiry date and registrar information
- Wishlist domains
- Get notified about soon to expire domains ([Apprise](https://github.com/caronc/apprise) integration)
- Built-in list of most popular registrars + prices for TLDs
- Flexible price-management, set per registrar prices, custom overrides, prices initial costs, yearly renewals, transfers, privacy add-ons etc.
- Multi-user application
- Single Docker image for easy deployment
- **No JavaScript** across the whole application

### Planed Features

- [ ] SSL check
- [ ] Domain A/AAAA host information (ASN, country, ...)
- [x] CSS darkmode
- [x] Cache CSS assets, cache-bust with current version

## Deployment

See example [Docker compose.yml](/compose.yml)

```bash
docker run -d \
  --name domaindex \
  -p 8080:8080 \
  -v ./data:/app/data \
  -e APP_SECRET=change-me-to-a-random-secret \
  -e DB_DRIVER=sqlite \
  -e DB_DSN=data/domaindex.db \
  ghcr.io/romanzipp/domaindex:main
```

## Configuration

- `APP_HOST` - Listen host (default: `0.0.0.0`)
- `APP_PORT` - Listen port (default: `8080`)
- `APP_SECRET` - Session cookie secret, generate a random secret by running `openssl rand -hex 32`
- `WHOIS_REFRESH_INTERVAL` - How often to refresh WHOIS data in the background (default: `6h`)
- `REGISTRATION_ENABLED` - Allow new user registration (default: `true`)

### Database

#### SQLite (default)

- `DB_DRIVER` - `sqlite`
- `DB_DSN` - Database file path (default: `data/domaindex.db`)

#### PostgreSQL

- `DB_DRIVER` - `postgres`
- `DB_DSN` - Connection string (default: `host=db user=domaindex password=secret dbname=domaindex port=5432 sslmode=disable`)

### Apprise (Notifications)

- `APPRISE_URL` - Apprise gateway URL
- `APPRISE_KEY` - Apprise key

## Details

### Default Registrars & Prices

The app comes with a set of default registrars (25 most popular) and default pricing which was fetched around April 2026 for the last time. You are free to update those default prices or set overrides for a single domain.

### Registrar Pricing

Information about domain pricing has been fetched from the following sources and are included in `.csv` seed files.

- `1910` Cloudflare: [cfdomainpricing JSON](https://cfdomainpricing.com/prices.json)

## Development

- Requirements: go 1.26+, node/npm for tailwind

See the [Makefile](/Makefile) for all available commands.

## Authors

- [Roman Zipp](https://romanzipp.com)

## License

MIT License
