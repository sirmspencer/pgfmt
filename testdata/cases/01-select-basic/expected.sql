select
  u.id
  , u.email
  , u.created_at
from users as u
where u.active = true;
