package phoneadd

import (
	"bufio"
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// LANAddr picks the IPv4 address phones on the local network can reach,
// preferring the interface that holds the default route.
func LANAddr() (net.IP, error) {
	if f, err := os.Open("/proc/net/route"); err == nil {
		name := defaultRouteIface(f)
		f.Close()
		if name != "" {
			if ip := ifaceIPv4(name); ip != nil {
				return ip, nil
			}
		}
	}
	// Fallback: any up, non-loopback interface with a private IPv4 address.
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		if ip := ifaceIPv4(ifc.Name); ip != nil && ip.IsPrivate() {
			return ip, nil
		}
	}
	return nil, errors.New("no local network address found")
}

// defaultRouteIface reads /proc/net/route content and returns the interface of
// the default route with the lowest metric.
func defaultRouteIface(r io.Reader) string {
	var best string
	bestMetric := -1
	sc := bufio.NewScanner(r)
	sc.Scan() // header
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 7 || f[1] != "00000000" {
			continue
		}
		flags, err := strconv.ParseUint(f[3], 16, 32)
		if err != nil || flags&0x1 == 0 { // RTF_UP
			continue
		}
		metric, _ := strconv.Atoi(f[6])
		if bestMetric < 0 || metric < bestMetric {
			best, bestMetric = f[0], metric
		}
	}
	return best
}

func ifaceIPv4(name string) net.IP {
	ifc, err := net.InterfaceByName(name)
	if err != nil {
		return nil
	}
	addrs, _ := ifc.Addrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok {
			if ip := n.IP.To4(); ip != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
				return ip
			}
		}
	}
	return nil
}
