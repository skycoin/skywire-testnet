package conn

const (
	STATUS_CONNECTING = iota
	STATUS_CONNECTED
	STATUS_ERROR
)

const (
	TCP_PINGTICK_PERIOD  = 60
	UDP_PING_TICK_PERIOD = 10
	UDP_GC_PERIOD        = 90
)

const (
	highGain  = 2.885
	drainGain = 1 / highGain
)

var (
	pacingGain = [...]float64{
		1.25, 0.75, 1, 1, 1, 1, 1, 1,
	}
)
