with recent as (
  select
    o.account_id,
    o.total,
    (
      select max(p.paid_at)
      from payments as p
      where
        p.order_id = o.id
        and p.status in (
          select s.code
          from statuses as s
          where
            s.kind = 'settled'
            and exists (
              select 1
              from status_flags as f
              where
                f.code = s.code
                and f.active
            )
        )
    ) as last_paid
  from orders as o
  where o.created_at > now() - interval '30 days'
),
ranked as (
  select r.*, row_number() over (partition by r.account_id order by r.total desc) as rn
  from recent as r
)
select
  a.id,
  coalesce(
    nullif(trim(a.name), ''),
    (
      select u.email
      from users as u
      where u.account_id = a.id
      order by u.created_at
      limit 1
    ),
    'unknown'
  ) as label,
  case
    when rk.total > 1000 then case
      when rk.last_paid is null then 'big unpaid'
      else 'big paid'
    end
    when rk.total > 100 then 'medium'
    else 'small'
  end as size
from accounts as a
inner join ranked as rk
  on rk.account_id = a.id
  and rk.rn = 1
where
  a.active
  and (
    a.region = :region
    or (
      a.region is null
      and (
        a.parent_id in (
          select id
          from accounts
          where region = :region
        )
        or a.id = any(:extra-ids)
      )
    )
  )
order by rk.total desc;

select t.account_id, t.total
from (
  select x.account_id, sum(x.total) as total
  from (
    select o.account_id, o.total
    from orders as o
    where o.id in (
      select i.order_id
      from order_items as i
      where i.sku in (
        select sku
        from products
        where category_id in (
          select id
          from categories
          where parent_id = :category
        )
      )
    )
  ) as x
  group by x.account_id
) as t
where t.total > (
  select avg(total)
  from (
    select sum(total) as total
    from orders
    group by account_id
  ) as per_account
);

select jsonb_build_object(
  'id',
  a.id,
  'owner',
  jsonb_build_object(
    'id',
    u.id,
    'tags',
    array[u.a, u.b, upper(lower(trim(u.c)))]
  )
)
from accounts as a
inner join users as u
  on u.id = a.owner_id;

select id
from accounts
where not (a and (b or (c and (d or e))));

select id
from (
  select id
  from accounts

  union all

  select id
  from archived_accounts
) as all_accounts
where id in (
  select account_id
  from orders

  intersect

  select account_id
  from payments
);

create function settle(account bigint)
  returns integer
  language plpgsql
as $$
  declare
    n integer;
  begin
    for i in 1..3 loop
      if i = 1 then
        update orders
        set status = 'settled'
        where
          account_id = account
          and id in (
            select order_id
            from payments
            where
              status = 'paid'
              and amount > (
                select avg(amount)
                from payments
              )
          );
      else
        if i = 2 then
          delete from holds
          where
            account_id = account
            and expires_at < now();
        end if;
      end if;
    end loop;
    select count(*)
    into n
    from orders
    where
      account_id = account
      and status in (
        select code
        from statuses
        where kind = 'settled'
      );
    return n;
  end;
$$;
