DELETE FROM sessions WHERE expires_at < now() RETURNING id;
delete from sessions s using users u where u.id = s.user_id and u.active = false;
