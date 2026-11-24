select id from users where id in (select user_id from orders where total > 100);
select id from users u where exists (select 1 from orders o where o.user_id = u.id);
select t.user_id, t.total from (select user_id, sum(total) as total from orders group by user_id) t where t.total > 1000;
select id, (select count(*) from orders o where o.user_id = u.id) as order_count from users u;
select id from users where id in (1, 2, 3);
