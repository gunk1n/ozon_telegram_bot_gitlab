// Code generated by MockGen. DO NOT EDIT.
// Source: internal/textrouter/texthandler/add_expense.go

// Package mock_texthandler is a generated GoMock package.
package mock_texthandler

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	usecase "gitlab.ozon.dev/myasnikov.alexander.s/telegram-bot/internal/usecase"
)

// MockIExpenseUsecaseAE is a mock of IExpenseUsecaseAE interface.
type MockIExpenseUsecaseAE struct {
	ctrl     *gomock.Controller
	recorder *MockIExpenseUsecaseAEMockRecorder
}

// MockIExpenseUsecaseAEMockRecorder is the mock recorder for MockIExpenseUsecaseAE.
type MockIExpenseUsecaseAEMockRecorder struct {
	mock *MockIExpenseUsecaseAE
}

// NewMockIExpenseUsecaseAE creates a new mock instance.
func NewMockIExpenseUsecaseAE(ctrl *gomock.Controller) *MockIExpenseUsecaseAE {
	mock := &MockIExpenseUsecaseAE{ctrl: ctrl}
	mock.recorder = &MockIExpenseUsecaseAEMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIExpenseUsecaseAE) EXPECT() *MockIExpenseUsecaseAEMockRecorder {
	return m.recorder
}

// AddExpense mocks base method.
func (m *MockIExpenseUsecaseAE) AddExpense(ctx context.Context, req usecase.AddExpenseReqDTO) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddExpense", ctx, req)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddExpense indicates an expected call of AddExpense.
func (mr *MockIExpenseUsecaseAEMockRecorder) AddExpense(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddExpense", reflect.TypeOf((*MockIExpenseUsecaseAE)(nil).AddExpense), ctx, req)
}