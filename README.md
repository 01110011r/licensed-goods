
# Licensed Goods Marketplace

A marketplace backend and web interface for managing products that require regulatory licences. The system evaluates product licensing requirements, manages seller licence applications, supports reviewer decisions, and hides restricted products from the buyer catalogue.

## Tech Stack

- **Backend:** Go (`net/http`, `database/sql`)
- **Database:** PostgreSQL
- **PostgreSQL driver:** pgx
- **Frontend:** React, TypeScript, Vite
- **Frontend package manager:** Bun

## Prerequisites

Install the following before running the project:

- Go 1.27.4 or compatible
- PostgreSQL
- Bun
- Git

## Quick start with Docker

With Docker and Docker Compose v2 installed, one script runs everything:

```bash
scripts/run-all.sh
```

This starts a disposable test database, runs `go vet` and `go test` against it, runs the web lint and production build, and then starts the database, migrations, API and web app if the checks pass. The script prints the reviewer token for the run.

| Command | What it does |
| --- | --- |
| `scripts/run-all.sh` | Tests, then start the app |
| `scripts/run-all.sh test` | Only the tests |
| `scripts/run-all.sh up` | Only the app |
| `scripts/run-all.sh down` | Stop everything (keeps dev data) |
| `scripts/run-all.sh reset` | Stop everything and delete all volumes |

Default ports: web `5173`, API `8080`, dev database `5433`, test database `5434`. Override them with `WEB_PORT`, `API_PORT`, `DB_PORT` and `TEST_DB_PORT`. The dev database uses a separate port so it does not clash with a local PostgreSQL on `5432`.

## Getting Started

### 1. Clone the repository

```bash
git clone <repository-url>
cd licensed-goods
```

### 2. Create the database

Create a PostgreSQL database:

```bash
createdb licensed_goods
```

Alternatively, create it using your preferred PostgreSQL administration tool.

### 3. Configure environment variables

Copy the example configuration:

```bash
cp .env.example .env
```

Update `DATABASE_URL` in `.env` with your PostgreSQL connection details:

```env
APP_PORT=8080
APP_ENV=development

DATABASE_URL=postgres://postgres:password@localhost:5432/licensed_goods?sslmode=disable

UPLOAD_DIR=./uploads

# Required. Reviewers enter this token in the web app to access the review
# queue, review decisions and the Today setting.
REVIEWER_TOKEN=change-me
```

Set `REVIEWER_TOKEN` to a long random value. The server refuses to start without it.

To run the PostgreSQL integration tests, point `TEST_DATABASE_URL` at a dedicated test database. Without it those tests are skipped:

```bash
TEST_DATABASE_URL=postgres://postgres:password@localhost:5432/licensed_goods_test?sslmode=disable go test ./...
```

Load the variables into your shell before starting the backend:

```bash
set -a
source .env
set +a
```

### 4. Apply database migrations

Skip this step if you use Docker (`scripts/run-all.sh` applies the migrations itself).

From the project root:

```bash
psql "$DATABASE_URL" -f migrations/001_init.sql
```

### 5. Start the backend

From the project root:

```bash
go run ./cmd/api
```

The backend connects to PostgreSQL and loads the initial dataset from `data/data.json` if the database has not yet been initialized.

The API listens on port `8080` by default.

Check the health endpoint:

```bash
curl http://localhost:8080/health
```

### 6. Start the frontend

Open another terminal:

```bash
cd web
bun install
bun run dev
```

Open the local URL printed by Vite, usually:

```text
http://localhost:5173
```

## Application Workspaces

The application provides four workspaces:

- **Dashboard:** Overview of products and licensing status. The pending-review count appears only after a reviewer token is entered.
- **Seller:** Browse seller products, inspect hold reasons, and submit licence applications with a supporting file.
- **Reviewer:** Review pending applications, inspect affected products, and approve or reject applications. Requires the reviewer token.
- **Buyer:** Browse the catalogue containing only active products that are not on hold.
- **Settings:** Change the application-wide current date to test licence validity and expiration behavior. Requires the reviewer token.

The app has no user accounts. Reviewer pages ask for the reviewer token and keep it in `sessionStorage` for the current browser tab only.

## Licensing Rules

### Product requirements

A product can require one or more licence groups.

A group can match a product through:

- Category rules: a configured category also applies to its descendants.
- IKPU rules: include and exclude prefixes are evaluated using the longest matching prefix.

When matching prefixes have equal length, an exclusion takes precedence.

Category and IKPU matching are independent: a product requires a group if either matching path applies.

IKPU codes must contain exactly 17 digits.

### Licence validity

An approved licence is valid when:

- The current application date is on or after `validFrom`.
- `validTo` is either absent or the current application date is on or before it.

A product is placed on hold if it requires a licence group for which its seller does not have a valid approved licence.

Product status (`draft`, `active`, or `archived`) is independent of its licensing hold state.

### Application review

- A seller can have at most one pending application per licence group.
- Submitting a new application does not invalidate an existing approved licence.
- Approval requires valid references and a non-expired validity period.
- Rejection requires a reason.
- Approving or rejecting an application updates the products affected by that licence requirement.

The application-wide date setting makes it possible to test expiry and validity transitions without changing the system clock.

## API Endpoints

| Method | Endpoint | Access | Description |
| --- | --- | --- | --- |
| GET | `/health` | Public | Database and API health |
| GET | `/api/products` | Public | List products with licensing information |
| GET | `/api/sellers/{sellerID}/products` | Public | List products belonging to a seller |
| POST | `/api/licences` | Public | Submit a licence application (multipart form with a file) |
| GET | `/api/reviews` | Reviewer token | List pending licence applications |
| PATCH | `/api/reviews/{id}` | Reviewer token | Approve or reject an application |
| GET | `/api/catalogue` | Public | List products visible to buyers |
| GET | `/api/settings/today` | Public | Get the application-wide date |
| PUT | `/api/settings/today` | Reviewer token | Update the application-wide date |

Reviewer-token routes need `Authorization: Bearer <REVIEWER_TOKEN>`. Without a valid token they return `401`.

Licence submissions must use `multipart/form-data`. JSON is refused with `415`. Each submission must include one file: PDF, PNG, JPG, or JPEG, up to 10 MB, with contents that match the extension.

## Testing and Code Quality

Run everything in Docker (database, migrations, Go and web checks, then the app):

```bash
scripts/run-all.sh
```

Run backend tests locally. The PostgreSQL integration tests are skipped unless `TEST_DATABASE_URL` is set:

```bash
go test ./cmd/... ./internal/...
```

Run the PostgreSQL integration tests against a test database (not the development one):

```bash
TEST_DATABASE_URL=postgres://postgres:password@localhost:5432/licensed_goods_test?sslmode=disable \
  go test ./internal/review/ -run Integration -v
```

Run backend static analysis:

```bash
go vet ./cmd/... ./internal/...
```

Build the frontend:

```bash
cd web
bun run build
```

Run frontend linting:

```bash
bun run lint
```

## Project Structure

```text
licensed-goods/
├── cmd/
│   └── api/            entry point
├── internal/
│   ├── catalogue/      product listing and hold calculation
│   ├── config/         environment configuration
│   ├── database/       connection, seed loading
│   ├── httpapi/        HTTP handlers, reviewer-token middleware
│   ├── licensing/      matching engine, licence submission
│   ├── review/         review queue and decisions
│   └── settings/       application-wide date
├── migrations/         SQL schema
├── data/               seed dataset
├── docs/               overview and task description
├── scripts/            run-all.sh (full stack in Docker)
├── web/                React and Vite frontend
├── uploads/            licence documents (created at runtime)
├── Dockerfile          API image
├── docker-compose.yml  databases, migrations, API, web, test runners
├── .env.example
├── go.mod
├── go.sum
└── README.md
```

## Notes

- Initial data is loaded automatically on first startup when the database has not been initialized.
- Existing database contents are not automatically reseeded on subsequent startups.
- Uploaded files are stored locally in the configured upload directory.
- The application uses an explicit SQL-based data access layer rather than an ORM.
