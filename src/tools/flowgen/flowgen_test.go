package flowgen_test

import (
	"aggregator/src/tools/flowgen"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateResult аргументы и результат одного вызова flowgen.Generate,
// необходимые для проверки созданного flow-файла
type generateResult struct {
	flowDir  string
	nasIP    string
	clientIP string
	download int
	upload   int
}

// assertGeneratedFlow проверяет, что flow создан в переданной директории,
// содержит корректный заголовок и переданный ip клиента
func assertGeneratedFlow(t *testing.T, r generateResult) {
	t.Helper()

	assert.Positive(t, r.download)
	assert.Positive(t, r.upload)

	nasDir := fmt.Sprintf("%s/%s", r.flowDir, r.nasIP)

	fileList, err := os.ReadDir(nasDir)
	require.NoError(t, err)
	require.Len(t, fileList, 1)

	assert.True(t, strings.HasPrefix(fileList[0].Name(), "ft-"),
		"имя файла %s, ожидался префикс ft-", fileList[0].Name())

	b, err := os.ReadFile(fmt.Sprintf("%s/%s", nasDir, fileList[0].Name()))
	require.NoError(t, err)

	content := string(b)

	assert.True(t, strings.HasPrefix(content, "#:doctets,srcaddr,dstaddr\n"),
		"в файле отсутствует заголовок flow")
	assert.Contains(t, content, r.clientIP, "в файле нет ip клиента")
	assert.Equal(t, 101, strings.Count(content, "\n"),
		"ожидалось 100 записей и заголовок")
}

// TestGenerate flow создается в переданной директории и содержит переданный ip клиента
func TestGenerate(t *testing.T) {
	tests := []struct {
		name       string
		nasIP      string
		clientIP   string
		assertFunc func(t *testing.T, r generateResult)
	}{
		{name: "директория dev-режима", nasIP: "127.0.0.0", clientIP: "127.0.0.1", assertFunc: assertGeneratedFlow},
		{name: "синтетическая директория", nasIP: "10.0.5.0", clientIP: "127.0.1.6", assertFunc: assertGeneratedFlow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flowDir := t.TempDir()

			download, upload, err := flowgen.Generate(flowgen.Params{
				NasIP:    tt.nasIP,
				ClientIP: tt.clientIP,
				FlowDir:  flowDir,
			})
			require.NoError(t, err)

			tt.assertFunc(t, generateResult{
				flowDir:  flowDir,
				nasIP:    tt.nasIP,
				clientIP: tt.clientIP,
				download: download,
				upload:   upload,
			})
		})
	}
}

// TestGenerateNoPanic регрессия на панику rand.IntN(0) при большом количестве генераций
func TestGenerateNoPanic(t *testing.T) {
	flowDir := t.TempDir()

	for i := range 500 {
		_, _, err := flowgen.Generate(flowgen.Params{
			NasIP:    "10.0.0.0",
			ClientIP: "127.0.1.1",
			FlowDir:  flowDir,
		})
		require.NoError(t, err, "Generate вернул ошибку на итерации %d", i)
	}
}
