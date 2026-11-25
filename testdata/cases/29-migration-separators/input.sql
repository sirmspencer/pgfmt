DROP TABLE IF EXISTS app.first_table CASCADE;
--;;
DROP TABLE IF EXISTS app.second_table CASCADE;

--;;

CREATE TABLE app.third_table (id bigint PRIMARY KEY);
-- a real comment
--;;
ALTER TABLE app.third_table ADD COLUMN note text;
