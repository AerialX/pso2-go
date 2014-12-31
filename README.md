# pso2-go

A [go](http://golang.org) library and tools for
[Phantasy Star Online 2](http://pso2.jp)

This project's import prefix is `aaronlindsay.com/go/pkg/pso2`

## cmd

### cmd/pso2-download

[pso2-download](http://aaronlindsay.com/pso2) is an alternate launcher, patcher, and
downloader for PSO2.

### cmd/pso2-net

The `pso2-net` command can be used to set up servers. Example usage to set up
and run a PSO2 proxy:

    go get aaronlindsay.com/go/pkg/pso2/cmd/pso2-net
    $GOPATH/bin/pso2-net -priv serverkey.pem -pub segakey.pem

See [PSO2Proxy](https://github.com/cyberkitsune/PSO2Proxy) for instructions on
creating and retrieving these key files. Try `-help` for a list of additional
options the tool accepts.

## net

A PSO2 protocol library with server, client, and proxy implementations. See
`cmd/pso2-net` for usage examples.

### net/packets

Contains data structures and parsing tools for handling game packets.
