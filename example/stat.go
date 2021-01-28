package main

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"gopkg.in/macaron.v1"

	"github.com/EggMD/sse"

	"github.com/shirou/gopsutil/v3/mem"
)

type stat struct {
	CpuStats          []cpu.TimesStat
	TotalMemory       uint64
	FreeMemory        uint64
	UsedMemoryPercent float64
}

func main() {
	m := macaron.Classic()

	// Use Renderer
	m.Use(macaron.Renderer())

	m.Get("/", func(c *macaron.Context) {
		c.HTML(200, "index", "")
	})

	m.Get("/stat", sse.Handler(stat{}), func(msg chan<- *stat) {
		msg <- getStat()

		for {
			select {
			case <-time.Tick(1 * time.Second):
				msg <- getStat()
			}
		}
	})

	m.Run()
}

func getStat() *stat {
	v, _ := mem.VirtualMemory()
	c, _ := cpu.Times(true)

	return &stat{
		CpuStats:          c,
		TotalMemory:       v.Total,
		FreeMemory:        v.Free,
		UsedMemoryPercent: v.UsedPercent,
	}
}
