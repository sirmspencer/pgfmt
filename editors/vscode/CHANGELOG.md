# Changelog

## 0.1.6

- Placeholders can stand for a parenthesized list, as in `values :rows` and `in :ids`, for set assignments, as in `update t set :assignments`, and for elements of an `array[...]` constructor.
- A placeholder after a dot, as in `t.:column`, is read as a name, so the statement keeps its meaning.
- A placeholder fragment stays between the words it was written between, as in `on conflict :target do nothing`, including at the end of a statement without a semicolon.
- A typed placeholder keeps its form: `interval :age` is no longer rewritten as `:age::interval`.

## 0.1.0

- First release.
