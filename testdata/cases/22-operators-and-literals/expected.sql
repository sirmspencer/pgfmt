select
  a + b,
  a - b * c,
  a || b,
  x <> y,
  x >= y,
  not flag,
  a is not distinct from b,
  null,
  true,
  false
from t
where a = any(:ids);

select
  a * b,
  a / b,
  a % b,
  a ^ b,
  -a,
  a < b,
  a <= b,
  a > b
from t
where
  a = b
  and c <> d;

select a + b
from t
where a = b;
