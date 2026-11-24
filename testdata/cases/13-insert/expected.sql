insert into users (id, email, created_at)
values (:id, :email, now())
returning id;

insert into tags (name)
values
    ('a'),
    ('b'),
    ('c');

insert into archived_users (id, email)
select
    id,
    email
from users
where deleted_at is not null;
