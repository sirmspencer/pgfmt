create or replace function touch_updated_at()
  returns trigger
  language plpgsql
as $$
  begin
    new.updated_at = now();
    return new;
  end;
$$;

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
  where events.user_id = find_events.user_id
  limit max_rows
$$;
