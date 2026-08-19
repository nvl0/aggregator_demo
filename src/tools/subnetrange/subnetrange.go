package subnetrange

import (
	"net"
	"os"
	"strings"

	"github.com/yl2chen/cidranger"
)

func CreateDisabledSubnetRange(path string) (ranger cidranger.Ranger, err error) {
	var b []byte

	// считывание файла
	if b, err = os.ReadFile(path); err != nil {
		return ranger, err
	}

	output := string(b)

	// парсинг подсетей
	ranger = cidranger.NewPCTrieRanger()
	var network *net.IPNet

	// разделение файла подсетей построчно
	for _, item := range strings.Split(output, "\n") {
		item = strings.TrimSpace(item)

		// определение конца строки
		if len(item) > 0 {
			// парсинг подсети
			if _, network, err = net.ParseCIDR(item); err != nil {
				return ranger, err
			}

			// добавление подсети в блок
			if err = ranger.Insert(cidranger.NewBasicRangerEntry(*network)); err != nil {
				return ranger, err
			}
		}
	}

	return ranger, err
}
