select
  u.id
  , o.id as order_id
  , p.name
  , a.label
from users as u
inner join orders as o
  on o.user_id = u.id
left join payments as p
  on p.order_id = o.id
  and p.status = 'settled'
inner join accounts as a
  using (account_id)
cross join regions as r
where r.code = :region;
