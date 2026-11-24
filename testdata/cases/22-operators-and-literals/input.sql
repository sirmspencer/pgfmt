select a+b, a-b*c, a||b, x!=y, x>=y, NOT flag, a IS NOT DISTINCT FROM b, NULL, TRUE, FALSE from t where a=ANY(:ids);
