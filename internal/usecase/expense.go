package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	lrucache "gitlab.ozon.dev/myasnikov.alexander.s/telegram-bot/internal/adapter/cache/lru"
	"gitlab.ozon.dev/myasnikov.alexander.s/telegram-bot/internal/entity"
	"gitlab.ozon.dev/myasnikov.alexander.s/telegram-bot/internal/utils"
	"go.opentelemetry.io/otel"
)

type ICurrencyStorage interface {
	Get(context.Context, string) (entity.Rate, error)
	GetAll(context.Context) ([]entity.Rate, error)
	Update(context.Context, entity.Rate) error
}

type IUserStorage interface {
	GetDefaultCurrency(context.Context, entity.UserID) (string, error)
	UpdateDefaultCurrency(context.Context, entity.UserID, string) error
	GetLimits(context.Context, entity.UserID) (decimal.Decimal, decimal.Decimal, decimal.Decimal, error)
	UpdateDayLimit(context.Context, entity.UserID, decimal.Decimal) error
	UpdateWeekLimit(context.Context, entity.UserID, decimal.Decimal) error
	UpdateMonthLimit(context.Context, entity.UserID, decimal.Decimal) error
}

type IExpenseStorage interface {
	Create(context.Context, entity.UserID, entity.Expense) error
	Get(context.Context, entity.UserID, time.Time, time.Time) ([]entity.Expense, error)
}

type IRatesUpdaterService interface {
	Get(ctx context.Context, base string, codes []string) ([]entity.Rate, error)
}

type GetReportClient interface {
	GetReport(ctx context.Context, req GetReportReqDTO) (GetReportRespDTO, error)
}

type IConfig interface {
	GetBaseCurrencyCode() string
	GetCurrencyCodes() []string
	GetFrequencyRateUpdateSec() int
	GetReportCacheEnable() bool
	GetReportCacheSize() int
	GetReportCacheTTL() int
}

type ExpenseUsecase struct {
	currencyStorage     ICurrencyStorage
	userStorage         IUserStorage
	expenseStorage      IExpenseStorage
	ratesUpdaterService IRatesUpdaterService
	getReportClient     GetReportClient
	config              IConfig
	cache               *lrucache.LRUCache
}

func NewExpenseUsecase(currencyStorage ICurrencyStorage, userStorage IUserStorage, expenseStorage IExpenseStorage,
	ratesUpdaterService IRatesUpdaterService, getReportClient GetReportClient, config IConfig,
) *ExpenseUsecase {
	var cache *lrucache.LRUCache
	if config.GetReportCacheEnable() {
		cache = lrucache.NewLRUCache(&sync.RWMutex{}, config.GetReportCacheSize())
	}

	return &ExpenseUsecase{
		currencyStorage:     currencyStorage,
		userStorage:         userStorage,
		expenseStorage:      expenseStorage,
		ratesUpdaterService: ratesUpdaterService,
		getReportClient:     getReportClient,
		config:              config,
		cache:               cache,
	}
}

func (uc *ExpenseUsecase) SetDefaultCurrency(ctx context.Context, req SetDefaultCurrencyReqDTO,
) (SetDefaultCurrencyRespDTO, error) {
	ctx, span := otel.Tracer("ExpenseUsecase").Start(ctx, "SetDefaultCurrency")
	defer span.End()

	userID := entity.UserID(req.UserID)

	ok := uc.isSupportedCurrencyCode(req.Currency)
	if !ok {
		return SetDefaultCurrencyRespDTO{}, errors.New("currency is unsupported")
	}

	err := uc.userStorage.UpdateDefaultCurrency(ctx, userID, req.Currency)

	return SetDefaultCurrencyRespDTO{}, errors.Wrap(err, "ExpenseUsecase.SetDefaultCurrency")
}

func (uc *ExpenseUsecase) SetLimit(ctx context.Context, req SetLimitReqDTO) (SetLimitRespDTO, error) {
	ctx, span := otel.Tracer("ExpenseUsecase").Start(ctx, "SetLimit")
	defer span.End()

	userID := entity.UserID(req.UserID)

	currency := uc.getCurrencyForUser(ctx, userID)

	rate, err := uc.currencyStorage.Get(ctx, currency)
	if err != nil {
		return SetLimitRespDTO{}, errors.Wrap(err, "ExpenseUsecase.SetLimit")
	}

	req.Limit = req.Limit.Div(rate.GetRatio())

	switch req.IntervalType {
	case utils.DayInterval:
		err = uc.userStorage.UpdateDayLimit(ctx, userID, req.Limit)
	case utils.WeekInterval:
		err = uc.userStorage.UpdateWeekLimit(ctx, userID, req.Limit)
	case utils.MonthInterval:
		err = uc.userStorage.UpdateMonthLimit(ctx, userID, req.Limit)
	default:
		return SetLimitRespDTO{}, errors.New("unknown intervalType")
	}

	resp := SetLimitRespDTO{
		Currency: currency,
	}

	return resp, errors.Wrap(err, "ExpenseUsecase.SetDefaultCurrency")
}

func (uc *ExpenseUsecase) GetLimits(ctx context.Context, req GetLimitsReqDTO) (GetLimitsRespDTO, error) {
	ctx, span := otel.Tracer("ExpenseUsecase").Start(ctx, "GetLimits")
	defer span.End()

	userID := entity.UserID(req.UserID)

	dayLimit, weekLimit, monthLimit, err := uc.userStorage.GetLimits(ctx, userID)
	if err != nil {
		return GetLimitsRespDTO{}, errors.Wrap(err, "ExpenseUsecase.GetLimits")
	}

	currency := uc.getCurrencyForUser(ctx, userID)

	rate, err := uc.currencyStorage.Get(ctx, currency)
	if err != nil {
		return GetLimitsRespDTO{}, errors.Wrap(err, "ExpenseUsecase.GetLimits")
	}

	resp := GetLimitsRespDTO{
		Currency: currency,
		Limits: map[int]decimal.Decimal{
			utils.DayInterval:   dayLimit.Mul(rate.GetRatio()),
			utils.WeekInterval:  weekLimit.Mul(rate.GetRatio()),
			utils.MonthInterval: monthLimit.Mul(rate.GetRatio()),
		},
	}

	return resp, errors.Wrap(err, "ExpenseUsecase.GetLimits")
}

func (uc *ExpenseUsecase) AddExpense(ctx context.Context, req AddExpenseReqDTO) (AddExpenseRespDTO, error) {
	ctx, span := otel.Tracer("ExpenseUsecase").Start(ctx, "AddExpense")
	defer span.End()

	uc.deleteReportFromCache(req)

	userID := entity.UserID(req.UserID)

	err := uc.tryUpdateRates(ctx, false)
	if err != nil {
		return AddExpenseRespDTO{}, errors.Wrap(err, "ExpenseUsecase.AddExpense")
	}

	currency := uc.getCurrencyForUser(ctx, userID)

	rate, err := uc.currencyStorage.Get(ctx, currency)
	if err != nil {
		return AddExpenseRespDTO{}, errors.Wrap(err, "ExpenseUsecase.AddExpense")
	}

	expense := entity.NewExpense(req.Category, req.Price.Div(rate.GetRatio()), req.Date)

	err = uc.expenseStorage.Create(ctx, userID, expense)
	if err != nil {
		return AddExpenseRespDTO{}, errors.Wrap(err, "ExpenseUsecase.AddExpense")
	}

	limits, err := uc.checkLimits(ctx, userID, req.Date)

	for i := range limits {
		limits[i] = limits[i].Mul(rate.GetRatio())
	}

	resp := AddExpenseRespDTO{
		Currency: currency,
		Limits:   limits,
	}

	return resp, errors.Wrap(err, "ExpenseUsecase.AddExpense")
}

func (uc *ExpenseUsecase) checkLimits(ctx context.Context, userID entity.UserID, date time.Time,
) (map[int]decimal.Decimal, error) {
	limits := make(map[int]decimal.Decimal, 1+1+1)

	dayLimit, weekLimit, monthLimit, err := uc.userStorage.GetLimits(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "ExpenseUsecase.checkLimits")
	}

	checkLimit := func(intervalType int, limit decimal.Decimal) error {
		if limit.LessThanOrEqual(decimal.Zero) {
			return nil
		}

		dateStart, dateEnd := utils.GetInterval(date, intervalType)

		expenses, err := uc.expenseStorage.Get(ctx, userID, dateStart, dateEnd)
		if err != nil {
			return errors.Wrap(err, "ExpenseUsecase.AddExpense")
		}

		for _, expense := range expenses {
			limit = limit.Sub(expense.GetPrice())
		}

		limits[intervalType] = limit

		return nil
	}

	err = checkLimit(utils.DayInterval, dayLimit)
	if err != nil {
		return nil, errors.Wrap(err, "ExpenseUsecase.AddExpense")
	}

	err = checkLimit(utils.WeekInterval, weekLimit)
	if err != nil {
		return nil, errors.Wrap(err, "ExpenseUsecase.AddExpense")
	}

	err = checkLimit(utils.MonthInterval, monthLimit)
	if err != nil {
		return nil, errors.Wrap(err, "ExpenseUsecase.AddExpense")
	}

	return limits, errors.Wrap(err, "ExpenseUsecase.AddExpense")
}

func (uc *ExpenseUsecase) GetReport(ctx context.Context, req GetReportReqDTO) (GetReportRespDTO, error) {
	ctx, span := otel.Tracer("ExpenseUsecase").Start(ctx, "GetReport")
	defer span.End()

	if report, ok := uc.getReportFromCache(req); ok {
		return report, nil
	}

	resp, err := uc.getReportClient.GetReport(ctx, req)

	uc.addReportToCache(req, resp)

	return resp, errors.Wrap(err, "ExpenseUsecase.GetReport")
}

func (uc *ExpenseUsecase) UpdateCurrency(ctx context.Context) error {
	err := uc.tryUpdateRates(ctx, true)

	return errors.Wrap(err, "ExpenseUsecase.UpdateCurrency")
}

// ---- helpers

func (uc *ExpenseUsecase) tryUpdateRates(ctx context.Context, force bool) error {
	if !force && !uc.needUpdateRates(ctx) {
		return nil
	}

	rates, err := uc.ratesUpdaterService.Get(ctx, uc.config.GetBaseCurrencyCode(), uc.config.GetCurrencyCodes())
	if err != nil {
		return errors.Wrap(err, "ExpenseUsecase.tryUpdateRates")
	}

	for _, rate := range rates {
		err := uc.currencyStorage.Update(ctx, rate)
		if err != nil {
			return errors.Wrap(err, "ExpenseUsecase.tryUpdateRates")
		}
	}

	return nil
}

func (uc *ExpenseUsecase) needUpdateRates(ctx context.Context) bool {
	rate, err := uc.currencyStorage.Get(ctx, uc.config.GetBaseCurrencyCode())
	if err != nil {
		return true
	}

	duration := time.Since(rate.GetTime()).Seconds()

	return duration > float64(uc.config.GetFrequencyRateUpdateSec())
}

func (uc *ExpenseUsecase) getCurrencyForUser(ctx context.Context, userID entity.UserID) string {
	currency, err := uc.userStorage.GetDefaultCurrency(ctx, userID)
	if err == nil {
		// Используется валюта дефолтная для пользователя
		return currency
	}

	// Используется валюта дефолтная для всех
	return uc.config.GetBaseCurrencyCode()
}

func (uc *ExpenseUsecase) isSupportedCurrencyCode(currency string) bool {
	if currency == uc.config.GetBaseCurrencyCode() {
		return true
	}

	for _, supportedCode := range uc.config.GetCurrencyCodes() {
		if currency == supportedCode {
			return true
		}
	}

	return false
}

func (uc *ExpenseUsecase) addReportToCache(req GetReportReqDTO, resp GetReportRespDTO) {
	if !uc.config.GetReportCacheEnable() {
		return
	}

	key := fmt.Sprintf("%d_%d_%d", req.UserID, utils.TruncDate(req.Date).Unix(), req.IntervalType)

	uc.cache.Add(time.Now(), key, resp, uc.config.GetReportCacheTTL())
}

func (uc *ExpenseUsecase) getReportFromCache(req GetReportReqDTO) (GetReportRespDTO, bool) {
	var resp GetReportRespDTO

	if !uc.config.GetReportCacheEnable() {
		return resp, false
	}

	key := fmt.Sprintf("%d_%d_%d", req.UserID, utils.TruncDate(req.Date).Unix(), req.IntervalType)

	val, ok := uc.cache.Get(time.Now(), key)
	if !ok {
		return resp, false
	}

	resp, ok = val.(GetReportRespDTO)
	if !ok {
		return resp, false
	}

	return resp, ok
}

func (uc *ExpenseUsecase) deleteReportFromCache(req AddExpenseReqDTO) {
	if !uc.config.GetReportCacheEnable() {
		return
	}

	uc.cache.Delete(time.Now(), fmt.Sprintf("%d_%d_%d",
		req.UserID, utils.TruncDate(req.Date).Unix(), utils.DayInterval))

	uc.cache.Delete(time.Now(), fmt.Sprintf("%d_%d_%d",
		req.UserID, utils.TruncDate(req.Date).Unix(), utils.WeekInterval))

	uc.cache.Delete(time.Now(), fmt.Sprintf("%d_%d_%d",
		req.UserID, utils.TruncDate(req.Date).Unix(), utils.MonthInterval))
}
