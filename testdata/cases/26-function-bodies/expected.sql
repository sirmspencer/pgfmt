create function find_events(
  user_id bigint,
  kind text,
  since timestamptz,
  max_rows integer default 100
)
  returns setof events
  language sql
as $$
  select *
  from events
  where
    events.user_id = find_events.user_id
    and events.kind = find_events.kind
  limit max_rows
$$;

create function one()
  returns integer
  language sql
as $$ select 1 $$;

create function close_order(order_id bigint)
  returns integer
  language plpgsql
as $$
  declare
    n integer;
  begin
    select count(*)
    into n
    from order_items as i
    where
      i.order_id = close_order.order_id
      and i.status <> 'void';
    if n = 0 then
      raise exception 'order % has no items', order_id;
    end if;
    update orders
    set
      status    = 'closed',
      closed_at = now()
    where id = order_id;
    if exists (select 1 from refunds r where r.order_id = order_id) then
      delete from refunds
      where
        refunds.order_id = close_order.order_id
        and refunds.status = 'pending';
    end if;
    return query
      select id
      from orders
      where id = order_id;
    select total into strict n from orders where id = order_id;
    return n;
  end;
$$;
