CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;
CREATE FUNCTION find_events(user_id BIGINT, kind TEXT, since TIMESTAMPTZ, max_rows INT DEFAULT 100) RETURNS SETOF events LANGUAGE sql AS $$
SELECT * FROM events WHERE events.user_id = find_events.user_id LIMIT max_rows
$$;
