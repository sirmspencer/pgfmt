update users
set
    email = :email,
    updated_at = now()
where id = :id
returning
    id,
    email;

update users
set active = false
where last_login < :cutoff;

update orders as o
set status = 'void'
from users as u
where
    u.id = o.user_id
    and u.deleted_at is not null;
