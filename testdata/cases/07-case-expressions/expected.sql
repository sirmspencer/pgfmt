select
  id
  , case
    when status = 'a' then 'active'
    when status = 'd' then 'disabled'
    else 'unknown'
  end as status_label
  , case role
    when 'admin' then 1
    else 0
  end as is_admin
from users;
