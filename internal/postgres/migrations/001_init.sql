CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE tenants (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE api_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    token_hash bytea NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    role text NOT NULL CHECK (role IN ('admin', 'cataloger', 'stock', 'reader')),
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id)
);

CREATE TABLE branches (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    name_key text NOT NULL,
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, name_key)
);

CREATE TABLE brands (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    name text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    name_key text NOT NULL,
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, name_key)
);

CREATE TABLE products (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    brand_id uuid NOT NULL,
    sku text NOT NULL CHECK (length(btrim(sku)) BETWEEN 1 AND 80),
    sku_key text NOT NULL,
    manufacturer_code text NOT NULL CHECK (length(btrim(manufacturer_code)) BETWEEN 1 AND 120),
    code_key text NOT NULL,
    description text NOT NULL CHECK (length(btrim(description)) BETWEEN 1 AND 300),
    unit text NOT NULL DEFAULT 'UN' CHECK (unit = 'UN'),
    search_text text GENERATED ALWAYS AS (lower(sku || ' ' || manufacturer_code || ' ' || description)) STORED,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, sku_key),
    UNIQUE (tenant_id, brand_id, code_key),
    FOREIGN KEY (tenant_id, brand_id) REFERENCES brands(tenant_id, id)
);
CREATE INDEX products_search_idx ON products USING gin (search_text gin_trgm_ops);

CREATE TABLE stock_balances (
    tenant_id uuid NOT NULL,
    branch_id uuid NOT NULL,
    product_id uuid NOT NULL,
    quantity bigint NOT NULL DEFAULT 0 CHECK (quantity BETWEEN 0 AND 9000000000000),
    PRIMARY KEY (tenant_id, branch_id, product_id),
    FOREIGN KEY (tenant_id, branch_id) REFERENCES branches(tenant_id, id),
    FOREIGN KEY (tenant_id, product_id) REFERENCES products(tenant_id, id)
);

CREATE TABLE stock_movements (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    branch_id uuid NOT NULL,
    product_id uuid NOT NULL,
    kind text NOT NULL CHECK (kind IN ('entrada', 'saida')),
    quantity bigint NOT NULL CHECK (quantity BETWEEN 1 AND 1000000000),
    reason text NOT NULL CHECK (length(btrim(reason)) BETWEEN 1 AND 300),
    idempotency_key text NOT NULL CHECK (idempotency_key ~ '^[a-zA-Z0-9_-]{16,100}$'),
    created_by uuid NOT NULL,
    balance_after bigint NOT NULL DEFAULT 0 CHECK (balance_after BETWEEN 0 AND 9000000000000),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key),
    FOREIGN KEY (tenant_id, branch_id) REFERENCES branches(tenant_id, id),
    FOREIGN KEY (tenant_id, product_id) REFERENCES products(tenant_id, id),
    FOREIGN KEY (tenant_id, created_by) REFERENCES api_keys(tenant_id, id)
);
CREATE INDEX stock_movements_history_idx ON stock_movements (tenant_id, branch_id, created_at DESC, id DESC);
