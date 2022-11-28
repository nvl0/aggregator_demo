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
