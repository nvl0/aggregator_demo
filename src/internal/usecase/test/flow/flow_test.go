package flow_test

import (
	"testing"

	"aggregator/src/bimport"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/uimport"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	testLogger = logger.NewNoFileLogger("test")
)

func TestPrepareFlow(t *testing.T) {
	type fields struct {
		ri rimport.TestRepositoryImports
		bi *bimport.TestBridgeImports
		ts *transaction.MockSession
	}
	type args struct {
		dirName       string
		skipFileNames map[string]bool
	}

	const (
		dirName  = "test_dir"
		fileName = "ft-test_file"
		oldFile  = "ft-old_file"
		output   = "test_output"
	)

	tests := []struct {
		name             string
		prepare          func(f *fields)
		args             args
		err              error
		data             string
		wantFileNameList []string
	}{
		{
			name: "успешный результат",
			prepare: func(f *fields) {
				fileNameListInDir := []string{fileName}

				gomock.InOrder(
					f.ri.MockRepository.Flow.EXPECT().
						ReadFileNamesInFlowDir(dirName).Return(fileNameListInDir, nil),
					f.ri.MockRepository.Flow.EXPECT().
						MoveFlowToTempDir(dirName, fileName).Return(nil),
					f.ri.MockRepository.Flow.EXPECT().
						ReadFlow(dirName, map[string]bool(nil)).
						Return(output, []string{fileName}, nil),
				)
			},
			args: args{
				dirName: dirName,
			},
			err:              nil,
			data:             output,
			wantFileNameList: []string{fileName},
		},
		{
			name: "закоммиченный файл пропускается, но остается в списке",
			prepare: func(f *fields) {
				fileNameListInDir := []string{fileName}
				skipFileNames := map[string]bool{oldFile: true}

				gomock.InOrder(
					f.ri.MockRepository.Flow.EXPECT().
						ReadFileNamesInFlowDir(dirName).Return(fileNameListInDir, nil),
					f.ri.MockRepository.Flow.EXPECT().
						MoveFlowToTempDir(dirName, fileName).Return(nil),
					f.ri.MockRepository.Flow.EXPECT().
						ReadFlow(dirName, skipFileNames).
						Return(output, []string{oldFile, fileName}, nil),
				)
			},
			args: args{
				dirName:       dirName,
				skipFileNames: map[string]bool{oldFile: true},
			},
			err:              nil,
			data:             output,
			wantFileNameList: []string{oldFile, fileName},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)

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

			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), f.bi.BridgeImports(), nil)

			data, fileNameList, err := ui.Usecase.Flow.PrepareFlow(tt.args.dirName, tt.args.skipFileNames)
			r.Equal(tt.err, err)
			r.Equal(tt.data, data)
			r.Equal(tt.wantFileNameList, fileNameList)
		})
	}
}
