UPDATE users SET email=:email, updated_at=now() WHERE id=:id RETURNING id, email;
update users set active = false where last_login < :cutoff;
update orders o set status = 'void' from users u where u.id = o.user_id and u.deleted_at is not null;
