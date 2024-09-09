package flow

import "net"

// Flow структура flow файла
type Flow struct {
	NasIP  string // nas папка именована в виде nas ip
	Output string // спарсенный flow
}

// IsEmpty проверка на пустоту
func (f *Flow) IsEmpty() bool {
	return f.NasIP == "" || f.Output == ""
}

// NewFlow конструктор
func NewFlow(nasIp, output string) Flow {
	return Flow{
		NasIP:  nasIp,
		Output: output,
	}
}

// Record одна строка с файла flow
type Record struct {
	SrcIP    net.IP // получатель
	DstIP    net.IP // отправитель
	ByteSize int    // всего байт
}

func (r *Record) Empty() {
	r.SrcIP = nil
	r.DstIP = nil
	r.ByteSize = 0
}

// SrcIPkey получение строки получателя
func (r Record) SrcIPkey() string {
	return r.SrcIP.String()
}

// DstIPkey получение строки отправителя
func (r Record) DstIPkey() string {
	return r.DstIP.String()
}
