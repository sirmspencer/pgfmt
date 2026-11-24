create table if not exists users (
  id         bigserial   primary key,
  email      text        not null unique,
  tenant_id  bigint      references tenants (id) on delete cascade,
  active     boolean     not null default true,
  created_at timestamptz not null default now(),
  constraint users_email_check check (email <> '')
);
