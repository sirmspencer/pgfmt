select id
from users
where
  active = true
  and (
    role = 'admin'
    or role = 'owner'
  )
  and created_at > :since
  and deleted_at is null;

select id
from users
where
  not (email like '%@example.com')
  and age between 18 and 65;
