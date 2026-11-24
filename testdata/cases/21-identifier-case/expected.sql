select userid, "DisplayName", public.users.email
from public.users
where "Status" = 'Active';
