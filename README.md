# diffmirror

A little utility that forwards requests to two hosts and compares their responses, logging differences and performance information.

It is likely most useful when paired with a tool like [gor](/buger/gor) to mirror production traffic to it.

## Features

- Requests that generate different responses can be recorded for later playback (in a [gor](/buger/gor) compatible file).

- Statistics -- number of matching vs different, latency, etc -- can be recorded to graphite and/or the console.

- Compare whole responses or just bodies -- often differences in headers are unavoidable or otherwise uninteresting, so choose what to compare (the `Date` header is always removed before comparison though).

- Optionally exclude errors responses from difference counts -- tracking errors (5xx returns or network issues) on their own and excluding them from diff counts helps keep numbers cleaner.


## Usage
`diffmirror [options] port [aliasA=]hostA [aliasB=]hostB`

- `port` is a tcp listen spec, eg `127.0.0.1:8000` or `:8000`. 
If it does not contain a `:`, one will be prepended, so just `8000` works fine too.

- `aliasA=hostA` will be split at the `=` and the first part used as that host's name in stats reporting. 
If an alias isn't provided, `A` and `B` will be used. 

## Options

####  `--body-only` (or `=false`)
compare only the body of responses (exclude headers) (default: true)

####  `--graphite hostname:port`
address of graphite receiver for stats

####  `--graphite-prefix foo.bar`
prefix for graphite writes

####  `--ignore-errors` (or `=false`)
ignore network errors and 5xx responses (default: true)

####  `--requestsfile foo.bin`
filename in which to store requests that generated diffs

####  `--stats`
print stats to console periodically (default true)

####  `--workers 10`
number of worker threads (default 10)

# Credits
diffmirror is developed at [Foursquare](/foursquare) and was heavily inspired by [gor](/buger/gor) and [clever/http-science](/clever/http-science).

### Authors
  - [David Taylor](/dt)
