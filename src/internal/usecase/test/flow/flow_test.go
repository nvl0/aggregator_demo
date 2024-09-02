package flow_test

import (
	"aggregator/src/bimport"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/uimport"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	testLogger = logger.NewNoFileLogger("test")
)

func TestPrepareFlow(t *testing.T) {
	r := assert.New(t)

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
			name: "успешный результат с перемещением flow",
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
		{
			name: "неуспешный результат с перемещением flow",
			prepare: func(f *fields) {
				fileNameListInDir := []string{fileName}

				gomock.InOrder(
					f.ri.MockRepository.Flow.EXPECT().ReadFileNamesInFlowDir(dirName).Return(fileNameListInDir, nil),
					f.ri.MockRepository.Flow.EXPECT().MoveFlowToTempDir(dirName, fileName).Return(global.ErrInternalError),
				)
			},
			args: args{
				dirName: dirName,
			},
			err:  global.ErrInternalError,
			data: "",
		},
		{
			name: "неуспешный результат",
			prepare: func(f *fields) {
				gomock.InOrder(
					f.ri.MockRepository.Flow.EXPECT().ReadFileNamesInFlowDir(dirName).Return(nil, global.ErrNoData),
				)
			},
			args: args{
				dirName: dirName,
			},
			err:  global.ErrNoData,
			data: "",
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

			sm := transaction.NewMockSessionManager(ctrl)
			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), f.bi.BridgeImports(), sm)

			data, err := ui.Usecase.Flow.PrepareFlow(tt.args.dirName)
			r.Equal(tt.err, err)
			r.Equal(tt.data, data)
		})
	}
}
