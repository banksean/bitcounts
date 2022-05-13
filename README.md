# bitcounts

Count the number of bits set to 1 in all of the files under the current directory.

For example I run it like so, from `/` on my macbook air:

```
sudo go run /Users/banksean/github/banksean/bitcounts/main.go
```

It runs for a few minutes, logging each filename to stdout as it goes. After it finishes reading, it prints out how many of the bits it read were ones and how many were zeroes.
