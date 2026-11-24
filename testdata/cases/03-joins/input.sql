select u.id, o.id as order_id, p.name, a.label
from users u
JOIN orders o ON o.user_id = u.id
left outer join payments p on p.order_id = o.id and p.status = 'settled'
inner join accounts a using (account_id)
cross join regions r
where r.code = :region;
