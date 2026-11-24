select id
from users
where id in (1, 2, 3);

select id
from users
where id in (
  1
  , 2
  , 3
  , 4
);

select coalesce(a, b, c)
from t;

select coalesce(
  a
  , b
  , c
  , d
)
from t;

select substr(name, 1, 3)
from users;

select jsonb_build_object(
  'id'
  , u.id
  , 'name'
  , u.name
)
from users as u;

select id
from users
where created_at > make_timestamp(
  :year
  , :month
  , :day
  , 0
  , 0
  , 0
);

select jsonb_build_object(
  'id'
  , u.id
  , 'tags'
  , array[
    u.a
    , u.b
    , u.c
    , u.d
  ]
)
from users as u;

insert into events (id, kind, created_at)
values (:id, :kind, now());

insert into events (
  id
  , user_id
  , kind
  , created_at
)
values (
  :id
  , :user_id
  , :kind
  , now()
);

insert into events (
  id
  , user_id
  , kind
  , created_at
)
values
  (
    :id
    , :user_id
    , :kind
    , now()
  )
  , (
    :id2
    , :user_id2
    , :kind2
    , now()
  );
