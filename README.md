# sanitize

Sanitize / normalize a string for use as a safe filename.

Usage: `sanitize <input-text>`

Examples:

- `sanitize This Is a TEXT` --> `this-is-a-text`
- `sanitize THIS_is a+a-TEXT` --> `this-is-a-text`

Caution: Different input strings can result in identical output.

All punctuation and spaces are converted to single `-`. No `-`s are left at the beginning or end of output string. All capital letters are converted to lowercase (see Issue #1). 

- `sanitize 1 2 3` --> `1-2-3`

Numbers are not affected.

- `sanitize Łączność` --> `lacznosc`

Diacritics are converted to their basic letters of the English alphabet.

If punctuation is getting in the way of the shell interpreting input correctly, escape `"input"` with quotes.

- `sanitize abc; de` --> `bash` throws an error `de: command not found`
- `sanitize "abc; de"` --> `abc-de` works as expected

`san.sh <filenames>` is just a wrapper that parses filenames into names and extensions, calls `sanitize` on both and renames original files (will not overwrite existing files).

## Handling of non-ASCII characters

All non-ASCII chars will be transformed to their ASCII equivalents, e.g.: 

`Kąt na łące żre źrebię` --> `kat-na-lace-zre-zrebie`

This is achieved by the function

`runes.Remove(runes.In(unicode.Mn))`

which strips all unicode runes of the [[Mark, Nonspacing] characters](https://www.fileformat.info/info/unicode/category/Mn/index.htm) they are possibly combined with. E.g:

`ą` is actually `a` combined with `0328 Below_Attached # Mn ( ̨ ) COMBINING OGONEK`.

([see here for a complete list of Mn characters](https://unicode.org/L2/L2005/05134-nonspacing-pos.html)).

### The curious case of 'Ł' and 'ł'

Curiously, however, unlike all other Polish diacritic characters `Ł` and `ł` are *not* created by combining `L` or `l`with any [Mark, Nonspacing] character but are characters of their own. Therefore they need to be handled separately, as 

     runes.Remove(runes.In(unicode.Mn))

will not work for them, i.e., 

    runes.In(unicode.Mn)

fails on them and therefore there are no runes to remove by

     runes.Remove()
     
## Usage with DEVONthink

`DEVONthink-Sanitize-Filenames.applescript` sanitizes names of selected DEVONthink records, while setting the `Finder Comment` fields to original file names. Caution: the contents of the field is overwritten.
