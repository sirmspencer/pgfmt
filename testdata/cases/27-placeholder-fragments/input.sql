SELECT account_id,
       created_at::date AS day,
       COUNT(*) AS orders,
       COUNT(*) FILTER (WHERE status = 'void') AS voided
FROM shop.orders
WHERE region = :region
:filters
GROUP BY account_id, created_at::date
ORDER BY day ASC, account_id;
select * from users where active :and-filters order by id;
select id, name :extra-columns from users;
select * from :table-name where id = :id;
select * from orders :where-clause;
select * from orders order by :sort-column limit :limit;
