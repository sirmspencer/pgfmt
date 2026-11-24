*Rules document generated with LLM assistance*

# pgfmt Formatting Rules

This document lists the formatting rules pgfmt applies. Each rule shows input and the output pgfmt produces, and names the test case in `testdata/cases` that enforces it. Every example in this file is checked against the formatter.

✅ implemented · 🔒 required by the parser, not a style choice

## Index

**Case**

- [keyword-case](#keyword-case) ✅
- [identifier-case](#identifier-case) ✅
- [function-and-type-case](#function-and-type-case) ✅
- [literal-case](#literal-case) ✅

**Layout**

- [indentation](#indentation) ✅
- [leading-commas](#leading-commas) ✅
- [clause-items](#clause-items) ✅
- [conditions](#conditions) ✅
- [grouped-conditions](#grouped-conditions) ✅
- [joins](#joins) ✅
- [join-conditions](#join-conditions) ✅
- [subqueries](#subqueries) ✅
- [lists](#lists) ✅
- [ctes](#ctes) ✅
- [case-expressions](#case-expressions) ✅
- [set-operations](#set-operations) ✅
- [limit-and-offset](#limit-and-offset) ✅
- [insert](#insert) ✅
- [update](#update) ✅
- [delete](#delete) ✅
- [upsert](#upsert) ✅
- [create-table](#create-table) ✅
- [alter-table](#alter-table) ✅
- [create-index](#create-index) ✅
- [function-definitions](#function-definitions) ✅
- [statement-separation](#statement-separation) ✅
- [line-length](#line-length) ✅

**Normalization**

- [aliases](#aliases) 🔒
- [casts](#casts) 🔒
- [operators](#operators) 🔒
- [join-keywords](#join-keywords) 🔒
- [type-names](#type-names) 🔒
- [parentheses](#parentheses) 🔒
- [quoting](#quoting) 🔒

**Preservation**

- [comments](#comments) ✅
- [placeholders](#placeholders) ✅
- [literals](#literals) ✅
- [semicolons](#semicolons) ✅
- [unsupported-statements](#unsupported-statements) ✅

---

## Case

### keyword-case

**Keywords are lowercase.** Test case: `01-select-basic`.

```sql
SELECT u.id FROM users u WHERE u.active = TRUE;
```

↓

```sql
select u.id
from users as u
where u.active = true;
```

---

### identifier-case

**Unquoted identifiers are lowercase; quoted identifiers are unchanged.** PostgreSQL folds unquoted identifiers to lowercase, so `UserId` and `userid` name the same column. A quoted identifier is case sensitive and keeps its quotes. String contents are never changed. Test case: `21-identifier-case`.

```sql
SELECT UserId, "DisplayName" FROM Public.Users WHERE "Status" = 'Active';
```

↓

```sql
select
  userid
  , "DisplayName"
from public.users
where "Status" = 'Active';
```

---

### function-and-type-case

**Function names and type names are lowercase.** Test case: `11-functions-and-casts`.

```sql
SELECT COALESCE(nickname, first_name), NOW(), amount::NUMERIC(10,2) FROM orders;
```

↓

```sql
select
  coalesce(nickname, first_name)
  , now()
  , amount::numeric(10, 2)
from orders;
```

---

### literal-case

**`true`, `false`, and `null` are lowercase.** Test case: `22-operators-and-literals`.

```sql
select NULL, TRUE, FALSE;
```

↓

```sql
select
  null
  , true
  , false;
```

---

## Layout

### indentation

**Each level is indented by two spaces.** Tabs are not used. Every rule below that indents a line uses this unit.

---

### leading-commas

**When a list is split across lines, each line after the first starts with a comma.** This applies to every multi-line list: select items, `group by`, `order by`, `set`, `returning`, CTEs, long parenthesized lists, table columns, and `alter table` actions. Lists that stay on one line use ordinary `, ` separators. Test case: `01-select-basic`.

```sql
select id, email, created_at from users;
```

↓

```sql
select
  id
  , email
  , created_at
from users;
```

---

### clause-items

**A clause with one item keeps the item on the keyword's line. A clause with two or more items places the keyword alone on its line and each item on its own indented line.** The rule applies the same way to `select`, `group by`, `order by`, `set`, `returning`, `using`, and `values`. Test cases: `02-single-item-clauses`, `08-group-order-limit`.

One item in each clause:

```sql
select user_id from orders group by user_id order by user_id;
```

↓

```sql
select user_id
from orders
group by user_id
order by user_id;
```

Two or more items in each clause:

```sql
select user_id, status, count(*) from orders group by user_id, status order by user_id, count(*) desc;
```

↓

```sql
select
  user_id
  , status
  , count(*)
from orders
group by
  user_id
  , status
order by
  user_id
  , count(*) desc;
```

`distinct` and `distinct on (...)` stay with `select`:

```sql
select distinct status from orders;
```

↓

```sql
select distinct status
from orders;
```

---

### conditions

**`where` and `having` follow the clause-items rule, with `and` or `or` leading each condition after the first.** A single condition stays on the keyword's line. Two or more conditions place the keyword alone on its line and each condition on its own indented line. Test cases: `04-where-boolean`, `24-conditions`.

`where` with one condition:

```sql
select id from users where active=true;
```

↓

```sql
select id
from users
where active = true;
```

`where` with several conditions:

```sql
select id from users where active = true and created_at > :since and deleted_at is null;
```

↓

```sql
select id
from users
where
  active = true
  and created_at > :since
  and deleted_at is null;
```

`having` with one condition:

```sql
select user_id from orders group by user_id having count(*)>5;
```

↓

```sql
select user_id
from orders
group by user_id
having count(*) > 5;
```

`having` with several conditions:

```sql
select user_id from orders group by user_id having count(*) > 5 and sum(total) > 100;
```

↓

```sql
select user_id
from orders
group by user_id
having
  count(*) > 5
  and sum(total) > 100;
```

---

### grouped-conditions

**A group of `and`/`or` conditions inside a condition list opens a parenthesis at the end of the line, indents its contents, and closes the parenthesis on its own line.** A single negated condition such as `not (x like y)` stays on one line. Test case: `04-where-boolean`.

```sql
select id from users where active = true and (role = 'admin' or role = 'owner') and not (email like '%@example.com');
```

↓

```sql
select id
from users
where
  active = true
  and (
    role = 'admin'
    or role = 'owner'
  )
  and not (email like '%@example.com');
```

---

### joins

**Each join starts at the same level as `from`.** Test case: `03-joins`.

```sql
select u.id from users u join orders o on o.user_id = u.id cross join regions r;
```

↓

```sql
select u.id
from users as u
join orders as o
  on o.user_id = u.id
cross join regions as r;
```

---

### join-conditions

**`on` and `using` are on the line below the join, indented one level. Additional conditions align with `on`.** Test case: `03-joins`.

```sql
select p.name from orders o left join payments p on p.order_id = o.id and p.status = 'settled' join accounts a using (account_id);
```

↓

```sql
select p.name
from orders as o
left join payments as p
  on p.order_id = o.id
  and p.status = 'settled'
join accounts as a
  using (account_id);
```

---

### subqueries

**A subquery opens a parenthesis at the end of the line, indents its body, and closes the parenthesis on its own line, aligned with the start of the line that opened it.** This applies to `in`, `exists`, scalar subqueries, and subqueries in `from`. Test case: `05-subqueries`.

```sql
select id from users where id in (select user_id from orders where total > 100);
```

↓

```sql
select id
from users
where id in (
  select user_id
  from orders
  where total > 100
);
```

```sql
select t.user_id from (select user_id, sum(total) as total from orders group by user_id) t;
```

↓

```sql
select t.user_id
from (
  select
    user_id
    , sum(total) as total
  from orders
  group by user_id
) as t;
```

A scalar subquery in a select list:

```sql
select id, (select count(*) from orders o where o.user_id = u.id) as order_count from users u;
```

↓

```sql
select
  id
  , (
    select count(*)
    from orders as o
    where o.user_id = u.id
  ) as order_count
from users as u;
```

---

### lists

**A parenthesized list with three or fewer items stays on one line. A list with four or more items keeps the opening parenthesis on its line, places each item on its own indented line with leading commas, and puts the closing parenthesis on its own line.** This covers function arguments, `in (...)` value lists, `coalesce`, `greatest`, `least`, `array[...]`, row values, `insert` column lists, and `values` rows. Aggregate calls with `distinct` or `order by` inside the parentheses stay on one line, as do window specifications in `over (...)` and type modifiers such as `numeric(10, 2)`. Test case: `25-long-lists`.

`in` lists with three and four items:

```sql
select id from users where id in (1, 2, 3);
```

↓

```sql
select id
from users
where id in (1, 2, 3);
```

```sql
select id from users where id in (1, 2, 3, 4);
```

↓

```sql
select id
from users
where id in (
  1
  , 2
  , 3
  , 4
);
```

A function call with three arguments:

```sql
select substr(name, 1, 3) from users;
```

↓

```sql
select substr(name, 1, 3)
from users;
```

A function call with four or more arguments:

```sql
select id from users where created_at > make_timestamp(:year, :month, :day, 0, 0, 0);
```

↓

```sql
select id
from users
where created_at > make_timestamp(
  :year
  , :month
  , :day
  , 0
  , 0
  , 0
);
```

The count is per argument, so a function that takes key and value pairs breaks after two pairs:

```sql
select jsonb_build_object('id', u.id, 'name', u.name) from users u;
```

↓

```sql
select jsonb_build_object(
  'id'
  , u.id
  , 'name'
  , u.name
)
from users as u;
```

Nested lists follow the same rule at their own level:

```sql
select jsonb_build_object('id', u.id, 'tags', array[u.a, u.b, u.c, u.d]) from users u;
```

↓

```sql
select jsonb_build_object(
  'id'
  , u.id
  , 'tags'
  , array[
    u.a
    , u.b
    , u.c
    , u.d
  ]
)
from users as u;
```

`coalesce` with three and four arguments:

```sql
select coalesce(a, b, c), coalesce(a, b, c, d) from t;
```

↓

```sql
select
  coalesce(a, b, c)
  , coalesce(
    a
    , b
    , c
    , d
  )
from t;
```

---

### ctes

**Each common table expression opens with `name as (`, indents its body, and closes with `)` at the start of a line. Each later CTE starts its line with a comma.** Test case: `06-ctes`.

```sql
with active as (select id from users where active), totals as (select user_id from orders) select a.id from active a;
```

↓

```sql
with active as (
  select id
  from users
  where active
)
, totals as (
  select user_id
  from orders
)
select a.id
from active as a;
```

---

### case-expressions

**A `case` expression places each `when` and the `else` on its own indented line and closes with `end`.** Test case: `07-case-expressions`.

```sql
select CASE WHEN status='a' THEN 'active' ELSE 'unknown' END as label from users;
```

↓

```sql
select case
  when status = 'a' then 'active'
  else 'unknown'
end as label
from users;
```

With several select items, the `case` sits in the indented list:

```sql
select id, case role when 'admin' then 1 else 0 end as is_admin from users;
```

↓

```sql
select
  id
  , case role
    when 'admin' then 1
    else 0
  end as is_admin
from users;
```

---

### set-operations

**`union`, `intersect`, and `except` are alone on their lines, with a blank line before and after.** Test case: `12-set-operations`.

```sql
select id from users union all select id from archived_users;
```

↓

```sql
select id
from users

union all

select id
from archived_users;
```

---

### limit-and-offset

**`limit` and `offset` are each on their own line.** Test case: `08-group-order-limit`.

```sql
select id from users order by id limit 10 offset :offset;
```

↓

```sql
select id
from users
order by id
limit 10
offset :offset;
```

---

### insert

**The column list follows the [lists](#lists) rule, so it stays on the `insert into` line with three or fewer columns. `values` follows the [clause-items](#clause-items) rule: one row stays on the `values` line, and two or more rows are indented one per line. Each row follows the lists rule.** Test cases: `13-insert`, `25-long-lists`.

Three or fewer columns:

```sql
INSERT INTO users(id,email) VALUES(:id,:email) RETURNING id;
```

↓

```sql
insert into users (id, email)
values (:id, :email)
returning id;
```

Several rows:

```sql
insert into tags (name) values ('a'), ('b');
```

↓

```sql
insert into tags (name)
values
  ('a')
  , ('b');
```

Four or more columns:

```sql
insert into events (id, user_id, kind, created_at) values (:id, :user_id, :kind, now());
```

↓

```sql
insert into events (
  id
  , user_id
  , kind
  , created_at
)
values (
  :id
  , :user_id
  , :kind
  , now()
);
```

Four or more columns and several rows:

```sql
insert into events (id, user_id, kind, created_at) values (:id, :user_id, :kind, now()), (:id2, :user_id2, :kind2, now());
```

↓

```sql
insert into events (
  id
  , user_id
  , kind
  , created_at
)
values
  (
    :id
    , :user_id
    , :kind
    , now()
  )
  , (
    :id2
    , :user_id2
    , :kind2
    , now()
  );
```

From a select:

```sql
insert into archived_users (id, email) select id, email from users where deleted_at is not null;
```

↓

```sql
insert into archived_users (id, email)
select
  id
  , email
from users
where deleted_at is not null;
```

---

### update

**`set` follows the clause-items rule: one assignment stays on the `set` line, and several assignments are indented one per line.** Test case: `14-update`.

```sql
UPDATE users SET email=:email WHERE id=:id;
```

↓

```sql
update users
set email = :email
where id = :id;
```

```sql
UPDATE users SET email=:email, updated_at=now() WHERE id=:id;
```

↓

```sql
update users
set
  email = :email
  , updated_at = now()
where id = :id;
```

---

### delete

**`using`, `where`, and `returning` each start a line.** Test case: `15-delete`.

```sql
delete from sessions s using users u where u.id = s.user_id;
```

↓

```sql
delete from sessions as s
using users as u
where u.id = s.user_id;
```

---

### upsert

**`on conflict` starts a line. For `do update`, the `set` clause follows on the next line.** Test case: `16-upsert`.

```sql
insert into user_settings (user_id, key, value) values (:user_id, :key, :value) on conflict (user_id, key) do update set value = excluded.value, updated_at = now();
```

↓

```sql
insert into user_settings (user_id, key, value)
values (:user_id, :key, :value)
on conflict (user_id, key) do update
set
  value = excluded.value
  , updated_at = now();
```

---

### create-table

**The opening parenthesis ends the first line. Each column and table constraint is on its own indented line with leading commas. Columns are not aligned into a grid.** Test case: `17-create-table`.

```sql
CREATE TABLE IF NOT EXISTS users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL UNIQUE, CONSTRAINT users_email_check CHECK (email <> ''));
```

↓

```sql
create table if not exists users (
  id bigserial primary key
  , email text not null unique
  , constraint users_email_check check (email <> '')
);
```

---

### alter-table

**One action stays on the `alter table` line. Several actions are indented one per line with leading commas.** Test case: `19-alter-and-index`.

```sql
alter table users drop column legacy_flag;
```

↓

```sql
alter table users drop column legacy_flag;
```

```sql
ALTER TABLE users ADD COLUMN last_login TIMESTAMPTZ, ADD COLUMN login_count INTEGER NOT NULL DEFAULT 0;
```

↓

```sql
alter table users
  add column last_login timestamptz
  , add column login_count integer not null default 0;
```

Other single actions:

```sql
alter table users alter column email set not null;
```

↓

```sql
alter table users alter column email set not null;
```

```sql
alter table users add constraint users_email_key unique (email);
```

↓

```sql
alter table users add constraint users_email_key unique (email);
```

Several kinds of action together:

```sql
alter table users drop column legacy_flag, alter column email type varchar(320), alter column active set default true;
```

↓

```sql
alter table users
  drop column legacy_flag
  , alter column email type varchar(320)
  , alter column active set default true;
```

---

### create-index

**An index without a `where` clause stays on one line. A partial index places `on` and `where` on indented lines below the index name.** Test case: `19-alter-and-index`.

```sql
create index orders_user_id_idx on orders (user_id);
```

↓

```sql
create index orders_user_id_idx on orders (user_id);
```

Several columns, sort order, and an index method stay on one line. `using btree` is the default and the parser records it either way, so it is omitted:

```sql
CREATE INDEX events_user_created_idx ON events USING btree (user_id, created_at DESC);
```

↓

```sql
create index events_user_created_idx on events (user_id, created_at desc);
```

```sql
create index users_tags_idx on users using gin (tags);
```

↓

```sql
create index users_tags_idx on users using gin (tags);
```

```sql
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS users_email_idx ON users (lower(email)) WHERE deleted_at IS NULL;
```

↓

```sql
create unique index concurrently if not exists users_email_idx
  on users (lower(email))
  where deleted_at is null;
```

---

### function-definitions

**The function signature is on the first line, and its parameters follow the [lists](#lists) rule. `returns` and each option are indented one level on the lines below it. `as` and the body come last, starting at column 0, so the closing delimiter lines up with `as`.** Options written after the body are moved before it; PostgreSQL does not depend on their order. Test cases: `23-function-body-passthrough`, `26-function-bodies`.

The body is a string to the PostgreSQL parser, so pgfmt formats it separately, by language:

- **`sql`:** the body is formatted with every rule in this document. A body that formats to one line stays on the `as` line; a longer body is indented one level inside its dollar quotes.
- **`plpgsql`:** each standalone SQL statement is formatted with every rule in this document. A statement is standalone when it starts with `select`, `insert`, `update`, `delete`, or `with` after a statement boundary (`;`, `begin`, `then`, `else`, `loop`) or after `return query`, and ends at its semicolon. A standalone statement always starts on its own line; if it followed other code on the line, it moves to the next line one level deeper, and code after its semicolon, such as `end if;`, moves to its own line. The procedural code around the statements keeps its line structure, with keywords and unquoted identifiers lowercased. The body is indented one level inside its dollar quotes: the indentation its lines share is replaced by one level, and their own nesting is kept.
- **Other languages,** such as `plpython3u`, are left exactly as written.

Two kinds of SQL inside `plpgsql` are not formatted: SQL inside an expression, such as `if exists (select ...)` or `for r in select ... loop`, and statements that use plpgsql-only syntax, such as `select ... into strict`. Both are kept as written apart from case. Strings, quoted identifiers, and comments in any body are never changed.

A `plpgsql` trigger function:

```sql
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = now();
  RAISE NOTICE 'Updated %', NEW.id;
  RETURN NEW;
END;
$$;
```

↓

```sql
create or replace function touch_updated_at()
  returns trigger
  language plpgsql
as $$
  begin
    new.updated_at = now();
    raise notice 'Updated %', new.id;
    return new;
  end;
$$;
```

A `plpgsql` body with embedded statements, including one after `then`, one after `return query`, `select ... into`, and `into strict`, which is left as written:

```sql
CREATE FUNCTION close_order(order_id bigint) RETURNS integer LANGUAGE plpgsql AS $$
DECLARE
  n integer;
BEGIN
  SELECT count(*) INTO n FROM order_items i WHERE i.order_id = close_order.order_id AND i.status <> 'void';
  UPDATE orders SET status = 'closed', closed_at = now() WHERE id = order_id;
  IF EXISTS (SELECT 1 FROM refunds r WHERE r.order_id = order_id) THEN DELETE FROM refunds WHERE refunds.order_id = close_order.order_id; END IF;
  RETURN QUERY SELECT id FROM orders WHERE id = order_id;
  SELECT total INTO STRICT n FROM orders WHERE id = order_id;
  RETURN n;
END;
$$;
```

↓

```sql
create function close_order(order_id bigint)
  returns integer
  language plpgsql
as $$
  declare
    n integer;
  begin
    select count(*)
    into n
    from order_items as i
    where
      i.order_id = close_order.order_id
      and i.status <> 'void';
    update orders
    set
      status = 'closed'
      , closed_at = now()
    where id = order_id;
    if exists (select 1 from refunds r where r.order_id = order_id) then
      delete from refunds
      where refunds.order_id = close_order.order_id;
    end if;
    return query
      select id
      from orders
      where id = order_id;
    select total into strict n from orders where id = order_id;
    return n;
  end;
$$;
```

A `sql` function with four parameters:

```sql
create function find_events(user_id bigint, kind text, since timestamptz, max_rows int default 100) returns setof events language sql as $$ select * from events where events.user_id = find_events.user_id and events.created_at > since limit max_rows $$;
```

↓

```sql
create function find_events(
  user_id bigint
  , kind text
  , since timestamptz
  , max_rows integer default 100
)
  returns setof events
  language sql
as $$
  select *
  from events
  where
    events.user_id = find_events.user_id
    and events.created_at > since
  limit max_rows
$$;
```

A `sql` body that formats to one line stays on the `as` line:

```sql
create function one() returns int language sql as $$ SELECT 1 $$;
```

↓

```sql
create function one()
  returns integer
  language sql
as $$ select 1 $$;
```

Options written after an already indented body:

```sql
CREATE FUNCTION add_one(a int) RETURNS int AS $$
    BEGIN
        RETURN a + 1;
    END;
$$ LANGUAGE plpgsql STRICT;
```

↓

```sql
create function add_one(a integer)
  returns integer
  language plpgsql
  strict
as $$
  begin
      return a + 1;
  end;
$$;
```

A body in another language is left as written:

```sql
create function py_one() returns int language plpython3u as $$
Return 1
$$;
```

↓

```sql
create function py_one()
  returns integer
  language plpython3u
as $$
Return 1
$$;
```

---

### statement-separation

**Statements are separated by exactly one blank line.** Statements on the same line are split apart, and runs of blank lines are reduced to one. Test case: `20-multiple-statements`.

```sql
select 1;select 2;


select 3;
```

↓

```sql
select 1;

select 2;

select 3;
```

---

### line-length

**There is no line length limit.** Lines break only where the rules above place a break, so the same query always formats the same way regardless of editor width.

---

## Normalization

The PostgreSQL parser does not record certain spelling choices, so pgfmt prints these constructs in one canonical form. Each output parses to the same syntax tree as its input.

### aliases

🔒 **Table and column aliases always use `as`.** Test case: `01-select-basic`.

```sql
select u.id user_id from users u;
```

↓

```sql
select u.id as user_id
from users as u;
```

---

### casts

🔒 **`cast(x as t)` is printed as `x::t`. A string literal cast to `interval` is printed as `interval '...'`.** Test case: `11-functions-and-casts`.

```sql
select CAST(created_at AS DATE), '1 day'::interval;
```

↓

```sql
select
  created_at::date
  , interval '1 day';
```

---

### operators

🔒 **Binary operators have exactly one space on each side, however they were written. `!=` is printed as `<>`.** Unary minus has no space. Test case: `22-operators-and-literals`.

No spaces in the input:

```sql
select a+b, a-b*c, a||b, x!=y, x>=y, -a from t where a=b and c<>d;
```

↓

```sql
select
  a + b
  , a - b * c
  , a || b
  , x <> y
  , x >= y
  , -a
from t
where
  a = b
  and c <> d;
```

Extra spaces in the input:

```sql
select a  +   b from t where a   =  b;
```

↓

```sql
select a + b
from t
where a = b;
```

---

### join-keywords

🔒 **`inner join` is printed as `join`, and `left outer join` as `left join`.** A join with no condition is printed as `cross join`. Test case: `03-joins`.

```sql
select 1 from a inner join b on b.id = a.id left outer join c on c.id = a.id;
```

↓

```sql
select 1
from a
join b
  on b.id = a.id
left join c
  on c.id = a.id;
```

---

### type-names

🔒 **Built-in types use one spelling:** `integer`, `bigint`, `smallint`, `boolean`, `real`, `double precision`, `numeric`, `varchar`, `char`, `timestamptz`, `timetz`. Test case: `18-type-names`.

```sql
create table t (a INT4, b BOOL, c TIMESTAMP WITH TIME ZONE, d FLOAT8, e DECIMAL(10,2));
```

↓

```sql
create table t (
  a integer
  , b boolean
  , c timestamptz
  , d double precision
  , e numeric(10, 2)
);
```

---

### parentheses

🔒 **Parentheses are printed only where operator precedence requires them, with one exception: nested `and`/`or` groups are always parenthesized for readability.** The parser discards parentheses, so pgfmt reconstructs them from precedence.

```sql
select (a + b) * c, a - (b - c), (a * b) + c, a or b and c;
```

↓

```sql
select
  (a + b) * c
  , a - (b - c)
  , a * b + c
  , a or (b and c);
```

---

### quoting

🔒 **Identifiers are quoted only when required:** when they contain uppercase or special characters, or when they are reserved words in that position. After `as`, any keyword is allowed without quotes.

```sql
select "user".id, 1 as "order" from "user";
```

↓

```sql
select
  "user".id
  , 1 as order
from "user";
```

---

## Preservation

### comments

**Comments are kept next to the code they were written beside.** A comment on its own line stays on its own line before the same item. A comment at the end of a line stays at the end of that item's line. Test case: `10-comments`.

```sql
-- leading comment for the query
select id, -- primary key
  email /* contact */
from users
-- only active users
where active = true; -- trailing
```

↓

```sql
-- leading comment for the query
select
  id -- primary key
  , email /* contact */
from users
-- only active users
where active = true; -- trailing
```

---

### placeholders

**`:name` placeholders are kept, including kebab-case names.** A colon inside array brackets, a `::` cast, a string, or a comment is not a placeholder. Subtraction from a placeholder needs spaces: `:limit - 1`. Test case: `09-placeholders`.

```sql
select :user-id, :address-2 from users where id = :id::bigint and tags[1:2] = :tags and note <> 'time: 10:30';
```

↓

```sql
select
  :user-id
  , :address-2
from users
where
  id = :id::bigint
  and tags[1:2] = :tags
  and note <> 'time: 10:30';
```

---

### literals

**String and number literals are printed exactly as written.** This includes `E'...'` strings, dollar-quoted strings, and multi-line strings, whose contents are never re-indented.

```sql
SELECT E'a\nb', $tag$raw$tag$, 1.50;
```

↓

```sql
select
  E'a\nb'
  , $tag$raw$tag$
  , 1.50;
```

---

### semicolons

**A statement's semicolon is kept if present and not added if absent.** Some SQL loaders expect statements without a trailing semicolon.

```sql
SELECT id FROM users
```

↓

```sql
select id
from users
```

---

### unsupported-statements

**A statement that uses syntax pgfmt does not lay out is cleaned up but keeps its structure, and pgfmt reports it on stderr.** Keywords and unquoted identifiers are lowercased, runs of spaces become one space, spaces before commas and semicolons are removed, and line breaks are kept. Strings, comments, and placeholders are kept as written. If the cleaned text would parse differently, the statement is left exactly as written. Other statements in the file are formatted normally.

```sql
SELECT a FROM t;
VACUUM   ANALYZE   Users , Orders ;
```

↓

```sql
select a
from t;

vacuum analyze users, orders;
```
