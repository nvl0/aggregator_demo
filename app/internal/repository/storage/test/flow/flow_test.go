package flow_test

import (
	"aggregator/app/internal/entity/flow"
	"aggregator/app/internal/repository/storage"

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

	r.NoError(os.Mkdir(path, flow.UserRWE))

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

	r.NoError(os.Mkdir(path, flow.UserRWE))

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

	r.NoError(os.Mkdir(path, flow.UserRWE))

	_, err := os.Create(fmt.Sprintf("%s/%s", path, fileName))
	r.NoError(err)

	t.Cleanup(func() {
		os.RemoveAll(path)
	})

	repo := storage.NewFlowRepository(flowDir, subnetDisabledDir)

	t.Run("перемещение в ./tmp", func(t *testing.T) {
		r.NoError(repo.MoveFlowToTempDir(dirName, fileName))

		t.Run("проверка файла", func(t *testing.T) {
			data, err := repo.ReadFileNamesInFlowDir(fmt.Sprintf("%s/%s", dirName, flow.FlowTempDir))
			r.NoError(err)
			r.Contains(data, fileName)
		})
	})
}

func TestReadFlow(t *testing.T) {
	r := require.New(t)

	const (
		dirName  = "test_dir"
		fileName = "test_file"
		fileData = `#:doctets,srcaddr,dstaddr
4123,127.0.0.1,127.0.0.2`
	)

	path := fmt.Sprintf("%s/%s", flowDir, dirName)

	r.NoError(os.Mkdir(path, flow.UserRWE))

	tmpPath := fmt.Sprintf("%s/%s", path, flow.FlowTempDir)

	r.NoError(os.Mkdir(tmpPath, flow.UserRWE))

	fileDisabled, err := os.Create(fmt.Sprintf("%s/%s", tmpPath, fileName))
	r.NoError(err)

	_, err = fileDisabled.WriteString(fileData)
	r.NoError(err)

	t.Cleanup(func() {
		os.RemoveAll(path)
	})

	repo := storage.NewFlowRepository(flowDir, subnetDisabledDir)

	t.Run("чтение flow", func(t *testing.T) {
		data, err := repo.ReadFlow(dirName)
		r.NoError(err)
		r.Equal(fileData, data)
	})
}
