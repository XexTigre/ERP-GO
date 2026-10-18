CREATE EXTENSION IF NOT EXISTS unaccent;
-- Search is deliberately separate from the exact commercial identity.
CREATE FUNCTION catalog_search_key(value text) RETURNS text
LANGUAGE sql STABLE STRICT AS $$
    SELECT regexp_replace(lower(unaccent(value)), '[^[:alnum:]]', '', 'g');
$$;
ALTER TABLE products ADD COLUMN search_normalized text NOT NULL DEFAULT '';
UPDATE products p SET search_normalized=catalog_search_key(p.sku || ' ' || p.manufacturer_code || ' ' || p.description || ' ' || b.name)
FROM brands b WHERE b.tenant_id=p.tenant_id AND b.id=p.brand_id;
CREATE FUNCTION refresh_product_search() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE brand_name text;
BEGIN
    SELECT name INTO brand_name FROM brands WHERE tenant_id=NEW.tenant_id AND id=NEW.brand_id;
    NEW.search_normalized := catalog_search_key(NEW.sku || ' ' || NEW.manufacturer_code || ' ' || NEW.description || ' ' || coalesce(brand_name, ''));
    RETURN NEW;
END;
$$;
CREATE TRIGGER products_search_refresh BEFORE INSERT OR UPDATE ON products
FOR EACH ROW EXECUTE FUNCTION refresh_product_search();
CREATE FUNCTION refresh_brand_product_search() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    UPDATE products SET search_normalized=catalog_search_key(sku || ' ' || manufacturer_code || ' ' || description || ' ' || NEW.name)
    WHERE tenant_id=NEW.tenant_id AND brand_id=NEW.id;
    RETURN NEW;
END;
$$;
CREATE TRIGGER brands_search_refresh AFTER UPDATE OF name ON brands
FOR EACH ROW EXECUTE FUNCTION refresh_brand_product_search();
CREATE INDEX products_search_normalized_idx ON products USING gin(search_normalized gin_trgm_ops);

-- Existing IDs stay stable. UUID ordering across old/new records is not a
-- chronological guarantee; movement history uses (created_at,id) explicitly.
ALTER TABLE tenants ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE api_keys ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE branches ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE brands ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE products ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE stock_movements ALTER COLUMN id SET DEFAULT uuidv7();
