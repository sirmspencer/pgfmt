select
  account_id,
  created_at::date                        as day,
  count(*)                                as orders,
  count(*) filter (where status = 'void') as voided
from shop.orders
where region = :region
:filters
group by account_id, created_at::date
order by day asc, account_id;

select *
from users
where active :and-filters
order by id;

select id, name :extra-columns
from users;

select *
from :table-name
where id = :id;

select *
from orders :where-clause;

select *
from orders
order by :sort-column
limit :limit;
