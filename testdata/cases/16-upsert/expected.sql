insert into user_settings (user_id, key, value)
values (:user_id, :key, :value)
on conflict (user_id, key) do update
set
    value = excluded.value,
    updated_at = now();

insert into tags (name)
values (:name)
on conflict do nothing;
