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
	// DefaultNasIP nas_ip директория dev-режима
	DefaultNasIP = "127.0.0.0"
	// DefaultClientIP ip клиента dev-режима, входит в подсеть internal
	DefaultClientIP = "127.0.0.1"
	// DefaultFlowDir корень flow директории dev-режима
	DefaultFlowDir = "./flow"

	// dirPerm права создаваемой nas_ip директории
	dirPerm = 0755
	// filePerm права создаваемого flow файла
	filePerm = 0644
	// recordCount количество записей в одном flow файле
	recordCount = 100
	// maxByteSize верхняя граница размера переданных байт в одной записи
	maxByteSize = 10000
	// flowBytesCapacity ожидаемый размер flow файла: 26 (заголовок) + 31 (max запись) * recordCount
	flowBytesCapacity = 357
	// ipv4ByteLen длина ipv4 адреса в байтах
	ipv4ByteLen = 4
)

// Params параметры генерации flow
type Params struct {
	// NasIP имя nas_ip директории
	NasIP string
	// ClientIP ip клиента, от лица которого генерируется трафик,
	// должен входить в подсеть internal, иначе трафик не будет посчитан
	ClientIP string
	// FlowDir корень flow директории
	FlowDir string
}

// "#:doctets,srcaddr,dstaddr\n" 26 byte max
// record 31 byte max
// total 26 + (31 * i)

// Generate генерация flow файла в <FlowDir>/<NasIP>/ft-<время>
func Generate(p Params) (download, upload int, err error) {
	dirPath := fmt.Sprintf("%s/%s", p.FlowDir, p.NasIP)
	if err = os.MkdirAll(dirPath, dirPerm); err != nil {
		return download, upload, err
	}

	flowBytes := make([]byte, 0, flowBytesCapacity)
	flowBytes = append(flowBytes, []byte("#:doctets,srcaddr,dstaddr\n")...)
	ipBuf := make([]byte, ipv4ByteLen)

	for i := range recordCount {
		binary.LittleEndian.PutUint32(ipBuf, rand.Uint32()) //nolint:gosec // dev-нагрузка, не крипто

		// +1 чтобы исключить панику rand.IntN(0)
		bytesTransfered := rand.IntN(maxByteSize) + 1 //nolint:gosec // dev-нагрузка, не крипто

		if i%2 == 0 {
			download += bytesTransfered
			flowBytes = append(flowBytes, []byte(
				fmt.Sprintf("%d,%s,%s\n", bytesTransfered, p.ClientIP, net.IP(ipBuf).String()),
			)...)
		} else {
			upload += bytesTransfered
			flowBytes = append(flowBytes, []byte(
				fmt.Sprintf("%d,%s,%s\n", bytesTransfered, net.IP(ipBuf).String(), p.ClientIP),
			)...)
		}
	}

	fileName := fmt.Sprintf("ft-%s", time.Now().Format("02.01.2006-15:04:05"))
	if err = os.WriteFile(fmt.Sprintf("%s/%s", dirPath, fileName), flowBytes, filePerm); err != nil {
		return download, upload, err
	}

	return download, upload, err
}
