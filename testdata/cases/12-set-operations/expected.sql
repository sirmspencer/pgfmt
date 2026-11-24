select id
from users

union all

select id
from archived_users

union

select id
from pending_users;
