create or replace function touch_updated_at()
returns trigger
language plpgsql
as $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;
