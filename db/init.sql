-- Dijalankan otomatis SEKALI oleh container Postgres, hanya saat volume masih kosong.
-- Untuk mengulang dari nol (data hilang): docker compose down -v
--
-- Satu server Postgres, dua database terpisah. Tidak ada foreign key lintas database.

CREATE DATABASE auth_db;
CREATE DATABASE shop_db;

-- =====================================================================
-- auth_db: dipakai auth-service
-- =====================================================================
\connect auth_db

CREATE TABLE users (
    id            SERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'customer' CHECK (role IN ('customer', 'admin')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Refresh token disimpan sebagai hash SHA-256 (bukan token aslinya),
-- sehingga bocornya database tidak langsung membocorkan sesi aktif.
CREATE TABLE refresh_tokens (
    id         SERIAL PRIMARY KEY,
    user_id    INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);

-- =====================================================================
-- shop_db: dipakai product-service
-- =====================================================================
\connect shop_db

-- pg_trgm membuat pencarian ILIKE '%kata%' bisa memakai index
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE products (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL,
    price       BIGINT NOT NULL CHECK (price >= 0),   -- rupiah, bilangan bulat
    stock       INT NOT NULL DEFAULT 0 CHECK (stock >= 0),
    image_url   TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_name_trgm ON products USING gin (name gin_trgm_ops);

CREATE TABLE orders (
    id         SERIAL PRIMARY KEY,
    user_id    INT NOT NULL,   -- tanpa FK: tabel users ada di database lain (auth_db)
    total      BIGINT NOT NULL CHECK (total >= 0),
    status     TEXT NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'paid', 'shipped', 'completed', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_user ON orders(user_id);

CREATE TABLE order_items (
    id         SERIAL PRIMARY KEY,
    order_id   INT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INT NOT NULL REFERENCES products(id),
    qty        INT NOT NULL CHECK (qty > 0),
    price      BIGINT NOT NULL CHECK (price >= 0)   -- harga saat dibeli
);

CREATE INDEX idx_order_items_order ON order_items(order_id);

CREATE TABLE cart_items (
    id         SERIAL PRIMARY KEY,
    user_id    INT NOT NULL,
    product_id INT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    qty        INT NOT NULL CHECK (qty > 0),
    UNIQUE (user_id, product_id)
);

CREATE INDEX idx_cart_items_user ON cart_items(user_id);
