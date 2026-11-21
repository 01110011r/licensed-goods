# Licensed Goods Marketplace — System Overview

## 1. Project Overview

The Licensed Goods Marketplace manages products that require specific licences before they can become available in the buyer catalogue.

The system supports four main workflows:

- **Seller:** manages products and submits licence applications.
- **Reviewer:** reviews pending licence applications and approves or rejects them. Reviewer access requires the reviewer token (see Access control below).
- **Buyer:** browses products that are eligible for sale.
- **Administrator:** controls the global business date (`Today`) used for licence validity calculations. In the current system this is a reviewer action and also requires the reviewer token.

The central business rule is:

> A product can appear in the buyer catalogue only when its status is `active` and the seller has valid approved licences for every licence group required by that product.

The project uses a **modular monolith architecture**: one Go backend application, internally separated into domain-oriented packages.

---

## 2. High-Level Architecture

```mermaid
flowchart TB
    subgraph Users
        Seller[Seller]
        Reviewer[Reviewer]
        Buyer[Buyer]
        Admin[Administrator]
    end

    subgraph Frontend["React + TypeScript"]
        SellerUI[Seller Workspace]
        ReviewerUI[Reviewer Workspace]
        BuyerUI[Buyer Catalogue]
        SettingsUI[Settings]
        Dashboard[Dashboard]
    end

    subgraph Backend["Go HTTP API — Modular Monolith"]
        HTTP[HTTP Handlers]

        subgraph Services["Application / Domain Services"]
            Licensing[Licensing Service]
            Matching[Matching Engine]
            Review[Review Service]
            Catalogue[Catalogue Service]
            Settings[Settings Service]
        end
    end

    subgraph Infrastructure
        DB[(PostgreSQL)]
        Files[Local File Storage]
    end

    Seller --> SellerUI
    Reviewer --> ReviewerUI
    Buyer --> BuyerUI
    Admin --> SettingsUI

    SellerUI --> HTTP
    ReviewerUI --> HTTP
    BuyerUI --> HTTP
    SettingsUI --> HTTP
    Dashboard --> HTTP

    HTTP --> Licensing
    HTTP --> Review
    HTTP --> Catalogue
    HTTP --> Settings

    Licensing --> DB
    Licensing --> Files
    Review --> DB
    Review --> Catalogue
    Catalogue --> Matching
    Catalogue --> DB
    Settings --> DB
```

### Architecture decisions

| Decision | Reason |
| --- | --- |
| Go standard library (`net/http`) | Lightweight HTTP server without unnecessary framework dependencies |
| PostgreSQL | Relational data storage and database-level constraints |
| Explicit SQL | Direct control over queries and database behavior |
| React + TypeScript | Component-based frontend with type safety |
| Modular monolith | Clear domain separation without distributed-system complexity |
| Local file storage | Simple document storage for the assessment scope |
| Docker Compose | One command runs the databases, migrations, checks, API and web app |

The backend is not a collection of microservices. All domain modules run inside one application process and share the same database connection.

---

## 3. Licence Matching Engine

The matching engine determines which licence groups a product requires.

There are two independent matching mechanisms:

1. **Category rules:** a rule can apply to a category and all its descendants.
2. **IKPU rules:** a rule matches an IKPU code by prefix.

A product requires a licence group if either matching mechanism matches that group.

### Matching flow

```mermaid
flowchart TD
    Start[Product] --> Category[Evaluate Category Rules]
    Start --> IKPU[Evaluate IKPU Rules]

    Category --> Descendants[Resolve Category and Descendants]
    Descendants --> CategoryMatch{Category Rule Matches?}

    IKPU --> Prefixes[Find Matching IKPU Prefixes]
    Prefixes --> Longest[Select Longest Matching Prefix]
    Longest --> Exclusion{Winning Rule Excludes?}

    Exclusion -->|No| IKPUResult[IKPU Match]
    Exclusion -->|Yes| NoIKPU[No IKPU Match]

    CategoryMatch -->|Yes| Required[Licence Group Required]
    CategoryMatch -->|No| CategoryResult[No Category Match]

    IKPUResult --> Required
    NoIKPU --> Combine[Combine Results]
    CategoryResult --> Combine
    Combine --> Result[Required Groups]
```

### Category matching

A category rule applies to the configured category and its descendants.

For example:

```text
Health
├── Medicines
│   ├── Painkillers
│   └── Vitamins
└── Cosmetics
```

If a licence group has a rule for `Medicines`, products in these categories require that licence group:

- Medicines
- Painkillers
- Vitamins

Products in `Cosmetics` do not match that category rule.

### IKPU prefix matching

IKPU rules use prefix matching.

When multiple rules match the same code, the **longest matching prefix wins**.

For example:

| Prefix | Rule |
| --- | --- |
| `30049` | Include |
| `300490` | Exclude |

For the code `30049000100000001`, both prefixes match. The longer prefix (`300490`) wins, so the IKPU matching path excludes the product.

### Important: matching paths are independent

A category match is not cancelled by an IKPU exclusion.

For example, the MED group has:

- Category rule: `Medicines`
- IKPU include rule: `30049`
- IKPU exclude rule: `300490`

A Paracetamol product in the `Painkillers` category with IKPU `30049000100000001` still requires MED because its category matches the `Medicines` rule.

This is an intentional **OR relationship**:

```text
Required licence group =
    Category rule matches
    OR
    IKPU rule matches
```

A product may also require multiple licence groups.

### Data validation

The matching engine also reports product data issues, including:

- Invalid IKPU length (expected 17 digits).
- Unknown category.
- Missing category parent.
- Category hierarchy cycle.

Invalid data in one matching path does not automatically disable the independent matching path.

---

## 4. Product Availability and Licence Validity

After identifying the required licence groups, the catalogue service checks whether the seller has valid approved licences for all of them.

### Licence validity

A licence is valid when:

```text
status = approved
AND today >= validFrom
AND (validTo IS NULL OR today <= validTo)
```

The start and end dates are inclusive.

If `validTo` is `NULL`, the licence has no configured expiration date.

### Product availability flow

```mermaid
flowchart TD
    Product[Product] --> Match[Determine Required Licence Groups]
    Match --> Groups{Any Required Groups?}

    Groups -->|No| ActiveCheck{Product Status Active?}
    Groups -->|Yes| Check[Check Seller's Licence for Each Group]

    Check --> Approved{Approved Licence Exists?}
    Approved -->|No| Hold[Product On Hold]
    Approved -->|Yes| Validity{Valid on Today?}

    Validity -->|No| Hold
    Validity -->|Yes| More{More Required Groups?}

    More -->|Yes| Check
    More -->|No| ActiveCheck

    ActiveCheck -->|No| Hidden[Not in Buyer Catalogue]
    ActiveCheck -->|Yes| HoldCheck{On Hold?}

    HoldCheck -->|Yes| Hidden
    HoldCheck -->|No| Visible[Visible in Buyer Catalogue]
```

### Hold and product status are separate

Product status represents the product's lifecycle:

- `draft`
- `active`
- `archived`

Licence hold is a separate, derived availability condition.

For example:

| Product status | On hold | Buyer catalogue |
| --- | ---: | --- |
| active | No | Visible |
| active | Yes | Hidden |
| draft | No | Hidden |
| archived | No | Hidden |

This separation matters because licence expiration should not overwrite the seller's original product status.

### Dynamic hold calculation

The system calculates licence validity and product hold when products are requested.

It does not rely on a background process to periodically update every product's hold status.

Consequently:

- Approving a licence can release matching products.
- Expiring a licence can put matching products on hold.
- Changing the global Today date recalculates availability on the next request.

This avoids maintaining a potentially stale, separately stored hold flag.

---

## 5. Licence Submission and Review Workflow

Sellers submit licence applications for specific licence groups.

The application is initially created with `pending` status.

### Submission flow

```mermaid
flowchart TD
    Seller[Seller Submits Application] --> Validate[Validate Required Fields, Dates and File]
    Validate --> References[Verify Seller and Licence Group]
    References --> Pending{Pending Application Already Exists?}

    Pending -->|Yes| Reject[Reject Submission]
    Pending -->|No| File[Save Required Licence Document]

    File --> Insert[Insert Pending Licence]
    Insert --> Queue[Reviewer Queue]
```

Applications are submitted as `multipart/form-data`. JSON requests are refused with `415 Unsupported Media Type`, because every licence must include a document.

The submission process validates:

- Seller and licence group references.
- Licence number and issuer.
- Required start date.
- Optional end date.
- End date must not precede start date.
- Required licence document (see below).
- Only one pending application per seller and licence group.

### Licence document

Each application must include one file:

- Accepted extensions: `.pdf`, `.png`, `.jpg`, `.jpeg`.
- The file's contents must match its extension. The server checks the first bytes of the upload, so renaming a text file to `.pdf` is rejected.
- Maximum size: 10 MB.
- Files are saved under a random name in the upload directory. The client's file name is not used on disk.
- The stored path is set by the server only. A client cannot choose it.
- If the application is rejected after the file is saved, the file is deleted.

A partial unique index in PostgreSQL enforces the last condition:

```sql
CREATE UNIQUE INDEX ...
ON licences (seller_id, group_id)
WHERE status = 'pending';
```

This ensures the rule is enforced even if multiple requests arrive concurrently.

### Review queue

Viewing the reviewer queue requires the reviewer token (see Access control below).

The reviewer queue:

- Contains pending applications.
- Displays seller and licence group information.
- Shows licence metadata.
- Shows affected products.
- Orders applications by oldest submission first.

### Approval and rejection

```mermaid
flowchart TD
    Queue[Pending Application] --> Decision{Reviewer Decision}

    Decision -->|Approve| CheckApproval[Validate Approval]
    Decision -->|Reject| CheckReason[Validate Rejection Reason]

    CheckApproval --> Expiry{validTo Before Today?}
    Expiry -->|Yes| Block[Approval Rejected]
    Expiry -->|No| References[Verify Seller and Group]
    References --> Approved[Status = Approved]

    CheckReason --> Reason{Reason Provided?}
    Reason -->|No| Block
    Reason -->|Yes| Rejected[Status = Rejected]

    Approved --> Recalculate[Product Availability Recalculated]
    Rejected --> End[Application Remains Rejected]
```

Approval checks:

- The application is still pending.
- The expiration date is not before Today.
- The seller exists.
- The licence group exists.

Rejection requires a non-empty reason.

### Transaction safety

The review decision uses a database transaction and locks the target licence row with `SELECT ... FOR UPDATE`.

This prevents concurrent reviewers from making conflicting decisions about the same pending application.

### Existing approved licences

A new pending application does not invalidate an existing approved licence.

The existing licence remains valid until it expires, is revoked, or otherwise ceases to satisfy the validity conditions.

---

## 6. Global Today Setting

The application has a configurable global business date called `Today`.

This date is used for licence validity calculations instead of relying directly on the server's current date.

It is stored in PostgreSQL:

```text
app_settings
└── key: today
    value: YYYY-MM-DD
```

### Update flow

```mermaid
flowchart TD
    Admin[Administrator Changes Today] --> Request[PUT /api/settings/today with reviewer token]
    Request --> Auth{Token valid?}
    Auth -->|No| Refuse[401 Unauthorized]
    Auth -->|Yes| Validate[Validate YYYY-MM-DD]
    Validate --> Save[Update app_settings]
    Save --> NextRequest[Next Product or Catalogue Request]
    NextRequest --> Load[Load New Today]
    Load --> Validity[Recalculate Licence Validity]
    Validity --> Hold[Recalculate Product Holds]
    Hold --> Catalogue[Filter Buyer Catalogue]
```

Changing Today does not directly update product records.

Instead, the next product or catalogue request reads the updated date and recalculates licence validity.

This also makes it possible to test expiration scenarios without waiting for the actual calendar date.

---

## 7. API Overview

| Method | Endpoint | Access | Purpose |
| --- | --- | --- | --- |
| GET | `/health` | Public | Health check |
| GET | `/api/products` | Public | List products with licence requirements and hold information |
| GET | `/api/sellers/{sellerID}/products` | Public | List products belonging to a seller |
| GET | `/api/catalogue` | Public | List buyer-visible products |
| POST | `/api/licences` | Public | Submit a licence application (multipart, with file) |
| GET | `/api/reviews` | Reviewer token | Get pending licence applications |
| PATCH | `/api/reviews/{id}` | Reviewer token | Approve or reject an application |
| GET | `/api/settings/today` | Public | Get the current business date |
| PUT | `/api/settings/today` | Reviewer token | Update the business date |

Reviewer-token routes take `Authorization: Bearer <REVIEWER_TOKEN>`. Without a valid token they return `401 Unauthorized`. If the server has no `REVIEWER_TOKEN` configured, these routes return `403 Forbidden` for every request.

The backend validates incoming data independently of frontend validation. This means API requests made directly through `curl` are subject to the same business rules.

### Access control

The system has no user accounts. Access is controlled in two ways:

- **Public:** sellers and buyers use the application without signing in. Anyone who can reach the API can submit a licence application or read the catalogue.
- **Reviewer token:** a single shared secret, `REVIEWER_TOKEN`, protects the review queue, review decisions and the Today setting. The server refuses to start without it. The web app asks for the token when a reviewer or settings page is opened, keeps it in `sessionStorage` for that browser tab only, and clears it if the server rejects it.

This is a deliberate simplification for the assessment scope. It does not identify individual reviewers, so the database records each decision and its outcome but not who made it. Per-user accounts would replace the shared token.

---

## 8. Database Design

The main tables are:

| Table | Responsibility |
| --- | --- |
| `categories` | Product category hierarchy |
| `sellers` | Seller information |
| `licence_groups` | Licence group definitions |
| `licence_group_rules` | Category and IKPU matching rules |
| `products` | Seller products |
| `licences` | Licence applications and decisions |
| `app_settings` | Global application settings |

### Important database decisions

- Category hierarchy is represented using `parent_id`.
- Licence matching rules are stored separately from licence groups.
- Product status and licence status are independent.
- Licence applications have their own lifecycle.
- A partial unique index prevents duplicate pending applications.
- Foreign keys are intentionally not used in this assessment database so malformed seed records can be preserved and reported by application-level validation.

---

## 9. Testing Strategy

Testing focuses on business rules and important edge cases.

### Matching engine

Tests cover:

- Category descendant matching.
- IKPU prefix matching.
- Longest-prefix precedence.
- Exclusion rules.
- Independent category and IKPU matching.
- Multiple required licence groups.
- Invalid IKPU codes.
- Unknown categories.
- Missing category parents.
- Category cycles.

### Catalogue

Tests cover:

- Products without required licence groups.
- Products with all required licences.
- Products missing one of several required licences.
- Licences belonging to another seller.
- Buyer catalogue visibility rules.

### Licence submission

Tests cover:

- Valid submissions.
- Required fields.
- Required licence document.
- Whitespace normalization.
- Invalid dates.
- End date before start date.
- Same-day validity.

### HTTP layer

Tests cover:

- Reviewer token checks: missing, wrong, malformed and correct tokens, and fail-closed behaviour when no token is configured.
- Every reviewer route refuses requests without a token.
- Licence upload rules: JSON is refused, a missing file is refused, a file whose content does not match its extension is refused, and an unsupported extension is refused.

These tests do not need a database.

### Review decisions

Unit tests cover:

- Valid approval and rejection decisions.
- Rejection reason requirements.
- Invalid decision values.
- Approval validation.

Integration tests run against PostgreSQL and cover:

- Approving a valid licence and a licence without an end date.
- Rejecting with a trimmed reason.
- Refusing to approve an expired licence, which stays pending.
- Refusing a second decision on the same licence.
- Unknown licence IDs.

These run only when `TEST_DATABASE_URL` is set. Otherwise they are skipped.

### Today setting

Tests cover:

- Valid dates.
- Leap years.
- Invalid dates.
- Incorrect formatting.
- Missing leading zeros.
- Timestamp values where only a date is expected.

### Useful commands

Run everything in Docker (database, migrations, Go tests, web lint and build, then the app):

```bash
scripts/run-all.sh
```

Run only the checks, without starting the app:

```bash
scripts/run-all.sh test
```

Run backend tests locally, without the integration tests:

```bash
go test ./cmd/... ./internal/...
```

Run only the PostgreSQL integration tests locally. Use a test database, not the development one:

```bash
TEST_DATABASE_URL=postgres://postgres:password@localhost:5432/licensed_goods_test?sslmode=disable \
  go test ./internal/review/ -run Integration -v
```

Run backend static analysis:

```bash
go vet ./cmd/... ./internal/...
```

Check frontend production build:

```bash
cd web
bun run build
```

Run frontend lint:

```bash
bun run lint
```

---

## 10. Key Design Decisions — Defense Notes

### Why a modular monolith instead of microservices?

The assessment has a limited scope and a small number of domain modules. A modular monolith provides clear separation while avoiding unnecessary network communication, deployment complexity, and distributed transactions.

If the system grows, individual modules can be extracted later based on actual scaling or organizational needs.

### Why Go?

Go provides a simple concurrency model, good performance, static typing, and a standard library that is sufficient for building this API without adding unnecessary dependencies.

### Why PostgreSQL?

The application has relational data and business constraints that benefit from transactions, indexes, and database-level uniqueness guarantees.

### Why explicit SQL instead of an ORM?

Explicit SQL provides direct visibility into queries and database behavior. For this assessment, the schema and query requirements are manageable without an ORM.

### Why does the longest IKPU prefix win?

A more specific rule should override a broader rule. Longest-prefix matching provides deterministic behavior when several prefixes match the same code.

### Why are category and IKPU rules independent?

They represent two separate reasons why a product may require a licence. A match in either path is sufficient, so an exclusion in one path must not cancel a match in the other.

### Why separate hold from product status?

Product status describes the seller's product lifecycle. Hold describes whether the product currently satisfies licensing requirements. Keeping them separate prevents licence changes from overwriting product lifecycle information.

### Why calculate hold dynamically?

Licence validity depends on Today and the current approved licences. Calculating it when data is requested avoids stale hold information and removes the need for a scheduled expiry job.

### Why use a partial unique index?

The rule of one pending application per seller and licence group must hold even under concurrent submissions. A database constraint is more reliable than checking only in application code.

### Why use a transaction and `FOR UPDATE` during review?

Two reviewers might attempt to decide on the same application at nearly the same time. Row locking ensures that the decision is made against the current pending state.

### Why validate everything in the backend?

Frontend validation improves usability, but it cannot guarantee data integrity. Clients can bypass the UI and send requests directly. The backend is the authoritative enforcement point for business rules.

---

## 11. Recommended Defense Presentation Order

A clear presentation can follow this sequence:

1. **Architecture:** explain the modular monolith and the main components.
2. **Matching engine:** demonstrate category descendants and IKPU longest-prefix matching.
3. **Licence validity:** explain how approved licences are checked against Today.
4. **Product availability:** demonstrate why a product is placed on hold or shown in the buyer catalogue.
5. **Submission and review:** show the pending application workflow and reviewer decisions.
6. **Global Today:** demonstrate how changing the business date affects product availability.
7. **Testing:** show the important business-rule tests and edge cases.

### Most important source files to explain

| File | What to explain |
| --- | --- |
| `internal/licensing/matcher.go` | Category matching, IKPU matching, longest-prefix precedence |
| `internal/catalogue/products.go` | Required groups, licence validity, hold calculation |
| `internal/licensing/submission.go` | Submission validation and pending application creation |
| `internal/review/decision.go` | Approval/rejection rules and transaction safety |
| `internal/settings/today.go` | Global business date validation and persistence |
| `internal/httpapi/auth.go` | Reviewer token check (fails closed when no token is configured) |
| `internal/httpapi/licences.go` | Multipart upload handling and file content checks |
| `docker-compose.yml`, `scripts/run-all.sh` | Local stack: databases, migrations, tests, API and web |

**Defense tip:** Focus on explaining the business rules and why the implementation behaves correctly in edge cases. Understanding the reasoning behind the code is more valuable than simply listing the technologies used.
