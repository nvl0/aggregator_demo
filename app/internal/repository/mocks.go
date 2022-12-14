// Code generated by MockGen. DO NOT EDIT.
// Source: ../internal/repository/interface.go

// Package repository is a generated GoMock package.
package repository

import (
	session "aggregator/app/internal/entity/session"
	transaction "aggregator/app/internal/transaction"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockSession is a mock of Session interface.
type MockSession struct {
	ctrl     *gomock.Controller
	recorder *MockSessionMockRecorder
}

// MockSessionMockRecorder is the mock recorder for MockSession.
type MockSessionMockRecorder struct {
	mock *MockSession
}

// NewMockSession creates a new mock instance.
func NewMockSession(ctrl *gomock.Controller) *MockSession {
	mock := &MockSession{ctrl: ctrl}
	mock.recorder = &MockSessionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSession) EXPECT() *MockSessionMockRecorder {
	return m.recorder
}

// LoadOnlineSessionList mocks base method.
func (m *MockSession) LoadOnlineSessionList(ts transaction.Session) ([]session.Session, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadOnlineSessionList", ts)
	ret0, _ := ret[0].([]session.Session)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LoadOnlineSessionList indicates an expected call of LoadOnlineSessionList.
func (mr *MockSessionMockRecorder) LoadOnlineSessionList(ts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadOnlineSessionList", reflect.TypeOf((*MockSession)(nil).LoadOnlineSessionList), ts)
}

// SaveChunkList mocks base method.
func (m *MockSession) SaveChunkList(ts transaction.Session, chunkList []session.Chunk) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveChunkList", ts, chunkList)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveChunkList indicates an expected call of SaveChunkList.
func (mr *MockSessionMockRecorder) SaveChunkList(ts, chunkList interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveChunkList", reflect.TypeOf((*MockSession)(nil).SaveChunkList), ts, chunkList)
}

// MockFlow is a mock of Flow interface.
type MockFlow struct {
	ctrl     *gomock.Controller
	recorder *MockFlowMockRecorder
}

// MockFlowMockRecorder is the mock recorder for MockFlow.
type MockFlowMockRecorder struct {
	mock *MockFlow
}

// NewMockFlow creates a new mock instance.
func NewMockFlow(ctrl *gomock.Controller) *MockFlow {
	mock := &MockFlow{ctrl: ctrl}
	mock.recorder = &MockFlowMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFlow) EXPECT() *MockFlowMockRecorder {
	return m.recorder
}

// MoveFlowToTempDir mocks base method.
func (m *MockFlow) MoveFlowToTempDir(dirName, fileName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MoveFlowToTempDir", dirName, fileName)
	ret0, _ := ret[0].(error)
	return ret0
}

// MoveFlowToTempDir indicates an expected call of MoveFlowToTempDir.
func (mr *MockFlowMockRecorder) MoveFlowToTempDir(dirName, fileName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MoveFlowToTempDir", reflect.TypeOf((*MockFlow)(nil).MoveFlowToTempDir), dirName, fileName)
}

// ReadFileNamesInFlowDir mocks base method.
func (m *MockFlow) ReadFileNamesInFlowDir(dirName string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadFileNamesInFlowDir", dirName)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadFileNamesInFlowDir indicates an expected call of ReadFileNamesInFlowDir.
func (mr *MockFlowMockRecorder) ReadFileNamesInFlowDir(dirName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFileNamesInFlowDir", reflect.TypeOf((*MockFlow)(nil).ReadFileNamesInFlowDir), dirName)
}

// ReadFlow mocks base method.
func (m *MockFlow) ReadFlow(dirName string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadFlow", dirName)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadFlow indicates an expected call of ReadFlow.
func (mr *MockFlowMockRecorder) ReadFlow(dirName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFlow", reflect.TypeOf((*MockFlow)(nil).ReadFlow), dirName)
}

// ReadFlowDirNames mocks base method.
func (m *MockFlow) ReadFlowDirNames() ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadFlowDirNames")
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadFlowDirNames indicates an expected call of ReadFlowDirNames.
func (mr *MockFlowMockRecorder) ReadFlowDirNames() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFlowDirNames", reflect.TypeOf((*MockFlow)(nil).ReadFlowDirNames))
}
