WITH active AS (SELECT id FROM users WHERE active), totals as (select user_id, sum(total) total from orders group by user_id)
select a.id, t.total from active a join totals t on t.user_id = a.id;
