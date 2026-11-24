ALTER TABLE users ADD COLUMN last_login TIMESTAMPTZ, ADD COLUMN login_count INTEGER NOT NULL DEFAULT 0;
alter table users drop column legacy_flag;
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS users_email_idx ON users (lower(email)) WHERE deleted_at IS NULL;
create index orders_user_id_idx on orders (user_id);
