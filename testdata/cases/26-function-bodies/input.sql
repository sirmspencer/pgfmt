CREATE FUNCTION find_events(user_id BIGINT, kind TEXT, since TIMESTAMPTZ, max_rows INT DEFAULT 100) RETURNS SETOF events LANGUAGE sql AS $$
SELECT * FROM events WHERE events.user_id = find_events.user_id AND events.kind = find_events.kind LIMIT max_rows
$$;
create function one() returns int language sql as $$ SELECT 1 $$;
CREATE FUNCTION close_order(order_id bigint) RETURNS integer LANGUAGE plpgsql AS $$
DECLARE
  n integer;
BEGIN
  SELECT count(*) INTO n FROM order_items i WHERE i.order_id = close_order.order_id AND i.status <> 'void';
  IF n = 0 THEN
    RAISE EXCEPTION 'order % has no items', order_id;
  END IF;
  UPDATE orders SET status = 'closed', closed_at = now() WHERE id = order_id;
  IF EXISTS (SELECT 1 FROM refunds r WHERE r.order_id = order_id) THEN DELETE FROM refunds WHERE refunds.order_id = close_order.order_id AND refunds.status = 'pending'; END IF;
  RETURN QUERY SELECT id FROM orders WHERE id = order_id;
  SELECT total INTO STRICT n FROM orders WHERE id = order_id;
  RETURN n;
END;
$$;
