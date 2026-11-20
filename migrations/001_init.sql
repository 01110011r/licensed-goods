

-- 001_init.sql

CREATE TABLE IF NOT EXISTS categories (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    parent_id   TEXT
);

CREATE TABLE IF NOT EXISTS sellers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS licence_groups (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS licence_group_rules (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    group_id    TEXT NOT NULL,
    kind        TEXT NOT NULL CHECK (kind IN ('category', 'ikpu')),
    category_id TEXT,
    prefix      TEXT,
    excluded    BOOLEAN NOT NULL DEFAULT FALSE,

    CHECK (
        (kind = 'category' AND category_id IS NOT NULL AND prefix IS NULL)
        OR
        (kind = 'ikpu' AND prefix IS NOT NULL AND category_id IS NULL)
    )
);

CREATE TABLE IF NOT EXISTS products (
    id          TEXT PRIMARY KEY,
    seller_id   TEXT NOT NULL,
    name        TEXT NOT NULL,
    category_id TEXT NOT NULL,
    ikpu        TEXT NOT NULL,
    status      TEXT NOT NULL CHECK (
        status IN ('draft', 'active', 'archived')
    )
);

CREATE TABLE IF NOT EXISTS licences (
    id              TEXT PRIMARY KEY,
    seller_id       TEXT NOT NULL,
    group_id        TEXT NOT NULL,
    number          TEXT NOT NULL DEFAULT '',
    issuer          TEXT NOT NULL DEFAULT '',
    valid_from      DATE NOT NULL,
    valid_to        DATE,
    status          TEXT NOT NULL CHECK (
        status IN ('pending', 'approved', 'rejected')
    ),
    rejection_reason TEXT,
    file_path       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS app_settings (
    key     TEXT PRIMARY KEY,
    value   TEXT NOT NULL
);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_categories_parent
    ON categories(parent_id);

CREATE INDEX IF NOT EXISTS idx_rules_group
    ON licence_group_rules(group_id);

CREATE INDEX IF NOT EXISTS idx_products_seller
    ON products(seller_id);

CREATE INDEX IF NOT EXISTS idx_products_category
    ON products(category_id);

CREATE INDEX IF NOT EXISTS idx_licences_seller_group
    ON licences(seller_id, group_id);

CREATE INDEX IF NOT EXISTS idx_licences_status_created
    ON licences(status, created_at);

-- Only one pending application per seller/group.
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_pending_licence
    ON licences(seller_id, group_id)
    WHERE status = 'pending';
