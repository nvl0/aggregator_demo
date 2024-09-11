package flow_test

import (
	"aggregator/src/bimport"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/uimport"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	testLogger = logger.NewNoFileLogger("test")
)

func TestPrepareFlow(t *testing.T) {
	r := require.New(t)

	type fields struct {
		ri rimport.TestRepositoryImports
		bi *bimport.TestBridgeImports
		ts *transaction.MockSession
	}
	type args struct {
		dirName string
	}

	const (
		dirName  = "test_dir"
		fileName = "ft-test_file"
		output   = "test_output"
	)

	tests := []struct {
		name    string
		prepare func(f *fields)
		args    args
		err     error
		data    string
	}{
		{
			name: "успешный результат",
			prepare: func(f *fields) {
				fileNameListInDir := []string{fileName}

				gomock.InOrder(
					f.ri.MockRepository.Flow.EXPECT().ReadFileNamesInFlowDir(dirName).Return(fileNameListInDir, nil),
					f.ri.MockRepository.Flow.EXPECT().MoveFlowToTempDir(dirName, fileName).Return(nil),
					f.ri.MockRepository.Flow.EXPECT().ReadFlow(dirName).Return(output, nil),
				)
			},
			args: args{
				dirName: dirName,
			},
			err:  nil,
			data: output,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				ri: rimport.NewTestRepositoryImports(ctrl),
				ts: transaction.NewMockSession(ctrl),
				bi: bimport.NewTestBridgeImports(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), f.bi.BridgeImports())

			data, err := ui.Usecase.Flow.PrepareFlow(tt.args.dirName)
			r.Equal(tt.err, err)
			r.Equal(tt.data, data)
		})
	}
}
