package measure

// Writer интерфейс модуля, которые будет писать куда-либо результаты
type Writer interface {
	Write(text string)
}
