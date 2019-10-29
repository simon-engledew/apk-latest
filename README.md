# apk-latest
Print the latest available versions of Alpine packages

Useful for figuring out what to pin when writing docker files:

```Dockerfile
# dependencies could change between builds if a remote package is updated
RUN apk add --no-cache git make gcc
```

## install

`go get github.com/simon-engledew/apk-latest/go/cmd/apk-latest`

## run

```
$ apk-latest git make gcc
git==2.22.0-r0 make==4.2.1-r2 gcc==8.3.0-r0
```

## apply

```Dockerfile
# this should always build the same or break trying
RUN apk add --no-cache git==2.22.0-r0 make==4.2.1-r2 gcc==8.3.0-r0
```
