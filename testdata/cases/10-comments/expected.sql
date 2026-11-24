-- leading comment for the query
select
    id, -- primary key
    email /* contact */
from users
-- only active users
where active = true; -- trailing
