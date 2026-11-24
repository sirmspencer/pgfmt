alter table users
  add column last_login  timestamptz,
  add column login_count integer not null default 0;

alter table users drop column legacy_flag;

create unique index concurrently if not exists users_email_idx
  on users (lower(email))
  where deleted_at is null;

create index orders_user_id_idx on orders (user_id);
