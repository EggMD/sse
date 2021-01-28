package sse

import (
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	"gopkg.in/macaron.v1"
)

const (
	defaultPingInterval = 10 * time.Second
)

type Options struct {
	// The time to wait between sending pings to the client
	PingInterval time.Duration
}

type Connection struct {
	*Options

	// Sender is the channel used for sending out data to the client.
	// This channel gets mapped for the next handler to use with the right type
	// and is asynchronous unless the SendChannelBuffer is set to 0.
	Sender reflect.Value

	// the request context
	context *macaron.Context

	// the ticker for pinging the client.
	ticker *time.Ticker
}

func Handler(bind interface{}, options ...*Options) macaron.Handler {
	return func(context *macaron.Context, req *http.Request, resp http.ResponseWriter) {
		context.Resp.Header().Set("Content-Type", "text/event-stream")
		context.Resp.Header().Set("Cache-Control", "no-cache")
		context.Resp.Header().Set("Connection", "keep-alive")
		context.Resp.Header().Set("X-Accel-Buffering", "no")

		sse := &Connection{
			Options: newOptions(options),
			context: context,
			Sender:  makeChanOfType(reflect.TypeOf(bind), 1),
		}
		context.Set(reflect.ChanOf(reflect.SendDir, sse.Sender.Type().Elem()), sse.Sender)

		go sse.handle()

		context.Next()
	}
}

// Creates new default options and assigns any given options
func newOptions(options []*Options) *Options {
	if len(options) == 0 {
		return &Options{
			PingInterval: defaultPingInterval,
		}
	}

	return options[0]
}

// Start the ticker used for pinging the client
func (c *Connection) startTicker() {
	c.ticker = time.NewTicker(c.PingInterval)
}

// Stop the ticker used for pinging the client
func (c *Connection) stopTicker() {
	c.ticker.Stop()
}

func (c *Connection) write(msg string) {
	_, _ = c.context.Write([]byte(msg))
}

func (c *Connection) flush() {
	c.context.Resp.Flush()
}

var (
	senderSend = 0
	tickerTick = 1
	timeout    = 2
)

func (c *Connection) handle() {
	c.startTicker()
	defer func() {
		c.stopTicker()
	}()

	c.write(": ping\n\n")
	c.write("events: stream opened\n\n")
	c.flush()

	cases := make([]reflect.SelectCase, 3)
	cases[senderSend] = reflect.SelectCase{reflect.SelectRecv, c.Sender, reflect.ValueOf(nil)}
	cases[tickerTick] = reflect.SelectCase{reflect.SelectRecv, reflect.ValueOf(c.ticker.C), reflect.ValueOf(nil)}
	cases[timeout] = reflect.SelectCase{reflect.SelectRecv, reflect.ValueOf(time.After(time.Hour)), reflect.ValueOf(nil)}

L:
	for {
		chosen, message, ok := reflect.Select(cases)
		switch chosen {

		case senderSend:
			if !ok {
				// Sender channel has been closed.
				return
			}

			c.write("data: ")
			evt, _ := json.Marshal(message.Interface())
			c.write(string(evt))
			c.write("\n\n")
			c.flush()

		case tickerTick:
			c.write(": ping\n\n")
			c.flush()

		case timeout:
			c.write("events: stream timeout\n\n")
			c.flush()
			break L
		}
	}

	c.write("event: error\ndata: eof\n\n")
	c.flush()
	c.write("events: stream closed")
	c.flush()
}

// Create a chan of the given type as a reflect.Value
func makeChanOfType(typ reflect.Type, chanBuffer int) reflect.Value {
	return reflect.MakeChan(reflect.ChanOf(reflect.BothDir, reflect.PtrTo(typ)), chanBuffer)
}
