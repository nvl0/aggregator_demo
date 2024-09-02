package traffic

// Traffic трафик
type Traffic struct {
	Download int // получено
	Upload   int // отдано
}

// NewEmptyTraffic конструктор пустого трафика
func NewEmptyTraffic() Traffic {
	return Traffic{}
}

// NewTrafficDownload конструктор записи скачивания
func NewTrafficDownload(byteSize int) Traffic {
	return Traffic{
		Download: byteSize,
	}
}

// NewTrafficUpload конструктор записи отдано
func NewTrafficUpload(byteSize int) Traffic {
	return Traffic{
		Upload: byteSize,
	}
}

// Merge сложить трафик вместе
func (t *Traffic) Merge(t1 Traffic) {
	t.Download += t1.Download
	t.Upload += t1.Upload
}
