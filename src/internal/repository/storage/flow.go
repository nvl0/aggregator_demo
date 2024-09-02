package storage

import (
	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/repository"
	"fmt"
	"os"
)

type flowRepository struct {
	flowDirPath        string
	disabledSubnetPath string
}

func NewFlowRepository(flowDirPath, disabledSubnetPath string) repository.Flow {
	return &flowRepository{
		flowDirPath,
		disabledSubnetPath,
	}
}

// ReadDirNames считать имена директорий по пути
func (r *flowRepository) ReadFlowDirNames() (dirNameList []string, err error) {
	dirList, err := os.ReadDir(r.flowDirPath)
	if err != nil {
		return
	}

	for _, dir := range dirList {
		if dir.IsDir() {
			dirNameList = append(dirNameList, dir.Name())
		}
	}

	if len(dirNameList) == 0 {
		err = global.ErrNoData
	}

	return
}

// ReadDirFileNames считать имена файлов по пути
func (r *flowRepository) ReadFileNamesInFlowDir(dirName string) (fileNameList []string, err error) {
	dirList, err := os.ReadDir(fmt.Sprintf("%s/%s", r.flowDirPath, dirName))
	if err != nil {
		return
	}

	for _, dir := range dirList {
		if !dir.IsDir() {
			fileNameList = append(fileNameList, dir.Name())
		}
	}

	if len(fileNameList) == 0 {
		err = global.ErrNoData
	}

	return
}

// MoveFlowToTempDir переместить бинарник flow из надлежащей директории в директорию tmp
func (r *flowRepository) MoveFlowToTempDir(dirName, fileName string) error {
	// создание директории ./tmp
	os.Mkdir(fmt.Sprintf("%s/%s/%s", r.flowDirPath, dirName, flow.FlowTempDir), flow.AllRWX)

	return os.Rename(
		// до nas_ip/ft-*
		fmt.Sprintf("%s/%s/%s", r.flowDirPath, dirName, fileName),
		// после nas_ip/tmp/ft-*
		fmt.Sprintf("%s/%s/%s/%s", r.flowDirPath, dirName, flow.FlowTempDir, fileName),
	)
}

// ReadFlow считать бинарник flow по пути
func (r *flowRepository) ReadFlow(dirName string) (output string, err error) {
	path := fmt.Sprintf("%s/%s/%s", r.flowDirPath, dirName, flow.FlowTempDir)

	dirList, err := os.ReadDir(path)
	if err != nil {
		return
	}

	var (
		sumB []byte
		b    []byte
	)

	for _, dir := range dirList {
		if !dir.IsDir() {
			b, err = os.ReadFile(fmt.Sprintf("%s/%s", path, dir.Name()))
			if err != nil {
				return
			}
			sumB = append(sumB, b...)
		}
	}

	output = string(sumB)
	return
}
