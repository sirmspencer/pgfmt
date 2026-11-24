SELECT id FROM users WHERE active=true;
select id from users where active = true and deleted_at is null;
select id from users where role = 'admin' or role = 'owner';
select user_id from orders group by user_id having count(*)>5;
select user_id from orders group by user_id having count(*) > 5 and sum(total) > 100;
select user_id from orders group by user_id having count(*) > 5 or max(total) > 1000;
