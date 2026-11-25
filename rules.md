*Rules document generated with LLM assistance*

# pgfmt Formatting Rules

This document lists the formatting rules pgfmt applies with its default settings; [settings](#settings) describes the ones a `.pgfmt.json` file can change. Each rule shows input and the output pgfmt produces, and names the test case in `testdata/cases` that enforces it. Every example in this file is checked against the formatter.

## Comparison with other formatters

Each rule below was compared with two other formatters, using the examples in this document:

- **IntelliJ:** the JetBrains SQL formatter, which IntelliJ IDEA Ultimate and DataGrip share, configured as close to these rules as its code style settings allow. The comparison used DataGrip 2025.1 through its command-line formatter with the code style below, which can be imported under Settings, Editor, Code Style, SQL.

<details>
<summary>IntelliJ code style</summary>

```xml
<code_scheme name="pgfmt" version="173">
  <SqlCodeStyleSettings version="6">
    <option name="KEYWORD_CASE" value="1" />
    <option name="TYPE_CASE" value="1" />
    <option name="CUSTOM_TYPE_CASE" value="1" />
    <option name="BUILT_IN_CASE" value="1" />
    <option name="IDENTIFIER_CASE" value="1" />
    <option name="ALIAS_CASE" value="1" />
    <option name="QUOTED_IDENTIFIER_CASE" value="0" />
    <option name="SELECT_USE_AS_WORD" value="1" />
    <option name="SELECT_EL_LINE" value="1" />
    <option name="SELECT_EL_WRAP" value="2" />
    <option name="SELECT_EL_COMMA" value="2" />
    <option name="SELECT_KEEP_N_ITEMS_IN_LINE" value="3" />
    <option name="SELECT_ALIGN_AS" value="true" />
    <option name="ALIGN_AS_IN_SELECT_STATEMENT" value="true" />
    <option name="FROM_EL_LINE" value="1" />
    <option name="FROM_EL_WRAP" value="1" />
    <option name="FROM_EL_COMMA" value="2" />
    <option name="FROM_INDENT_JOIN" value="false" />
    <option name="INDENT_JOIN" value="false" />
    <option name="FROM_ALIGN_JOIN_TABLES" value="false" />
    <option name="FROM_ALIGN_ALIASES" value="false" />
    <option name="FROM_PLACE_ON" value="11" />
    <option name="FROM_WRAP_ON" value="true" />
    <option name="INDENT_JOIN_CONDITION" value="true" />
    <option name="NEW_LINE_BEFORE_JOIN_CONDITION" value="true" />
    <option name="WHERE_EL_LINE" value="1" />
    <option name="WHERE_EL_WRAP" value="1" />
    <option name="ORDER_EL_LINE" value="1" />
    <option name="ORDER_EL_WRAP" value="1" />
    <option name="ORDER_EL_COMMA" value="2" />
    <option name="SET_EL_LINE" value="1" />
    <option name="SET_EL_WRAP" value="1" />
    <option name="SET_EL_COMMA" value="2" />
    <option name="SET_ALIGN_EQUAL_SIGN" value="true" />
    <option name="ALIGN_EQ_INSIDE_SET_CLAUSE" value="true" />
    <option name="WITH_EL_LINE" value="1" />
    <option name="WITH_EL_WRAP" value="1" />
    <option name="WITH_EL_COMMA" value="2" />
    <option name="SUBQUERY_OPENING" value="1" />
    <option name="SUBQUERY_CONTENT" value="2" />
    <option name="SUBQUERY_CLOSING" value="2" />
    <option name="TABLE_OPENING" value="1" />
    <option name="TABLE_CONTENT" value="2" />
    <option name="TABLE_CLOSING" value="2" />
    <option name="TABLE_TYPES_ALIGN" value="true" />
    <option name="INSERT_INTO_NL" value="2" />
    <option name="INSERT_EL_COMMA" value="2" />
    <option name="EXPR_SPACE_AROUND_OPERATOR" value="1" />
    <option name="SPACES_AROUND_OPERATORS" value="true" />
    <option name="EXPR_CASE_WHEN_WRAP" value="true" />
    <option name="EXPR_CASE_WHEN_INDENT" value="true" />
    <option name="EXPR_CASE_END" value="1" />
    <option name="QUERY_IN_ONE_STRING" value="1" />
    <option name="SCRIPT_STMT_FOLLOWS_BLANK_LINE" value="1" />
    <option name="TABLE_ALTER_INSTRUCTION_INDENT" value="true" />
    <option name="TABLE_ALTER_ITEM_INDENT" value="true" />
    <option name="FROM_ONLY_JOIN_INDENT" value="2" />
    <option name="WITH_ALIGN_AS" value="false" />
    <option name="WHERE_EL_BOUND" value="1" />
  </SqlCodeStyleSettings>
  <codeStyleSettings language="SQL">
    <indentOptions>
      <option name="INDENT_SIZE" value="2" />
      <option name="CONTINUATION_INDENT_SIZE" value="2" />
      <option name="TAB_SIZE" value="2" />
    </indentOptions>
  </codeStyleSettings>
</code_scheme>
```

</details>

- **sql-formatter:** the npm package `sql-formatter` 15.10.0, configured as close to these rules as its options allow:

```json
{
  "language": "postgresql",
  "tabWidth": 2,
  "useTabs": false,
  "keywordCase": "lower",
  "identifierCase": "lower",
  "dataTypeCase": "lower",
  "functionCase": "lower",
  "indentStyle": "standard",
  "logicalOperatorNewline": "before",
  "expressionWidth": 50,
  "linesBetweenQueries": 1,
  "denseOperators": false,
  "newlineBeforeSemicolon": false,
  "paramTypes": {
    "custom": [
      {
        "regex": ":[A-Za-z_][A-Za-z0-9_?!]*(?:-[A-Za-z0-9_?!]+)*"
      }
    ]
  }
}
```

IntelliJ: 9 same · 26 different · 7 not handled

sql-formatter: 12 same · 22 different · 8 not handled

| Rule | IntelliJ | sql-formatter |
|---|---|---|
| [keyword-case](#keyword-case) | Same | Same |
| [identifier-case](#identifier-case) | Same | Same |
| [function-and-type-case](#function-and-type-case) | Same | Same |
| [literal-case](#literal-case) | Same | Same |
| [indentation](#indentation) | Different | Same |
| [commas](#commas) | Same | Same |
| [column-lists](#column-lists) | Different | Different |
| [clause-items](#clause-items) | Different | Different |
| [conditions](#conditions) | Different | Different |
| [grouped-conditions](#grouped-conditions) | Different | Same |
| [joins](#joins) | Same | Different |
| [join-conditions](#join-conditions) | Different | Different |
| [subqueries](#subqueries) | Same | Same |
| [lists](#lists) | Different | Different |
| [ctes](#ctes) | Different | Different |
| [case-expressions](#case-expressions) | Different | Same |
| [set-operations](#set-operations) | Different | Different |
| [limit-and-offset](#limit-and-offset) | Different | Different |
| [insert](#insert) | Different | Different |
| [update](#update) | Different | Different |
| [delete](#delete) | Different | Different |
| [upsert](#upsert) | Different | Different |
| [create-table](#create-table) | Different | Different |
| [alter-table](#alter-table) | Different | Different |
| [create-index](#create-index) | Not handled | Different |
| [function-definitions](#function-definitions) | Different | Different |
| [statement-separation](#statement-separation) | Different | Same |
| [line-length](#line-length) | Different | Different |
| [alignment](#alignment) | Different | Not handled |
| [aliases](#aliases) | Different | Not handled |
| [casts](#casts) | Not handled | Not handled |
| [operators](#operators) | Different | Different |
| [join-keywords](#join-keywords) | Not handled | Not handled |
| [type-names](#type-names) | Not handled | Not handled |
| [parentheses](#parentheses) | Not handled | Not handled |
| [quoting](#quoting) | Not handled | Not handled |
| [stray-commas](#stray-commas) | Not handled | Not handled |
| [comments](#comments) | Different | Different |
| [placeholders](#placeholders) | Different | Different |
| [literals](#literals) | Same | Same |
| [semicolons](#semicolons) | Same | Same |
| [unsupported-statements](#unsupported-statements) | Different | Different |

## Index

**Case**

- [keyword-case](#keyword-case)
- [identifier-case](#identifier-case)
- [function-and-type-case](#function-and-type-case)
- [literal-case](#literal-case)

**Layout**

- [indentation](#indentation)
- [commas](#commas)
- [column-lists](#column-lists)
- [clause-items](#clause-items)
- [conditions](#conditions)
- [grouped-conditions](#grouped-conditions)
- [joins](#joins)
- [join-conditions](#join-conditions)
- [subqueries](#subqueries)
- [lists](#lists)
- [ctes](#ctes)
- [case-expressions](#case-expressions)
- [set-operations](#set-operations)
- [limit-and-offset](#limit-and-offset)
- [insert](#insert)
- [update](#update)
- [delete](#delete)
- [upsert](#upsert)
- [create-table](#create-table)
- [alter-table](#alter-table)
- [create-index](#create-index)
- [function-definitions](#function-definitions)
- [statement-separation](#statement-separation)
- [line-length](#line-length)
- [alignment](#alignment)

**Normalization**

- [aliases](#aliases)
- [casts](#casts)
- [operators](#operators)
- [join-keywords](#join-keywords)
- [type-names](#type-names)
- [parentheses](#parentheses)
- [quoting](#quoting)
- [comments](#comments)

**Repairs**

- [stray-commas](#stray-commas)

**Preservation**

- [placeholders](#placeholders)
- [literals](#literals)
- [semicolons](#semicolons)
- [unsupported-statements](#unsupported-statements)

---

## Case

### keyword-case

**Keywords are lowercase.** Test case: `01-select-basic`.

**IntelliJ:** Same: with keyword case set to lower.

**sql-formatter:** Same: with `keywordCase: lower`.

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

**IntelliJ:** Same: with identifier case set to lower; quoted identifiers are left alone.

**sql-formatter:** Same: with `identifierCase: lower`, which leaves quoted identifiers alone.

```sql
SELECT UserId, "DisplayName" FROM Public.Users WHERE "Status" = 'Active';
```

↓

```sql
select userid, "DisplayName"
from public.users
where "Status" = 'Active';
```

---

### function-and-type-case

**Function names and type names are lowercase.** Test case: `11-functions-and-casts`.

**IntelliJ:** Same: with built-in and type case set to lower.

**sql-formatter:** Same: with `functionCase: lower` and `dataTypeCase: lower`.

```sql
SELECT COALESCE(nickname, first_name), NOW(), amount::NUMERIC(10,2) FROM orders;
```

↓

```sql
select coalesce(nickname, first_name), now(), amount::numeric(10, 2)
from orders;
```

---

### literal-case

**`true`, `false`, and `null` are lowercase.** Test case: `22-operators-and-literals`.

**IntelliJ:** Same: with keyword case set to lower.

**sql-formatter:** Same: with `keywordCase: lower`.

```sql
select NULL, TRUE, FALSE;
```

↓

```sql
select null, true, false;
```

---

## Layout

### indentation

**Each level is indented by two spaces.** Tabs are not used. Every rule below that indents a line uses this unit.

**IntelliJ:** Different: the indent is two spaces, but continuation lines of several lists are aligned under the first item instead of indented.

**sql-formatter:** Same: with `tabWidth: 2`.

---

### commas

**When a list is split across lines, each item except the last ends with a comma.** This applies to every multi-line list: select items, `group by`, `order by`, `set`, `returning`, CTEs, long parenthesized lists, table columns, and `alter table` actions. Lists that stay on one line use ordinary `, ` separators. Test case: `01-select-basic`.

**IntelliJ:** Same: commas trail.

**sql-formatter:** Same: commas always trail.

```sql
select id, email, created_at, updated_at from users;
```

↓

```sql
select
  id,
  email,
  created_at,
  updated_at
from users;
```

---

### column-lists

**`select`, `group by`, `order by`, and `returning` keep three or fewer items on the keyword's line. With four or more items, the keyword is alone on its line and each item is on its own indented line.** This is the same count as the [lists](#lists) rule. A shorter list is also split one item per line when an item spans several lines, such as a `case` expression or a subquery. Test cases: `01-select-basic`, `07-case-expressions`, `08-group-order-limit`.

**IntelliJ:** Different: a select list stays on one line until it passes the right margin, whatever its length, and `group by` and `order by` items after the first are aligned under it.

**sql-formatter:** Different: the keyword is always alone on its line and every item is on its own line below it, even a single item. No setting keeps short lists on the keyword's line.

Three or fewer items in each clause:

```sql
select user_id, status, count(*) from orders group by user_id, status order by user_id, count(*) desc;
```

↓

```sql
select user_id, status, count(*)
from orders
group by user_id, status
order by user_id, count(*) desc;
```

Four or more items in each clause:

```sql
select region, account_id, status, day, count(*) from orders group by region, account_id, status, day order by region, account_id, status, day;
```

↓

```sql
select
  region,
  account_id,
  status,
  day,
  count(*)
from orders
group by
  region,
  account_id,
  status,
  day
order by
  region,
  account_id,
  status,
  day;
```

`returning` with three and four items:

```sql
insert into users (id, email) values (:id, :email) returning id, email, created_at;
```

↓

```sql
insert into users (id, email)
values (:id, :email)
returning id, email, created_at;
```

```sql
update users set email = :email where id = :id returning id, email, updated_at, version;
```

↓

```sql
update users
set email = :email
where id = :id
returning
  id,
  email,
  updated_at,
  version;
```

A multi-line item:

```sql
select id, case when status = 'a' then 'active' else 'other' end as label from users;
```

↓

```sql
select
  id,
  case
    when status = 'a' then 'active'
    else 'other'
  end as label
from users;
```

`distinct` and `distinct on (...)` stay with `select`:

```sql
select distinct status, region from orders;
```

↓

```sql
select distinct status, region
from orders;
```

---

### clause-items

**`set`, `from`, `using`, and `values` keep one item on the keyword's line. With two or more items, the keyword is alone on its line and each item is on its own indented line.** Column lists follow the [column-lists](#column-lists) rule instead. Test cases: `02-single-item-clauses`, `13-insert`, `14-update`.

**IntelliJ:** Different: the first item stays on the keyword's line, and later items are aligned under it.

**sql-formatter:** Different: the keyword is always alone on its line, with even a single item indented below it.

One item in each clause:

```sql
update users set email = :email from accounts where accounts.id = users.account_id;
```

↓

```sql
update users
set email = :email
from accounts
where accounts.id = users.account_id;
```

Two or more items in each clause:

```sql
update users set email = :email, updated_at = now() from accounts, regions where accounts.id = users.account_id;
```

↓

```sql
update users
set
  email      = :email,
  updated_at = now()
from
  accounts,
  regions
where accounts.id = users.account_id;
```

`values` with one and two rows:

```sql
insert into tags (name) values ('a');
```

↓

```sql
insert into tags (name)
values ('a');
```

```sql
insert into tags (name) values ('a'), ('b');
```

↓

```sql
insert into tags (name)
values
  ('a'),
  ('b');
```

---

### conditions

**`where` and `having` keep a single condition on the keyword's line. Two or more conditions place the keyword alone on its line and each condition on its own indented line, with `and` or `or` leading each condition after the first.** Conditions do not use the three-item rule of [column-lists](#column-lists): two conditions already split. Test cases: `04-where-boolean`, `24-conditions`.

**IntelliJ:** Different: a single condition behaves the same. With several, the first stays on the keyword's line and the rest follow on indented lines led by `and` or `or`. Placing elements on a new line splits a single condition as well.

**sql-formatter:** Different: several conditions are laid out the same way, but a single condition also moves below the keyword.

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

**IntelliJ:** Different: a parenthesized group stays on one line.

**sql-formatter:** Same: groups open a parenthesis at the end of the line and close it on its own line.

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

**IntelliJ:** Same: joins start at the level of `from` with the join indentation set to unindent.

**sql-formatter:** Different: joins are indented one level under `from`.

```sql
select u.id from users u join orders o on o.user_id = u.id cross join regions r;
```

↓

```sql
select u.id
from users as u
inner join orders as o
  on o.user_id = u.id
cross join regions as r;
```

---

### join-conditions

**`on` and `using` are on the line below the join, indented one level. Additional conditions align with `on`.** Test case: `03-joins`.

**IntelliJ:** Different: `on` and `using` are on an indented line below the join, but additional conditions stay on the `on` line.

**sql-formatter:** Different: `on` and `using` stay on the join's line, and additional conditions start a new line at the join's indentation.

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
inner join accounts as a
  using (account_id);
```

---

### subqueries

**A subquery opens a parenthesis at the end of the line, indents its body, and closes the parenthesis on its own line, aligned with the start of the line that opened it.** This applies to `in`, `exists`, scalar subqueries, and subqueries in `from`. Test case: `05-subqueries`.

**IntelliJ:** Same: with the subquery content set to wrap in the Egyptian style, the body is indented and the closing parenthesis has its own line.

**sql-formatter:** Same: subqueries open a parenthesis at the end of the line and close it on its own line.

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
  select user_id, sum(total) as total
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
  id,
  (
    select count(*)
    from orders as o
    where o.user_id = u.id
  ) as order_count
from users as u;
```

---

### lists

**A parenthesized list with three or fewer items stays on one line. A list with four or more items keeps the opening parenthesis on its line, places each item on its own indented line with trailing commas, and puts the closing parenthesis on its own line.** This covers function arguments and parameters, `in (...)` value lists, `coalesce`, `greatest`, `least`, `array[...]`, row values, `insert` column lists, `values` rows, index columns and `include` lists, constraint key lists, join `using` lists, `on conflict` targets, alias and CTE column names, `distinct on`, and the names in a `drop` statement. A shorter list is also split one item per line when an item spans several lines, such as a subquery argument. Aggregate calls with `distinct` or `order by` inside the parentheses stay on one line, as do window specifications in `over (...)` and type modifiers such as `numeric(10, 2)`. Test case: `25-long-lists`.

**IntelliJ:** Different: parenthesized lists are split only when they pass the right margin, not by item count, and a space is added after `array`.

**sql-formatter:** Different: parenthesized lists break by width, set with `expressionWidth`, not by item count, so a short list of four or more items stays on one line.

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
  1,
  2,
  3,
  4
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
  :year,
  :month,
  :day,
  0,
  0,
  0
);
```

The count is per argument, so a function that takes key and value pairs breaks after two pairs:

```sql
select jsonb_build_object('id', u.id, 'name', u.name) from users u;
```

↓

```sql
select jsonb_build_object(
  'id',
  u.id,
  'name',
  u.name
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
  'id',
  u.id,
  'tags',
  array[
    u.a,
    u.b,
    u.c,
    u.d
  ]
)
from users as u;
```

Index columns and constraint keys:

```sql
create index events_lookup_idx on events (account_id, kind, status, created_at);
```

↓

```sql
create index events_lookup_idx on events (
  account_id,
  kind,
  status,
  created_at
);
```

```sql
alter table events add constraint events_account_fk foreign key (account_id, region, shard, kind) references accounts (id, region, shard, kind);
```

↓

```sql
alter table events add constraint events_account_fk foreign key (
  account_id,
  region,
  shard,
  kind
) references accounts (
  id,
  region,
  shard,
  kind
);
```

`coalesce` with three and four arguments:

```sql
select coalesce(a, b, c), coalesce(a, b, c, d) from t;
```

↓

```sql
select
  coalesce(a, b, c),
  coalesce(
    a,
    b,
    c,
    d
  )
from t;
```

---

### ctes

**Each common table expression opens with `name as (`, indents its body, and closes with `)` at the start of a line. A comma follows each CTE's closing parenthesis, and the next CTE starts on the following line.** Test case: `06-ctes`.

**IntelliJ:** Different: the CTE body layout matches, but later CTEs are aligned under the first.

**sql-formatter:** Different: `with` is alone on its line and every CTE is indented under it.

```sql
with active as (select id from users where active), totals as (select user_id from orders) select a.id from active a;
```

↓

```sql
with active as (
  select id
  from users
  where active
),
totals as (
  select user_id
  from orders
)
select a.id
from active as a;
```

---

### case-expressions

**A `case` expression places each `when` and the `else` on its own indented line and closes with `end`.** Test case: `07-case-expressions`.

**IntelliJ:** Different: a `case` expression stays on one line.

**sql-formatter:** Same: `when` and `else` are indented one per line and `end` closes the expression.

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
  id,
  case role
    when 'admin' then 1
    else 0
  end as is_admin
from users;
```

---

### set-operations

**`union`, `intersect`, and `except` are alone on their lines, with a blank line before and after.** Test case: `12-set-operations`.

**IntelliJ:** Different: `union` is alone on its line, but without blank lines before and after.

**sql-formatter:** Different: `union` is alone on its line, but without blank lines before and after.

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

**IntelliJ:** Different: `limit` and `offset` share one line.

**sql-formatter:** Different: `limit` and `offset` are alone on their lines, with the value indented below.

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

**IntelliJ:** Different: the column list and rows stay on one line until they pass the right margin, several rows are aligned under the first, and no space is added between a table name and its column list.

**sql-formatter:** Different: `insert into` and `values` are alone on their lines, with the table and each row indented below.

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
  ('a'),
  ('b');
```

Four or more columns:

```sql
insert into events (id, user_id, kind, created_at) values (:id, :user_id, :kind, now());
```

↓

```sql
insert into events (
  id,
  user_id,
  kind,
  created_at
)
values (
  :id,
  :user_id,
  :kind,
  now()
);
```

Four or more columns and several rows:

```sql
insert into events (id, user_id, kind, created_at) values (:id, :user_id, :kind, now()), (:id2, :user_id2, :kind2, now());
```

↓

```sql
insert into events (
  id,
  user_id,
  kind,
  created_at
)
values
  (
    :id,
    :user_id,
    :kind,
    now()
  ),
  (
    :id2,
    :user_id2,
    :kind2,
    now()
  );
```

From a select:

```sql
insert into archived_users (id, email) select id, email from users where deleted_at is not null;
```

↓

```sql
insert into archived_users (id, email)
select id, email
from users
where deleted_at is not null;
```

---

### update

**`set` follows the clause-items rule: one assignment stays on the `set` line, and several assignments are indented one per line and aligned on `=`.** Test case: `14-update`.

**IntelliJ:** Different: assignments are aligned on `=`, but the first stays on the `set` line and `=` is not spaced.

**sql-formatter:** Different: assignments are not aligned on `=`.

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
  email      = :email,
  updated_at = now()
where id = :id;
```

---

### delete

**`using`, `where`, and `returning` each start a line.** Test case: `15-delete`.

**IntelliJ:** Different: `delete` is alone on its line, and `from` and `using` share the next line.

**sql-formatter:** Different: `using` stays on the `delete from` line.

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

**IntelliJ:** Different: assignments are aligned on `=`, but stay on the `on conflict` line.

**sql-formatter:** Different: the layout matches, but assignments are not aligned on `=`.

```sql
insert into user_settings (user_id, key, value) values (:user_id, :key, :value) on conflict (user_id, key) do update set value = excluded.value, updated_at = now();
```

↓

```sql
insert into user_settings (user_id, key, value)
values (:user_id, :key, :value)
on conflict (user_id, key) do update
set
  value      = excluded.value,
  updated_at = now();
```

---

### create-table

**The opening parenthesis ends the first line. Each column and table constraint is on its own indented line with trailing commas, and columns are aligned as described in [alignment](#alignment).** Test case: `17-create-table`.

**IntelliJ:** Different: the layout matches with the table content set to the Egyptian style, but types are not padded so that constraints line up.

**sql-formatter:** Different: the layout matches, but columns are not aligned into a grid.

```sql
CREATE TABLE IF NOT EXISTS users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL UNIQUE, CONSTRAINT users_email_check CHECK (email <> ''));
```

↓

```sql
create table if not exists users (
  id    bigserial primary key,
  email text      not null unique,
  constraint users_email_check check (email <> '')
);
```

---

### alter-table

**One action stays on the `alter table` line. Several actions are indented one per line with trailing commas.** Test case: `19-alter-and-index`.

**IntelliJ:** Different: several actions match, but a single action also moves to its own line.

**sql-formatter:** Different: actions start at the `alter table` indentation instead of one level in, a single action also moves to its own line, and `set not null` or `set default` is split onto a new line.

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
  add column last_login  timestamptz,
  add column login_count integer not null default 0;
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
  drop column legacy_flag,
  alter column email type varchar(320),
  alter column active set default true;
```

---

### create-index

**An index without a `where` clause stays on one line. A partial index places `on` and `where` on indented lines below the index name.** Test case: `19-alter-and-index`.

**IntelliJ:** Not handled: index statements are left on one line, including partial indexes, and `using btree` is kept.

**sql-formatter:** Different: index statements are kept on one line, including partial indexes, and `using btree` is kept.

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

**IntelliJ:** Different: `returns` stays on the signature line, the options and `as` share an indented line, and the body opens on its own line. SQL in `sql` and `plpgsql` bodies is formatted with the same code style. Parameters stay on one line and options after the body stay there.

**sql-formatter:** Different: the signature, `returns`, and options stay on one line, a space is added before the parameter list, and the body is left exactly as written.

The body is a string to the PostgreSQL parser, so pgfmt formats it separately, by language:

- **`sql`:** the body is formatted with every rule in this document. A body that formats to one line stays on the `as` line; a longer body is indented one level inside its dollar quotes.
- **`plpgsql`:** each standalone SQL statement is formatted with every rule in this document. A statement is standalone when it starts with `select`, `insert`, `update`, `delete`, or `with` after a statement boundary (`;`, `begin`, `then`, `else`, `loop`) or after `return query`, and ends at its semicolon. A standalone statement always starts on its own line; if it followed other code on the line, it moves to the next line one level deeper, and code after its semicolon, such as `end if;`, moves to its own line. The procedural code around the statements keeps its line structure, with keywords and unquoted identifiers lowercased. The body is indented one level inside its dollar quotes: the indentation its lines share is replaced by one level, and their own nesting is kept.
- **Other languages,** such as `plpython3u`, are left exactly as written.

Two kinds of SQL inside `plpgsql` are not formatted: SQL inside an expression, such as `if exists (select ...)` or `for r in select ... loop`, and statements that use plpgsql-only syntax, such as `select ... into strict`. Both are kept as written apart from case. Strings and quoted identifiers in any body are never changed, and comments in `sql` and `plpgsql` bodies are removed.

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
      status    = 'closed',
      closed_at = now()
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
  user_id bigint,
  kind text,
  since timestamptz,
  max_rows integer default 100
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

**IntelliJ:** Different: statements are split apart with a blank line between them, but existing runs of blank lines are kept.

**sql-formatter:** Same: with `linesBetweenQueries: 1`.

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

**IntelliJ:** Different: lines are wrapped at the right margin, so layout depends on length.

**sql-formatter:** Different: parenthesized expressions are split once they pass `expressionWidth`, so layout depends on length.

---

### alignment

**Four constructs are aligned into columns.** Elsewhere one space separates a name from what follows it.

- **`create table` columns:** column names, types, and constraints form a grid. Table constraints, such as `constraint ... check (...)`, are not part of the grid.
- **`alter table` with several `add column` actions:** the added columns form the same grid.
- **`set` in `update` and in `on conflict ... do update`:** assignments are aligned on `=`.
- **Select lists split across lines:** aliases are aligned on `as`. Items without an alias, and items that span several lines such as `case` expressions, are not padded.

A cell is padded to the widest cell in its column among the lines that have another cell after it. Test cases: `11-functions-and-casts`, `14-update`, `16-upsert`, `17-create-table`, `19-alter-and-index`.

**IntelliJ:** Different: `create table` names, added columns, and `set` assignments are aligned, but types are not padded, and select lists stay on one line, so aliases are not aligned.

**sql-formatter:** Not handled: nothing is aligned into columns.

Columns in `create table`:

```sql
create table rate_limits (id bigserial primary key, max_requests_per_minute integer not null, burst integer not null default 10, updated_at timestamptz not null default now(), constraint burst_positive check (burst > 0));
```

↓

```sql
create table rate_limits (
  id                      bigserial   primary key,
  max_requests_per_minute integer     not null,
  burst                   integer     not null default 10,
  updated_at              timestamptz not null default now(),
  constraint burst_positive check (burst > 0)
);
```

Added columns in `alter table`:

```sql
alter table rate_limits add column window_seconds integer not null default 60, add column note text;
```

↓

```sql
alter table rate_limits
  add column window_seconds integer not null default 60,
  add column note           text;
```

Assignments in an upsert, and in a plain update:

```sql
insert into rate_limits (id, burst) values (:id, :burst) on conflict (id) do update set max_requests_per_minute = :max-requests-per-minute, burst = excluded.burst;
```

↓

```sql
insert into rate_limits (id, burst)
values (:id, :burst)
on conflict (id) do update
set
  max_requests_per_minute = :max-requests-per-minute,
  burst                   = excluded.burst;
```

```sql
update rate_limits set max_requests_per_minute = :max-requests-per-minute, burst = :burst where id = :id;
```

↓

```sql
update rate_limits
set
  max_requests_per_minute = :max-requests-per-minute,
  burst                   = :burst
where id = :id;
```

Aliases in a select list:

```sql
select count(*) as calls, coalesce(sum(latency_ms), 0) as total_latency_ms, max(called_at) as last_call, region from calls group by region;
```

↓

```sql
select
  count(*)                     as calls,
  coalesce(sum(latency_ms), 0) as total_latency_ms,
  max(called_at)               as last_call,
  region
from calls
group by region;
```

---

## Normalization

Each of these constructs has several equivalent spellings, and pgfmt always prints the same one. Each output parses to the same syntax tree as its input.

### aliases

**Table and column aliases always use `as`.** Test case: `01-select-basic`.

**IntelliJ:** Different: with `as` set to add, column aliases get `as`, but table aliases do not.

**sql-formatter:** Not handled: aliases are left with or without `as`, as written.

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

**`cast(x as t)` is printed as `x::t`. A string literal cast to `interval` is printed as `interval '...'`.** Test case: `11-functions-and-casts`.

**IntelliJ:** Not handled: `cast(x as t)` and `::` are left as written.

**sql-formatter:** Not handled: `cast(x as t)` and `::` are left as written.

```sql
select CAST(created_at AS DATE), '1 day'::interval;
```

↓

```sql
select created_at::date, interval '1 day';
```

---

### operators

**Binary operators have exactly one space on each side, however they were written. `!=` is printed as `<>`.** Unary minus has no space. Test case: `22-operators-and-literals`.

**IntelliJ:** Different: spaces are added around binary operators, but `!=` is kept and `set` assignments are left unspaced.

**sql-formatter:** Different: spacing around binary operators matches, but `!=` is kept and unary minus gets a space, as in `- a`.

No spaces in the input:

```sql
select a+b, a-b*c, a||b, x!=y, x>=y, -a from t where a=b and c<>d;
```

↓

```sql
select
  a + b,
  a - b * c,
  a || b,
  x <> y,
  x >= y,
  -a
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

**`join` is printed as `inner join`, so every inner join says so, and `left outer join` is printed as `left join`.** A natural join is printed as `natural inner join`, and a join with no condition as `cross join`. Test case: `03-joins`.

**IntelliJ:** Not handled: `join`, `inner join`, and `left outer join` are left as written.

**sql-formatter:** Not handled: `join`, `inner join`, and `left outer join` are left as written.

```sql
select 1 from a join b on b.id = a.id left outer join c on c.id = a.id natural join d;
```

↓

```sql
select 1
from a
inner join b
  on b.id = a.id
left join c
  on c.id = a.id
natural inner join d;
```

---

### type-names

**Built-in types use one spelling:** `integer`, `bigint`, `smallint`, `boolean`, `real`, `double precision`, `numeric`, `varchar`, `char`, `timestamptz`, `timetz`. Test case: `18-type-names`.

**IntelliJ:** Not handled: type spellings such as `int4` and `timestamp with time zone` are left as written.

**sql-formatter:** Not handled: type names are only lowercased, so `int4` and `timestamp with time zone` stay as written.

```sql
create table t (a INT4, b BOOL, c TIMESTAMP WITH TIME ZONE, d FLOAT8, e DECIMAL(10,2));
```

↓

```sql
create table t (
  a integer,
  b boolean,
  c timestamptz,
  d double precision,
  e numeric(10, 2)
);
```

---

### parentheses

**Parentheses are printed only where operator precedence requires them, with one exception: nested `and`/`or` groups are always parenthesized for readability.** The parser discards parentheses, so pgfmt reconstructs them from precedence.

**IntelliJ:** Not handled: redundant parentheses are kept and none are added.

**sql-formatter:** Not handled: parentheses are left as written, and `and` and `or` in a select item are split onto separate lines.

```sql
select (a + b) * c, a - (b - c), (a * b) + c, a or b and c;
```

↓

```sql
select
  (a + b) * c,
  a - (b - c),
  a * b + c,
  a or (b and c);
```

---

### quoting

**Identifiers are quoted only when required:** when they contain uppercase or special characters, or when they are reserved words in that position. After `as`, any keyword is allowed without quotes.

**IntelliJ:** Not handled: quotes are left as written, including unnecessary ones.

**sql-formatter:** Not handled: quotes are left as written, including unnecessary ones.

```sql
select "user".id, 1 as "order" from "user";
```

↓

```sql
select "user".id, 1 as order
from "user";
```

---

### comments

**Comments are removed.** That includes `--` and `/* */` comments anywhere in a statement, between statements, and inside `sql` and `plpgsql` function bodies, where a line that held only a comment is dropped. Comments in bodies of other languages, such as `plpython3u`, are left with the body. Text inside strings that looks like a comment is kept, and so are `:name` placeholders. Line numbers in error messages still refer to the original file. Test case: `10-comments`.

**IntelliJ:** Different: comments are kept beside the code they were written next to.

**sql-formatter:** Different: comments are kept, and some move to a different line.

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
select id, email
from users
where active = true;
```

Text in a string is not a comment:

```sql
select '-- not a comment', '/* nor this */' from t;
```

↓

```sql
select '-- not a comment', '/* nor this */'
from t;
```

---

## Repairs

### stray-commas

**A comma that can only be extra is removed.** That is a trailing comma before a closing parenthesis or bracket, a semicolon, the end of the statement, or a reserved keyword such as `from`, `where`, or `order`; a doubled comma; or a comma directly after an opening parenthesis or a keyword. `pgfmt -check` reports each one as fixable, and `pgfmt -fix` removes it. A missing comma cannot be repaired, because the statement either fails to parse or means something else, so it is reported as a syntax error.

**IntelliJ:** Not handled: stray commas are left in place.

**sql-formatter:** Not handled: stray commas are left in place.

A trailing comma before `from`:

```sql
select a, b, from t;
```

↓

```sql
select a, b
from t;
```

A trailing comma in a list written with leading commas:

```sql
select
  a
  , b
  ,
from t;
```

↓

```sql
select a, b
from t;
```

Trailing commas inside parentheses:

```sql
insert into t (a, b,) values (1, 2,);
```

↓

```sql
insert into t (a, b)
values (1, 2);
```

A doubled comma and a leading comma:

```sql
select a,, b from t;
```

↓

```sql
select a, b
from t;
```

```sql
select , a from t;
```

↓

```sql
select a
from t;
```

---

## Preservation

### placeholders

**`:name` placeholders are kept exactly as written.** Names follow Clojure keywords as inquery uses them: snake_case such as `:example_key`, kebab-case such as `:user-id`, digits, mixed case, and `?` or `!` are all part of the name. A colon inside array brackets, a `::` cast, a string, or a comment is not a placeholder. Subtraction from a placeholder needs spaces: `:limit - 1`. Test cases: `09-placeholders`, `27-placeholder-fragments`.

**IntelliJ:** Different: plain `:name` placeholders are kept, but kebab-case placeholders are split into a subtraction, and a fragment after a name, as in `active :and-filters`, becomes `active : and -filters`, which no longer parses.

**sql-formatter:** Different: with `paramTypes` set to a custom pattern for inquery names, value and name placeholders are kept, including kebab-case. A fragment on its own line, such as `:filters`, is joined to the line before it. With the built-in `named` type instead, kebab-case names are split into a subtraction.

inquery replaces each placeholder with raw text, so a placeholder can stand for a value, a fragment of SQL, or a table or column name. pgfmt reads each placeholder in that order and uses the first reading that parses:

1. **A value,** such as `where id = :id` or `order by :sort-column`. This covers most placeholders.
2. **A fragment** when a value does not fit, such as `:filters` between a `where` condition and `group by`. A fragment is kept where it was written, like a comment: on its own line or at the end of a line. The rest of the statement is formatted as usual. A fragment name needs at least three characters.
3. **A name** when the statement does not parse without the placeholder, such as `from :table-name`.

A placeholder after an ordinary name, as in `where active :filters`, is read as a fragment rather than as a typed literal. A typed literal on a built-in type, such as `date :day`, remains a value.

Values:

```sql
select :user-id, :address-2, :example_key from users where id = :id::bigint and tags[1:2] = :tags and note <> 'time: 10:30';
```

↓

```sql
select :user-id, :address-2, :example_key
from users
where
  id = :id::bigint
  and tags[1:2] = :tags
  and note <> 'time: 10:30';
```

A fragment on its own line:

```sql
SELECT account_id, COUNT(*) AS orders FROM shop.orders WHERE region = :region
:filters
GROUP BY account_id ORDER BY account_id;
```

↓

```sql
select account_id, count(*) as orders
from shop.orders
where region = :region
:filters
group by account_id
order by account_id;
```

Fragments at the end of a line, and a name:

```sql
select id, name :extra-columns from :table-name where active :and-filters;
```

↓

```sql
select id, name :extra-columns
from :table-name
where active :and-filters;
```

---

### literals

**String and number literals are printed exactly as written.** This includes `E'...'` strings, dollar-quoted strings, and multi-line strings, whose contents are never re-indented.

**IntelliJ:** Same: literals are kept as written.

**sql-formatter:** Same: literals are kept as written.

```sql
SELECT E'a\nb', $tag$raw$tag$, 1.50;
```

↓

```sql
select E'a\nb', $tag$raw$tag$, 1.50;
```

---

### semicolons

**A statement's semicolon is kept if present and not added if absent.** Some SQL loaders expect statements without a trailing semicolon.

**IntelliJ:** Same: a missing semicolon is not added.

**sql-formatter:** Same: a missing semicolon is not added.

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

**IntelliJ:** Different: runs of spaces are collapsed, but the space before a comma is kept.

**sql-formatter:** Different: every statement is laid out by sql-formatter's token rules, so statements pgfmt cleans up, such as `vacuum`, can be split onto several lines.

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

---

## Settings

The rules above are the defaults. A `.pgfmt.json` file changes individual settings; settings it does not mention keep their defaults. pgfmt uses the nearest `.pgfmt.json` in the formatted file's directory or a parent directory, or, for stdin, in the current directory or a parent. `pgfmt -config file` uses a specific file instead. An unknown setting or an invalid value is an error.

The default settings:

```json
{
  "indent": 2,
  "clauseListLimit": 3,
  "parenListLimit": 3,
  "keepComments": false,
  "align": true,
  "blankLinesAroundSetOperators": true
}
```

### indent

**The number of spaces per indentation level,** from 1 to 8. It applies everywhere, including inside function bodies. Default `2`.

With `{"indent": 4}`:

```sql
select id, email, name, created_at from users where active = true and id = :id;
```

↓

```sql
select
    id,
    email,
    name,
    created_at
from users
where
    active = true
    and id = :id;
```

### clauseListLimit

**The most items `select`, `group by`, `order by`, and `returning` keep on the keyword's line.** `0` always splits, even a single item. It does not affect parenthesized lists. Default `3`.

With `{"clauseListLimit": 1}`:

```sql
select id, email from users order by id, email;
```

↓

```sql
select
  id,
  email
from users
order by
  id,
  email;
```

With `{"clauseListLimit": 0}`:

```sql
select id from users order by id;
```

↓

```sql
select
  id
from users
order by
  id;
```

With `{"clauseListLimit": 5}`:

```sql
select id, email, name, region, created_at from users;
```

↓

```sql
select id, email, name, region, created_at
from users;
```

### parenListLimit

**The most items a parenthesized list keeps on one line:** function arguments and parameters, `in (...)` lists, `insert` columns and rows, and the other lists in [lists](#lists). `0` always splits a list with at least one item; an empty list such as `now()` stays as it is. It does not affect `select` lists. Default `3`.

With `{"parenListLimit": 6}`:

```sql
select make_timestamp(:year, :month, :day, 0, 0, 0) where id in (1, 2, 3, 4);
```

↓

```sql
select make_timestamp(:year, :month, :day, 0, 0, 0)
where id in (1, 2, 3, 4);
```

With `{"parenListLimit": 1}`:

```sql
select coalesce(a, b) from t;
```

↓

```sql
select coalesce(
  a,
  b
)
from t;
```

With `{"parenListLimit": 0}`:

```sql
select lower(email), now() from users where id in (:id);
```

↓

```sql
select
  lower(
    email
  ),
  now()
from users
where id in (
  :id
);
```

### keepComments

**Keep comments in place instead of removing them.** Comments stay next to the code they were written beside, and a select list with a comment between items is split one item per line so the comment keeps its place. Default `false`.

With `{"keepComments": true}`:

```sql
-- leading comment for the query
select id, -- primary key
  email
from users; -- trailing
```

↓

```sql
-- leading comment for the query
select
  id, -- primary key
  email
from users; -- trailing
```

### align

**Align `create table` columns, `set` assignments, and select aliases into columns,** as described in [alignment](#alignment). With `false`, one space separates a name from what follows it everywhere. Default `true`.

With `{"align": false}`:

```sql
update rate_limits set max_requests_per_minute = :max-requests-per-minute, burst = :burst where id = :id;
```

↓

```sql
update rate_limits
set
  max_requests_per_minute = :max-requests-per-minute,
  burst = :burst
where id = :id;
```

With `{"align": false}`:

```sql
create table rate_limits (id bigserial primary key, max_requests_per_minute integer not null, burst integer not null default 10);
```

↓

```sql
create table rate_limits (
  id bigserial primary key,
  max_requests_per_minute integer not null,
  burst integer not null default 10
);
```

### blankLinesAroundSetOperators

**Put a blank line before and after `union`, `intersect`, and `except`.** Default `true`.

With `{"blankLinesAroundSetOperators": false}`:

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
