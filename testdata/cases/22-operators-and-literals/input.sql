select a+b, a-b*c, a||b, x!=y, x>=y, NOT flag, a IS NOT DISTINCT FROM b, NULL, TRUE, FALSE from t where a=ANY(:ids);
select a*b, a/b, a%b, a^b, -a, a<b, a<=b, a>b from t where a=b and c<>d;
select a  +   b from t where a   =  b;
