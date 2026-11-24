delete from sessions
where expires_at < now()
returning id;

delete from sessions as s
using users as u
where
    u.id = s.user_id
    and u.active = false;
