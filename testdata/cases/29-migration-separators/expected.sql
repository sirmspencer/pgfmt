drop table if exists app.first_table cascade;
--;;
drop table if exists app.second_table cascade;
--;;
create table app.third_table (
  id bigint primary key
);
--;;
alter table app.third_table add column note text;
