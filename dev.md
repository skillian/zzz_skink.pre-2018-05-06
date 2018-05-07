# Skink Developers README

## Naming Convention

### `Create` vs. `Make` vs. `New`

It probably seems arbitrary, but there is a difference about when to use one or
another (though the distinction may be useless):

  - `Create` is prefixed on functions that create new values and that operation
    might not succeed.  The last return value is always an `error`.

  - `Make` is used for constructors that initialize values and should not fail.
    An example is `MakeString` which takes a Go `string` and creates a
    `skink.String`.  None of the functions within a `Make` function should
    return `error`s.  If they do, then change the `Make` function to a `Create`
    function.

  - `New` is just like `Make` except the new value is explicitly allocated.
    It's supposed to make it clear(er?) that the returned value is pointed-to
    elsewhere and mutating it will result in mutations visible elsewhere, too.
