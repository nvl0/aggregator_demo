package flowgen_test

import (
	"aggregator/src/tools/flowgen"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestGenerate flow создается в переданной директории и содержит переданный ip клиента
func TestGenerate(t *testing.T) {
	tests := []struct {
		name     string
		nasIP    string
		clientIP string
	}{
		{name: "директория dev-режима", nasIP: "127.0.0.0", clientIP: "127.0.0.1"},
		{name: "синтетическая директория", nasIP: "10.0.5.0", clientIP: "127.0.1.6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flowDir := t.TempDir()

			download, upload, err := flowgen.Generate(flowgen.Params{
				NasIP:    tt.nasIP,
				ClientIP: tt.clientIP,
				FlowDir:  flowDir,
			})
			if err != nil {
				t.Fatalf("Generate вернул ошибку %v", err)
			}

			if download <= 0 || upload <= 0 {
				t.Fatalf("download %d upload %d, ожидались положительные значения", download, upload)
			}

			nasDir := fmt.Sprintf("%s/%s", flowDir, tt.nasIP)

			fileList, err := os.ReadDir(nasDir)
			if err != nil {
				t.Fatalf("не удалось прочитать директорию %s, ошибка %v", nasDir, err)
			}

			if len(fileList) != 1 {
				t.Fatalf("в директории %d файлов, ожидался 1", len(fileList))
			}

			if !strings.HasPrefix(fileList[0].Name(), "ft-") {
				t.Fatalf("имя файла %s, ожидался префикс ft-", fileList[0].Name())
			}

			b, err := os.ReadFile(fmt.Sprintf("%s/%s", nasDir, fileList[0].Name()))
			if err != nil {
				t.Fatalf("не удалось прочитать flow, ошибка %v", err)
			}

			content := string(b)

			if !strings.HasPrefix(content, "#:doctets,srcaddr,dstaddr\n") {
				t.Fatal("в файле отсутствует заголовок flow")
			}

			if !strings.Contains(content, tt.clientIP) {
				t.Fatalf("в файле нет ip клиента %s", tt.clientIP)
			}

			if got := strings.Count(content, "\n"); got != 101 {
				t.Fatalf("в файле %d строк, ожидалось 101 (заголовок и 100 записей)", got)
			}
		})
	}
}

// TestGenerateNoPanic регрессия на панику rand.IntN(0) при большом количестве генераций
func TestGenerateNoPanic(t *testing.T) {
	flowDir := t.TempDir()

	for i := 0; i < 500; i++ {
		if _, _, err := flowgen.Generate(flowgen.Params{
			NasIP:    "10.0.0.0",
			ClientIP: "127.0.1.1",
			FlowDir:  flowDir,
		}); err != nil {
			t.Fatalf("Generate вернул ошибку на итерации %d, ошибка %v", i, err)
		}
	}
}
