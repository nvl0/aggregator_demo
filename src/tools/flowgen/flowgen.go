package flowgen

import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"net"
	"os"
	"time"
)

const (
	ip1   = "127.0.0.1"
	nasIP = "127.0.0.0"
	path  = "./flow/127.0.0.0/"
)

// "#:doctets,srcaddr,dstaddr\n" 26 byte max
// record 31 byte max
// total 26 + (31 * i)

func Generate() (download, upload int, err error) {
	flowBytes := make([]byte, 0, 357)
	flowBytes = append(flowBytes, []byte("#:doctets,srcaddr,dstaddr\n")...)
	ipBuf := make([]byte, 4)

	for i := 0; i < 100; i++ {
		binary.LittleEndian.PutUint32(ipBuf, rand.Uint32())
		bytesTransfered := rand.IntN(rand.IntN(10000))
		if i%2 == 0 {
			download += bytesTransfered
			flowBytes = append(flowBytes, []byte(
				fmt.Sprintf("%d,%s,%s\n", bytesTransfered, ip1, net.IP(ipBuf).String()),
			)...)
		} else {
			upload += bytesTransfered
			flowBytes = append(flowBytes, []byte(
				fmt.Sprintf("%d,%s,%s\n", bytesTransfered, net.IP(ipBuf).String(), ip1),
			)...)
		}
	}

	fileName := fmt.Sprintf("ft-%s", time.Now().Format("02.01.2006-15:04:05"))
	if err = os.WriteFile(path+fileName, flowBytes, 0644); err != nil {
		return
	}

	return
}
