with active as (
  select id
  from users
  where active
),
totals as (
  select user_id, sum(total) as total
  from orders
  group by user_id
)
select a.id, t.total
from active as a
inner join totals as t
  on t.user_id = a.id;
