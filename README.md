
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
```

Load the variables into your shell before starting the backend:

```bash
set -a
source .env
set +a
```

### 4. Apply database migrations

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

- **Dashboard:** Overview of products, licensing status, and pending reviews.
- **Seller:** Browse seller products, inspect hold reasons, and submit licence applications with supporting files.
- **Reviewer:** Review pending applications, inspect affected products, and approve or reject applications.
- **Buyer:** Browse the catalogue containing only active products that are not on hold.
- **Settings:** Change the application-wide current date to test licence validity and expiration behavior.

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

| Method | Endpoint | Description |
| --- | --- | --- |
| GET | `/health` | Database and API health |
| GET | `/api/products` | List products with licensing information |
| GET | `/api/sellers/{sellerID}/products` | List products belonging to a seller |
| POST | `/api/licences` | Submit a licence application |
| GET | `/api/reviews` | List pending licence applications |
| PATCH | `/api/reviews/{id}` | Approve or reject an application |
| GET | `/api/catalogue` | List products visible to buyers |
| GET | `/api/settings/today` | Get the application-wide date |
| PUT | `/api/settings/today` | Update the application-wide date |

Licence submissions support JSON and multipart form data. Supporting files are optional; accepted file extensions are PDF, PNG, JPG, and JPEG, with a 10 MB upload limit.

## Testing and Code Quality

Run backend tests:

```bash
go test ./cmd/... ./internal/...
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
│   └── api/
├── internal/
│   ├── config/
│   ├── database/
│   ├── category/
│   ├── product/
│   ├── licensing/
│   ├── review/
│   ├── catalogue/
│   ├── settings/
│   └── httpapi/
├── migrations/
├── data/
├── uploads/
├── web/
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## Notes

- Initial data is loaded automatically on first startup when the database has not been initialized.
- Existing database contents are not automatically reseeded on subsequent startups.
- Uploaded files are stored locally in the configured upload directory.
- The application uses an explicit SQL-based data access layer rather than an ORM.
