// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/DoomSentinel/scheduler-api/gen/go/v1 (interfaces: SchedulerServiceServer)

// Package client is a generated GoMock package.
package client

import (
	context "context"
	scheduler "github.com/DoomSentinel/scheduler-api/gen/go/v1"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockSchedulerServiceServer is a mock of SchedulerServiceServer interface
type MockSchedulerServiceServer struct {
	ctrl     *gomock.Controller
	recorder *MockSchedulerServiceServerMockRecorder
}

// MockSchedulerServiceServerMockRecorder is the mock recorder for MockSchedulerServiceServer
type MockSchedulerServiceServerMockRecorder struct {
	mock *MockSchedulerServiceServer
}

// NewMockSchedulerServiceServer creates a new mock instance
func NewMockSchedulerServiceServer(ctrl *gomock.Controller) *MockSchedulerServiceServer {
	mock := &MockSchedulerServiceServer{ctrl: ctrl}
	mock.recorder = &MockSchedulerServiceServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSchedulerServiceServer) EXPECT() *MockSchedulerServiceServerMockRecorder {
	return m.recorder
}

// ExecutionNotifications mocks base method
func (m *MockSchedulerServiceServer) ExecutionNotifications(arg0 *scheduler.ExecutionNotificationsRequest, arg1 scheduler.SchedulerService_ExecutionNotificationsServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecutionNotifications", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// ExecutionNotifications indicates an expected call of ExecutionNotifications
func (mr *MockSchedulerServiceServerMockRecorder) ExecutionNotifications(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecutionNotifications", reflect.TypeOf((*MockSchedulerServiceServer)(nil).ExecutionNotifications), arg0, arg1)
}

// ScheduleCommandTask mocks base method
func (m *MockSchedulerServiceServer) ScheduleCommandTask(arg0 context.Context, arg1 *scheduler.ScheduleCommandTaskRequest) (*scheduler.ScheduleCommandTaskResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ScheduleCommandTask", arg0, arg1)
	ret0, _ := ret[0].(*scheduler.ScheduleCommandTaskResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ScheduleCommandTask indicates an expected call of ScheduleCommandTask
func (mr *MockSchedulerServiceServerMockRecorder) ScheduleCommandTask(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ScheduleCommandTask", reflect.TypeOf((*MockSchedulerServiceServer)(nil).ScheduleCommandTask), arg0, arg1)
}

// ScheduleDummyTask mocks base method
func (m *MockSchedulerServiceServer) ScheduleDummyTask(arg0 context.Context, arg1 *scheduler.ScheduleDummyTaskRequest) (*scheduler.ScheduleDummyTaskResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ScheduleDummyTask", arg0, arg1)
	ret0, _ := ret[0].(*scheduler.ScheduleDummyTaskResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ScheduleDummyTask indicates an expected call of ScheduleDummyTask
func (mr *MockSchedulerServiceServerMockRecorder) ScheduleDummyTask(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ScheduleDummyTask", reflect.TypeOf((*MockSchedulerServiceServer)(nil).ScheduleDummyTask), arg0, arg1)
}

// ScheduleRemoteTask mocks base method
func (m *MockSchedulerServiceServer) ScheduleRemoteTask(arg0 context.Context, arg1 *scheduler.ScheduleRemoteTaskRequest) (*scheduler.ScheduleRemoteTaskResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ScheduleRemoteTask", arg0, arg1)
	ret0, _ := ret[0].(*scheduler.ScheduleRemoteTaskResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ScheduleRemoteTask indicates an expected call of ScheduleRemoteTask
func (mr *MockSchedulerServiceServerMockRecorder) ScheduleRemoteTask(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ScheduleRemoteTask", reflect.TypeOf((*MockSchedulerServiceServer)(nil).ScheduleRemoteTask), arg0, arg1)
}