select id
from users
where id in (
    select user_id
    from orders
    where total > 100
);

select id
from users as u
where exists (
    select 1
    from orders as o
    where o.user_id = u.id
);

select
    t.user_id,
    t.total
from (
    select
        user_id,
        sum(total) as total
    from orders
    group by user_id
) as t
where t.total > 1000;

select
    id,
    (
        select count(*)
        from orders as o
        where o.user_id = u.id
    ) as order_count
from users as u;

select id
from users
where id in (1, 2, 3);
