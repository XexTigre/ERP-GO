-- Runtime uses a non-owner, non-superuser role. Owners retain maintenance
-- access; migration/admin credentials must never be used to serve requests.
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON tenants
USING (id = nullif(current_setting('app.tenant_id', true), '')::uuid)
WITH CHECK (id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
CREATE POLICY key_tenant_read ON api_keys FOR SELECT USING (
    tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid
    OR token_hash = decode(nullif(current_setting('app.key_hash', true), ''), 'hex')
);
CREATE POLICY key_tenant_write ON api_keys FOR ALL
USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE branches ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON branches
USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);
ALTER TABLE brands ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON brands
USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);
ALTER TABLE products ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON products
USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);
ALTER TABLE stock_balances ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON stock_balances
USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);
ALTER TABLE stock_movements ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_scope ON stock_movements
USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid);

DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='erp_app') THEN
  GRANT USAGE ON SCHEMA public TO erp_app;
  GRANT SELECT ON tenants,api_keys TO erp_app;
  GRANT SELECT,INSERT ON brands,branches,products,stock_movements TO erp_app;
  GRANT SELECT,INSERT,UPDATE ON stock_balances TO erp_app;
  REVOKE ALL ON schema_migrations FROM erp_app;
 END IF;
END $$;
