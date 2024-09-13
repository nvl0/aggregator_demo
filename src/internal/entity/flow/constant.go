package flow

import (
	"io/fs"
)

// AllRWX доступы файла
const AllRWX fs.FileMode = 0777

// FlowNameSubStr имя файла с которого начинается файл flow
const FlowNameSubStr = "ft-"

// FlowTempDir директория готовых flow файлов
const FlowTempDir = "tmp"

// FlowHeader заголовок flow файла
const FlowHeader = "#:doctets,srcaddr,dstaddr"

// SubnetFileName название файла с подсетями
type SubnetFileName string

const (
	// InternalDisabled имя файла с исключенными из отчета подсетями
	InternalDisabled SubnetFileName = "internal"
)

// GitKeepName название файла для сохранения директорий git
const GitKeepName = ".gitkeep"
