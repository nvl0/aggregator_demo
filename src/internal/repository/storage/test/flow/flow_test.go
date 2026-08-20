package flow_test

import (
	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/repository/storage"

	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var flowDir = os.Getenv("FLOW_DIR")
var subnetDisabledDir = os.Getenv("SUBNET_DISABLED_DIR")

func TestReadFlowDirNames(t *testing.T) {
	r := require.New(t)

	const (
		dirName  = "test_dir"
		fileName = "test_file"
	)

	path := fmt.Sprintf("%s/%s", flowDir, dirName)

	r.NoError(os.Mkdir(path, flow.AllRWX))

	_, err := os.Create(fmt.Sprintf("%s/%s", path, fileName))
	r.NoError(err)

	t.Cleanup(func() {
		os.RemoveAll(path)
	})

	repo := storage.NewFlowRepository(flowDir, subnetDisabledDir)

	data, err := repo.ReadFlowDirNames()
	r.NoError(err)
	r.Contains(data, dirName)
}

func TestReadFileNamesInFlowDir(t *testing.T) {
	r := require.New(t)

	const (
		dirName  = "test_dir"
		fileName = "test_file"
	)

	path := fmt.Sprintf("%s/%s", flowDir, dirName)

	r.NoError(os.Mkdir(path, flow.AllRWX))

	_, err := os.Create(fmt.Sprintf("%s/%s", path, fileName))
	r.NoError(err)

	t.Cleanup(func() {
		os.RemoveAll(path)
	})

	repo := storage.NewFlowRepository(flowDir, subnetDisabledDir)

	data, err := repo.ReadFileNamesInFlowDir(dirName)
	r.NoError(err)
	r.Contains(data, fileName)
}

func TestMoveFlowToTempDir(t *testing.T) {
	r := require.New(t)

	const (
		dirName  = "test_dir"
		fileName = "test_file"
	)

	path := fmt.Sprintf("%s/%s", flowDir, dirName)

	r.NoError(os.Mkdir(path, flow.AllRWX))

	_, err := os.Create(fmt.Sprintf("%s/%s", path, fileName))
	r.NoError(err)

	t.Cleanup(func() {
		os.RemoveAll(path)
	})

	repo := storage.NewFlowRepository(flowDir, subnetDisabledDir)

	t.Run("перемещение в ./tmp", func(t *testing.T) {
		r.NoError(repo.MoveFlowToTempDir(dirName, fileName))

		t.Run("проверка файла", func(_ *testing.T) {
			data, errRead := repo.ReadFileNamesInFlowDir(fmt.Sprintf("%s/%s", dirName, flow.FlowTempDir))
			r.NoError(errRead)
			r.Contains(data, fileName)
		})
	})
}

// TestReadFlow чтение flow из tmp с пропуском уже закоммиченных файлов
func TestReadFlow(t *testing.T) {
	const (
		dirName   = "test_dir"
		fileName1 = "ft-01.01.2026-00:00:00"
		fileName2 = "ft-01.01.2026-00:05:00"
		fileData1 = "#:doctets,srcaddr,dstaddr\n4123,127.0.0.1,127.0.0.2\n"
		fileData2 = "#:doctets,srcaddr,dstaddr\n5670,127.0.0.1,127.0.0.3\n"
	)

	tests := []struct {
		name             string
		skipFileNames    map[string]bool
		wantOutput       string
		wantFileNameList []string
	}{
		{
			name:             "все файлы новые",
			skipFileNames:    nil,
			wantOutput:       fileData1 + fileData2,
			wantFileNameList: []string{fileName1, fileName2},
		},
		{
			name:             "первый файл уже закоммичен",
			skipFileNames:    map[string]bool{fileName1: true},
			wantOutput:       fileData2,
			wantFileNameList: []string{fileName1, fileName2},
		},
		{
			name:             "все файлы уже закоммичены",
			skipFileNames:    map[string]bool{fileName1: true, fileName2: true},
			wantOutput:       "",
			wantFileNameList: []string{fileName1, fileName2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)

			path := fmt.Sprintf("%s/%s", flowDir, dirName)
			r.NoError(os.Mkdir(path, flow.AllRWX))

			t.Cleanup(func() {
				os.RemoveAll(path)
			})

			tmpPath := fmt.Sprintf("%s/%s", path, flow.FlowTempDir)
			r.NoError(os.Mkdir(tmpPath, flow.AllRWX))

			r.NoError(os.WriteFile(fmt.Sprintf("%s/%s", tmpPath, fileName1),
				[]byte(fileData1), flow.AllRWX))
			r.NoError(os.WriteFile(fmt.Sprintf("%s/%s", tmpPath, fileName2),
				[]byte(fileData2), flow.AllRWX))
			// служебный файл git не является flow: ни в output, ни в fileNameList он попадать не должен
			r.NoError(os.WriteFile(fmt.Sprintf("%s/%s", tmpPath, flow.GitKeepName),
				[]byte{}, flow.AllRWX))

			repo := storage.NewFlowRepository(flowDir, subnetDisabledDir)

			output, fileNameList, err := repo.ReadFlow(dirName, tt.skipFileNames)
			r.NoError(err)
			r.Equal(tt.wantOutput, output)
			r.Equal(tt.wantFileNameList, fileNameList)
		})
	}
}

func TestRemoveOld(t *testing.T) {
	r := require.New(t)

	const (
		dirName  = "test_dir"
		fileName = "test_file"
	)

	path := fmt.Sprintf("%s/%s", flowDir, dirName)

	r.NoError(os.Mkdir(path, flow.AllRWX))

	_, err := os.Create(fmt.Sprintf("%s/%s", path, fileName))
	r.NoError(err)

	t.Cleanup(func() {
		os.RemoveAll(path)
	})

	repo := storage.NewFlowRepository(flowDir, subnetDisabledDir)

	r.NoError(repo.RemoveOld(dirName))
	data, err := os.ReadDir(path)
	r.NoError(err)
	r.NotContains(data, fileName)
}
