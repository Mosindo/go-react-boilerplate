# Demo data (seed)

`services/api/cmd/seed` fills a **development** database with a small, realistic dataset so the mobile app has something to show.

```bash
# from services/api, API not required, database must be reachable
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
go run ./cmd/seed

# with Docker Compose (the image ships the binary)
docker compose exec api ./seed
```

## What it creates

- 24 users `demo01@demo.invalid` .. `demo24@demo.invalid` (the `.invalid` TLD can never receive mail). About 10 women, 10 men and 4 non-binary profiles, ages 22..44, 3..6 interests each, 1..3 generated photos each.
- **One shared password for every demo account: `DemoPass123!`** (documented on purpose; these accounts must never exist in production).
- Portraits are synthetic JPEGs generated in code (gradient background plus an abstract silhouette). There are no real people and no text in them.
- Locations are random points around a configurable city, rounded to 2 decimals exactly like the API does.
- A few swipes: six users have liked `demo01`, `demo01` passed four others, and five mutual matches with conversations exist (`demo01` with `demo02`..`demo04`, plus `demo15/demo16` and `demo17/demo18`), some with messages.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `DATABASE_URL` | required | Postgres connection string |
| `APP_ENV` | unset | **`production` makes the tool refuse to run** (exit 1) |
| `SEED_CITY_LAT` / `SEED_CITY_LON` | `48.8566` / `2.3522` | centre of the demo city |
| `SEED_CITY_LABEL` | `Demo City` | `locationLabel` shown on profiles |
| `SEED_RADIUS_KM` | `15` | users are spread inside this radius |

## Idempotency

Running the tool again does not duplicate anything: users are found by email, profiles, preferences, interests, swipes, matches and conversations use `ON CONFLICT DO NOTHING`, photos are only generated for users without any, and messages are only added to conversations that are still empty. It also applies pending migrations first.

## Try it

```bash
curl -s localhost:18080/auth/login -H 'content-type: application/json' \
  -d '{"email":"demo01@demo.invalid","password":"DemoPass123!"}'
```
