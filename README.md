# genw

Generate a wordlist from the query parameter names and path segments of a list of URLs.

## Build

```sh
go build -o genw .
```

## Usage

```sh
# from a file
./genw -input urls.txt

# from stdin
cat urls.txt | ./genw -output words.txt
```

| Flag | Description |
| --- | --- |
| `-input` | File with one URL per line. Defaults to stdin. |
| `-output` | File to append new words to. Defaults to `wordlist.txt`. |

Blank lines and lines starting with `#` are ignored.

## Example

Input:

```
https://example.com/search?q=shoes&page=2
https://example.com/blog/travel?tag=asia
```

Output (stdout and appended to the output file, order may vary):

```
q
page
search
tag
blog
travel
```

Words are deduplicated across query parameters and paths. Set `-output ""` to only print to stdout.
