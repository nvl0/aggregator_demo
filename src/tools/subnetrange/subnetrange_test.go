package subnetrange_test

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"aggregator/src/tools/subnetrange"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yl2chen/cidranger"
)

// TestCreateDisabledSubnetRange проверяет построение ranger из файла подсетей
func TestCreateDisabledSubnetRange(t *testing.T) {
	tests := []struct {
		name       string
		fileBody   string
		wantErr    bool
		assertFunc func(t *testing.T, ranger cidranger.Ranger)
	}{
		{
			name:     "валидные подсети добавляются в ranger",
			fileBody: "192.168.1.0/24\n10.0.0.0/8\n",
			assertFunc: func(t *testing.T, ranger cidranger.Ranger) {
				assert.Equal(t, 2, ranger.Len())

				ok, err := ranger.Contains(net.ParseIP("192.168.1.42"))
				require.NoError(t, err)
				assert.True(t, ok, "192.168.1.42 должен входить в 192.168.1.0/24")

				ok, err = ranger.Contains(net.ParseIP("8.8.8.8"))
				require.NoError(t, err)
				assert.False(t, ok, "8.8.8.8 не должен входить ни в одну из подсетей")
			},
		},
		{
			name:     "пустые строки и пробелы пропускаются",
			fileBody: "192.168.1.0/24\n\n   \n10.0.0.0/8\n",
			assertFunc: func(t *testing.T, ranger cidranger.Ranger) {
				assert.Equal(t, 2, ranger.Len())
			},
		},
		{
			name:     "невалидный CIDR - ошибка",
			fileBody: "не-подсеть\n",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "subnets.txt")
			require.NoError(t, os.WriteFile(path, []byte(tt.fileBody), 0o600))

			ranger, err := subnetrange.CreateDisabledSubnetRange(path)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.assertFunc != nil {
				tt.assertFunc(t, ranger)
			}
		})
	}
}

// TestCreateDisabledSubnetRangeFileNotFound проверяет ошибку при отсутствующем файле
func TestCreateDisabledSubnetRangeFileNotFound(t *testing.T) {
	_, err := subnetrange.CreateDisabledSubnetRange(filepath.Join(t.TempDir(), "нет-такого-файла.txt"))

	require.Error(t, err)
}
