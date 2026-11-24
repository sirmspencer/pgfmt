select
    coalesce(nickname, first_name) as display_name,
    created_at::date as created_on,
    amount::numeric(10, 2),
    now() - interval '1 day' as yesterday,
    row_number() over (partition by user_id order by created_at desc) as rn,
    count(distinct user_id)
from orders;
