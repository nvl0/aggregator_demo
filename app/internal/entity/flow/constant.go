package flow

import (
	"io/fs"
)

// UserRWE доступы файла
const UserRWE fs.FileMode = 0700

// FlowNameSubStr имя файла с которого начинается файл flow
const FlowNameSubStr = "ft-"

// FlowTempDir директория готовых flow файлов
const FlowTempDir = "tmp"

// FlowHeader заголовок flow файла
const FlowHeader = "#:doctets,srcaddr,dstaddr"
