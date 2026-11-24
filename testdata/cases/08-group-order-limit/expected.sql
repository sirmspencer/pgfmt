select user_id, status, count(*)
from orders
group by user_id, status
having count(*) > 5
order by user_id asc, count(*) desc nulls last
limit 10
offset :offset;

select distinct status
from orders
order by status;
