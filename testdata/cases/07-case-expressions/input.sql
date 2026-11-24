select id, CASE WHEN status='a' THEN 'active' WHEN status = 'd' then 'disabled' ELSE 'unknown' END as status_label, case role when 'admin' then 1 else 0 end as is_admin from users;
