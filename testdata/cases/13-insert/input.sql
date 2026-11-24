INSERT INTO users(id,email,created_at) VALUES(:id,:email,now()) RETURNING id;
insert into tags (name) values ('a'), ('b'), ('c');
insert into archived_users (id, email) select id, email from users where deleted_at is not null;
