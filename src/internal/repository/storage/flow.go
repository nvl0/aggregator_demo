package storage

import (
	"fmt"
	"os"
	"strings"

	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/repository"
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
		return dirNameList, err
	}

	for _, dir := range dirList {
		if dir.IsDir() {
			dirNameList = append(dirNameList, dir.Name())
		}
	}

	if len(dirNameList) == 0 {
		err = global.ErrNoData
	}

	return dirNameList, err
}

// ReadDirFileNames считать имена файлов по пути
func (r *flowRepository) ReadFileNamesInFlowDir(dirName string) (fileNameList []string, err error) {
	dirList, err := os.ReadDir(fmt.Sprintf("%s/%s", r.flowDirPath, dirName))
	if err != nil {
		return fileNameList, err
	}

	for _, dir := range dirList {
		if !dir.IsDir() {
			fileNameList = append(fileNameList, dir.Name())
		}
	}

	if len(fileNameList) == 0 {
		err = global.ErrNoData
	}

	return fileNameList, err
}

// MoveFlowToTempDir переместить бинарник flow из надлежащей директории в директорию tmp
func (r *flowRepository) MoveFlowToTempDir(dirName, fileName string) error {
	// создание директории ./tmp
	// вызывается на каждый файл в dirName, поэтому "уже существует" не ошибка
	tmpDirPath := fmt.Sprintf("%s/%s/%s", r.flowDirPath, dirName, flow.FlowTempDir)
	if err := os.Mkdir(tmpDirPath, flow.AllRWX); err != nil && !os.IsExist(err) {
		return err
	}

	return os.Rename(
		// до nas_ip/ft-*
		fmt.Sprintf("%s/%s/%s", r.flowDirPath, dirName, fileName),
		// после nas_ip/tmp/ft-*
		fmt.Sprintf("%s/%s/%s/%s", r.flowDirPath, dirName, flow.FlowTempDir, fileName),
	)
}

// ReadFlow считать бинарники flow из директории tmp.
// Содержимое файлов, перечисленных в skipFileNames, в output не попадает:
// их чанки уже закоммичены в предыдущем цикле и повторный подсчет задвоил бы трафик.
// fileNameList содержит имена всех найденных flow файлов, включая пропущенные.
func (r *flowRepository) ReadFlow(
	dirName string,
	skipFileNames map[string]bool,
) (output string, fileNameList []string, err error) {
	path := fmt.Sprintf("%s/%s/%s", r.flowDirPath, dirName, flow.FlowTempDir)

	dirList, err := os.ReadDir(path)
	if err != nil {
		return output, fileNameList, err
	}

	var (
		sumB []byte
		b    []byte
	)

	for _, dir := range dirList {
		// вложенные директории и служебные файлы (.gitkeep) не являются flow
		if dir.IsDir() || !strings.Contains(dir.Name(), flow.FlowNameSubStr) {
			continue
		}

		fileNameList = append(fileNameList, dir.Name())

		if skipFileNames[dir.Name()] {
			continue
		}

		b, err = os.ReadFile(fmt.Sprintf("%s/%s", path, dir.Name()))
		if err != nil {
			return output, fileNameList, err
		}

		sumB = append(sumB, b...)
	}

	output = string(sumB)

	return output, fileNameList, err
}

// RemoveOld удаляет старый flow
func (r *flowRepository) RemoveOld(nasIP string) (err error) {
	path := fmt.Sprintf("%s/%s/%s", r.flowDirPath, nasIP, flow.FlowTempDir)
	if err = os.RemoveAll(path); err != nil {
		return err
	}
	// создание директории ./tmp
	if err = os.Mkdir(path, flow.AllRWX); err != nil {
		return err
	}

	return os.WriteFile(fmt.Sprintf("%s/%s", path, flow.GitKeepName), []byte{}, flow.AllRWX)
}
