WITH recent AS (SELECT o.account_id, o.total, (SELECT max(p.paid_at) FROM payments p WHERE p.order_id = o.id AND p.status IN (SELECT s.code FROM statuses s WHERE s.kind = 'settled' AND EXISTS (SELECT 1 FROM status_flags f WHERE f.code = s.code AND f.active))) AS last_paid FROM orders o WHERE o.created_at > now() - interval '30 days'), ranked AS (SELECT r.*, row_number() OVER (PARTITION BY r.account_id ORDER BY r.total DESC) AS rn FROM recent r) SELECT a.id, coalesce(nullif(trim(a.name), ''), (SELECT u.email FROM users u WHERE u.account_id = a.id ORDER BY u.created_at LIMIT 1), 'unknown') AS label, CASE WHEN rk.total > 1000 THEN CASE WHEN rk.last_paid IS NULL THEN 'big unpaid' ELSE 'big paid' END WHEN rk.total > 100 THEN 'medium' ELSE 'small' END AS size FROM accounts a INNER JOIN ranked rk ON rk.account_id = a.id AND rk.rn = 1 WHERE a.active AND (a.region = :region OR (a.region IS NULL AND (a.parent_id IN (SELECT id FROM accounts WHERE region = :region) OR a.id = ANY(:extra-ids)))) ORDER BY rk.total DESC;
SELECT t.account_id, t.total FROM (SELECT x.account_id, sum(x.total) AS total FROM (SELECT o.account_id, o.total FROM orders o WHERE o.id IN (SELECT i.order_id FROM order_items i WHERE i.sku IN (SELECT sku FROM products WHERE category_id IN (SELECT id FROM categories WHERE parent_id = :category)))) x GROUP BY x.account_id) t WHERE t.total > (SELECT avg(total) FROM (SELECT sum(total) AS total FROM orders GROUP BY account_id) per_account);
SELECT jsonb_build_object('id', a.id, 'owner', jsonb_build_object('id', u.id, 'tags', array[u.a, u.b, upper(lower(trim(u.c)))])) FROM accounts a INNER JOIN users u ON u.id = a.owner_id;
SELECT id FROM accounts WHERE NOT (a AND (b OR (c AND (d OR e))));
SELECT id FROM (SELECT id FROM accounts UNION ALL SELECT id FROM archived_accounts) all_accounts WHERE id IN (SELECT account_id FROM orders INTERSECT SELECT account_id FROM payments);
CREATE FUNCTION settle(account bigint) RETURNS integer LANGUAGE plpgsql AS $$
DECLARE
  n integer;
BEGIN
  FOR i IN 1..3 LOOP
    IF i = 1 THEN
      UPDATE orders SET status = 'settled' WHERE account_id = account AND id IN (SELECT order_id FROM payments WHERE status = 'paid' AND amount > (SELECT avg(amount) FROM payments));
    ELSE
      IF i = 2 THEN DELETE FROM holds WHERE account_id = account AND expires_at < now(); END IF;
    END IF;
  END LOOP;
  SELECT count(*) INTO n FROM orders WHERE account_id = account AND status IN (SELECT code FROM statuses WHERE kind = 'settled');
  RETURN n;
END;
$$;
