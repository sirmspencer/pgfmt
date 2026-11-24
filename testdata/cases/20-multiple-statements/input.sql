select 1;select 2;


select 3;
begin; update users set active=false where id=:id; commit;
