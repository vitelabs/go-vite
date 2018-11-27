package monitor

import (
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"net"
	"sort"
	"time"
)

// NTPV4 client

const (
	server = "pool.ntp.org"
	times  = 3
)

var threshold = 10 * time.Second

type durations []time.Duration

func (s durations) Len() int           { return len(s) }
func (s durations) Less(i, j int) bool { return s[i] < s[j] }
func (s durations) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

var ntp_logger = log15.New("module", "ntp")

func init() {
	checkTime()

	go checkLoop()
}

func checkLoop() {
	t := time.NewTicker(time.Minute)
	for {
		select {
		case <-t.C:
			checkTime()
		}
	}
}

func checkTime() {
	addr, err := net.ResolveUDPAddr("udp", server+":123")
	if err != nil {
		ntp_logger.Error(fmt.Sprintf("ntp server address parse error: %v", err))
		return
	}

	drift, err := request(times, addr)
	if err != nil {
		ntp_logger.Error(fmt.Sprint("can not get ntp server time"))
		return
	}

	if drift < -threshold || drift > threshold {
		ntp_logger.Error(fmt.Sprintf("too much delta to ntp server: %s", drift))
	} else {
		ntp_logger.Info(fmt.Sprintf("time dela to ntp server: %s", drift))
	}
}

func request(times int, saddr *net.UDPAddr) (time.Duration, error) {
	// Construct the time request (empty package with only 2 fields set):
	//   Bits 3-5: Protocol version, 4
	//   Bits 6-8: Mode of operation, client, 3
	request := make([]byte, 48)
	request[0] = 4<<3 | 3

	times += 2
	ds := make(durations, 0, times)

	for i := 0; i < times; i++ {
		conn, err := net.DialUDP("udp", nil, saddr)
		if err != nil {
			return 0, err
		}
		defer conn.Close()

		sent := time.Now()
		conn.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err = conn.Write(request); err != nil {
			return 0, err
		}

		reply := make([]byte, 48)
		if _, err = conn.Read(reply); err != nil {
			return 0, err
		}
		elapsed := time.Since(sent)

		// extract ntp server time
		sec := uint64(reply[43]) | uint64(reply[42])<<8 | uint64(reply[41])<<16 | uint64(reply[40])<<24
		frac := uint64(reply[47]) | uint64(reply[46])<<8 | uint64(reply[45])<<16 | uint64(reply[44])<<24

		nanosec := sec*1e9 + (frac*1e9)>>32

		serverTime := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(nanosec)).Local()

		// local time - ntp server time
		ds = append(ds, sent.Sub(serverTime)+elapsed/2)
	}

	// Calculate average delta (drop two extremities to avoid outliers)
	sort.Sort(durations(ds))

	delta := time.Duration(0)
	for i := 1; i < len(ds)-1; i++ {
		delta += ds[i]
	}

	return delta / time.Duration(times), nil
}
