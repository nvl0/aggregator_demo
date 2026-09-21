package flow_test

import (
	"testing"

	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/uimport"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	testLogger = logger.NewDiscard()
)

func TestStreamFlow(t *testing.T) {
	type fields struct {
		ri rimport.TestRepositoryImports
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
		line1    = "4123,127.0.0.1,127.0.0.2"
		flowSize = 25
	)

	tests := []struct {
		name             string
		prepare          func(f *fields)
		args             args
		err              error
		wantLines        []string
		wantFlowSize     int
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
					// репозиторий отдает строку в callback, который пришел сверху
					f.ri.MockRepository.Flow.EXPECT().
						StreamFlow(dirName, map[string]bool(nil), gomock.Any()).
						DoAndReturn(func(_ string, _ map[string]bool,
							onLine func(line string) error) ([]string, int, error) {
							if err := onLine(line1); err != nil {
								return nil, 0, err
							}

							return []string{fileName}, flowSize, nil
						}),
				)
			},
			args: args{
				dirName: dirName,
			},
			err:              nil,
			wantLines:        []string{line1},
			wantFlowSize:     flowSize,
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
						StreamFlow(dirName, skipFileNames, gomock.Any()).
						DoAndReturn(func(_ string, _ map[string]bool,
							onLine func(line string) error) ([]string, int, error) {
							if err := onLine(line1); err != nil {
								return nil, 0, err
							}

							return []string{oldFile, fileName}, flowSize, nil
						}),
				)
			},
			args: args{
				dirName:       dirName,
				skipFileNames: map[string]bool{oldFile: true},
			},
			err:              nil,
			wantLines:        []string{line1},
			wantFlowSize:     flowSize,
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
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), nil, nil)

			var gotLines []string

			fileNameList, flowSizeGot, err := ui.Usecase.Flow.StreamFlow(tt.args.dirName,
				tt.args.skipFileNames, func(line string) error {
					gotLines = append(gotLines, line)

					return nil
				})
			r.Equal(tt.err, err)
			r.Equal(tt.wantLines, gotLines)
			r.Equal(tt.wantFlowSize, flowSizeGot)
			r.Equal(tt.wantFileNameList, fileNameList)
		})
	}
}
