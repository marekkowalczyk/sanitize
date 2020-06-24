# sanitize
Sanitize a string for use as a filename

Usage: `sanitize <input-text>`

Examples:

- `sanitize This Is a TEXT` --> `this-is-a-text`
- `sanitize THIS_is a+a-TEXT` --> `this-is-a-text`

Caution: Different input strings can result in identical output.

All punctuation and spaces are converted to single `-`. No `-`s are left at the beginning or end of output string. All capital letters are converted to lowercase. 

- `sanitize 1 2 3` --> `1-2-3`

Numbers are not affected.

- `sanitize Łączność` --> `lacznosc`

Diacritics are converted to their basic letters of the English alphabet.

If punctuation is getting in the way of the shell interpreting input correctly, escape `"input"` with quotes.

- `sanitize abc; de` --> `bash` throws an error `de: command not found`
- `sanitize "abc; de"` --> `abc-de` works as expected

`san.sh <filenames>` is just a wrapper that parses filenames into names and extensions, calls `sanitize` on both and renames original files (will not overwrite existing files).
