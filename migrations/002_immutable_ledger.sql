-- Never rewrite 001_init.sql: deployed databases check its original checksum.
CREATE FUNCTION reject_stock_movement_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'stock_movements is append-only' USING ERRCODE = '55000';
END;
$$;
CREATE TRIGGER stock_movements_immutable
BEFORE UPDATE OR DELETE ON stock_movements
FOR EACH ROW EXECUTE FUNCTION reject_stock_movement_change();
CREATE TRIGGER stock_movements_no_truncate
BEFORE TRUNCATE ON stock_movements
FOR EACH STATEMENT EXECUTE FUNCTION reject_stock_movement_change();
ALTER TABLE stock_movements ALTER COLUMN balance_after DROP DEFAULT;
ALTER TABLE stock_movements ALTER COLUMN created_at SET DEFAULT clock_timestamp();

CREATE INDEX stock_balances_product_idx ON stock_balances (tenant_id, product_id);
CREATE INDEX stock_movements_product_idx ON stock_movements (tenant_id, product_id, created_at DESC, id DESC);
CREATE INDEX stock_movements_tenant_history_idx ON stock_movements (tenant_id, created_at DESC, id DESC);
CREATE INDEX stock_movements_key_author_idx ON stock_movements (tenant_id, created_by);
-- products(tenant_id,brand_id,code_key) and api_keys(tenant_id,id) already
-- support their tenant/brand and tenant foreign keys by their left prefixes.
