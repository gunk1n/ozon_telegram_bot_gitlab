// Code generated by MockGen. DO NOT EDIT.
// Source: internal/textrouter/texthandler/set_default_currency.go

// Package mock_texthandler is a generated GoMock package.
package mock_texthandler

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	usecase "gitlab.ozon.dev/myasnikov.alexander.s/telegram-bot/internal/usecase"
)

// MockIExpenseUsecaseSDC is a mock of IExpenseUsecaseSDC interface.
type MockIExpenseUsecaseSDC struct {
	ctrl     *gomock.Controller
	recorder *MockIExpenseUsecaseSDCMockRecorder
}

// MockIExpenseUsecaseSDCMockRecorder is the mock recorder for MockIExpenseUsecaseSDC.
type MockIExpenseUsecaseSDCMockRecorder struct {
	mock *MockIExpenseUsecaseSDC
}

// NewMockIExpenseUsecaseSDC creates a new mock instance.
func NewMockIExpenseUsecaseSDC(ctrl *gomock.Controller) *MockIExpenseUsecaseSDC {
	mock := &MockIExpenseUsecaseSDC{ctrl: ctrl}
	mock.recorder = &MockIExpenseUsecaseSDCMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIExpenseUsecaseSDC) EXPECT() *MockIExpenseUsecaseSDCMockRecorder {
	return m.recorder
}

// SetDefaultCurrency mocks base method.
func (m *MockIExpenseUsecaseSDC) SetDefaultCurrency(arg0 context.Context, arg1 usecase.SetDefaultCurrencyReqDTO) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetDefaultCurrency", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetDefaultCurrency indicates an expected call of SetDefaultCurrency.
func (mr *MockIExpenseUsecaseSDCMockRecorder) SetDefaultCurrency(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDefaultCurrency", reflect.TypeOf((*MockIExpenseUsecaseSDC)(nil).SetDefaultCurrency), arg0, arg1)
}