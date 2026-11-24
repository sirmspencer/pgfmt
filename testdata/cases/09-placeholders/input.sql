-- find users by :id within a tenant
select id, :tenant_id as tenant
from users
where id = :id::bigint and tags[1:2] = :tags and note <> 'time: 10:30' and created_at > :since;
select id from users where owner_id = :id or creator_id = :id;
