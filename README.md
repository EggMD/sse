# sse

Middleware sse provides Server-Sent Events to channels binding for [Macaron](https://github.com/go-macaron/macaron).

### Installation

	go get github.com/go-macaron/sockets

### Usage

Have a look into the [example directory](https://github.com/EggMD/sse) to get a feeling for how to use the sse package.

This package essentially provides a binding of Server-Sent Event to channels, which you can use as in the following,
contrived example:

```go
m.Get("/stat", sse.Handler(stat{}), func(msg chan<- *stat) {
    for {
        select {
        case <-time.Tick(1 * time.Second):
            msg <- getStat()
        }
    }
})
```