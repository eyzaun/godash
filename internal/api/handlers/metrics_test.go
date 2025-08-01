package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/eyzaun/godash/internal/models"
)

// MockMetricsRepository is a mock implementation of MetricsRepository
type MockMetricsRepository struct {
	mock.Mock
}

func (m *MockMetricsRepository) Create(metric *models.Metric) error {
	args := m.Called(metric)
	return args.Error(0)
}

func (m *MockMetricsRepository) CreateBatch(metrics []*models.Metric) error {
	args := m.Called(metrics)
	return args.Error(0)
}

func (m *MockMetricsRepository) GetByID(id uint) (*models.Metric, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Metric), args.Error(1)
}

func (m *MockMetricsRepository) GetLatest() (*models.Metric, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Metric), args.Error(1)
}

func (m *MockMetricsRepository) GetLatestByHostname(hostname string) (*models.Metric, error) {
	args := m.Called(hostname)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Metric), args.Error(1)
}

func (m *MockMetricsRepository) GetHistory(from, to time.Time, limit, offset int) ([]*models.Metric, error) {
	args := m.Called(from, to, limit, offset)
	return args.Get(0).([]*models.Metric), args.Error(1)
}

func (m *MockMetricsRepository) GetHistoryByHostname(hostname string, from, to time.Time, limit, offset int) ([]*models.Metric, error) {
	args := m.Called(hostname, from, to, limit, offset)
	return args.Get(0).([]*models.Metric), args.Error(1)
}

func (m *MockMetricsRepository) GetAverageUsage(duration time.Duration) (*models.AverageMetrics, error) {
	args := m.Called(duration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AverageMetrics), args.Error(1)
}

func (m *MockMetricsRepository) GetAverageUsageByHostname(hostname string, duration time.Duration) (*models.AverageMetrics, error) {
	args := m.Called(hostname, duration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AverageMetrics), args.Error(1)
}

func (m *MockMetricsRepository) GetMetricsSummary(from, to time.Time) (*models.MetricsSummary, error) {
	args := m.Called(from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MetricsSummary), args.Error(1)
}

func (m *MockMetricsRepository) GetTopHostsByUsage(metricType string, limit int) ([]*models.HostUsage, error) {
	args := m.Called(metricType, limit)
	return args.Get(0).([]*models.HostUsage), args.Error(1)
}

func (m *MockMetricsRepository) GetUsageTrends(hostname string, hours int) ([]*models.UsageTrend, error) {
	args := m.Called(hostname, hours)
	return args.Get(0).([]*models.UsageTrend), args.Error(1)
}

func (m *MockMetricsRepository) DeleteOldRecords(olderThan time.Time) (int64, error) {
	args := m.Called(olderThan)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMetricsRepository) GetTotalCount() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMetricsRepository) GetCountByDateRange(from, to time.Time) (int64, error) {
	args := m.Called(from, to)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMetricsRepository) GetSystemStatus() ([]*models.SystemStatus, error) {
	args := m.Called()
	return args.Get(0).([]*models.SystemStatus), args.Error(1)
}

// Test helper functions

func setupTestRouter() (*gin.Engine, *MockMetricsRepository) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockMetricsRepository)
	handler := NewMetricsHandler(mockRepo)

	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.GET("/metrics/current", handler.GetCurrentMetrics)
		v1.GET("/metrics/current/:hostname", handler.GetCurrentMetricsByHostname)
		v1.GET("/metrics/history", handler.GetMetricsHistory)
		v1.GET("/metrics/history/:hostname", handler.GetMetricsHistoryByHostname)
		v1.GET("/metrics/average", handler.GetAverageMetrics)
		v1.GET("/metrics/average/:hostname", handler.GetAverageMetricsByHostname)
		v1.GET("/metrics/summary", handler.GetMetricsSummary)
		v1.GET("/metrics/trends/:hostname", handler.GetUsageTrends)
		v1.GET("/metrics/top/:type", handler.GetTopHostsByUsage)
		v1.POST("/metrics", handler.CreateMetric)
		v1.GET("/system/status", handler.GetSystemStatus)
		v1.GET("/system/hosts", handler.GetHosts)
		v1.GET("/system/stats", handler.GetStats)
		v1.DELETE("/admin/metrics/cleanup", handler.CleanupOldMetrics)
	}

	return router, mockRepo
}

func createSampleMetric() *models.Metric {
	return &models.Metric{
		BaseModel: models.BaseModel{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Hostname:      "test-host",
		Timestamp:     time.Now(),
		CPUUsage:      75.5,
		MemoryPercent: 50.0,
		DiskPercent:   50.0,
	}
}

// Test cases

func TestGetCurrentMetrics_Success(t *testing.T) {
	router, mockRepo := setupTestRouter()

	expectedMetric := createSampleMetric()
	mockRepo.On("GetLatest").Return(expectedMetric, nil)

	req, _ := http.NewRequest("GET", "/api/v1/metrics/current", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)

	mockRepo.AssertExpectations(t)
}

func TestGetCurrentMetrics_NotFound(t *testing.T) {
	router, mockRepo := setupTestRouter()

	mockRepo.On("GetLatest").Return((*models.Metric)(nil), fmt.Errorf("no metrics found"))

	req, _ := http.NewRequest("GET", "/api/v1/metrics/current", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.NotEmpty(t, response.Error)

	mockRepo.AssertExpectations(t)
}

func TestGetCurrentMetricsByHostname_Success(t *testing.T) {
	router, mockRepo := setupTestRouter()

	hostname := "test-host"
	expectedMetric := createSampleMetric()
	mockRepo.On("GetLatestByHostname", hostname).Return(expectedMetric, nil)

	req, _ := http.NewRequest("GET", "/api/v1/metrics/current/"+hostname, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	mockRepo.AssertExpectations(t)
}

func TestGetCurrentMetricsByHostname_NotFound(t *testing.T) {
	router, mockRepo := setupTestRouter()

	hostname := "nonexistent-host"
	mockRepo.On("GetLatestByHostname", hostname).Return((*models.Metric)(nil), fmt.Errorf("no metrics found for hostname %s", hostname))

	req, _ := http.NewRequest("GET", "/api/v1/metrics/current/"+hostname, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)

	mockRepo.AssertExpectations(t)
}

func TestGetMetricsHistory_Success(t *testing.T) {
	router, mockRepo := setupTestRouter()

	expectedMetrics := []*models.Metric{createSampleMetric()}
	mockRepo.On("GetHistory", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 50, 0).Return(expectedMetrics, nil)
	mockRepo.On("GetCountByDateRange", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(int64(1), nil)

	req, _ := http.NewRequest("GET", "/api/v1/metrics/history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, int64(1), response.Pagination.Total)

	mockRepo.AssertExpectations(t)
}

func TestGetMetricsHistory_WithTimeRange(t *testing.T) {
	router, mockRepo := setupTestRouter()

	expectedMetrics := []*models.Metric{createSampleMetric()}
	mockRepo.On("GetHistory", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 50, 0).Return(expectedMetrics, nil)
	mockRepo.On("GetCountByDateRange", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(int64(1), nil)

	from := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
	to := time.Now().Format(time.RFC3339)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/metrics/history?from=%s&to=%s", from, to), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockRepo.AssertExpectations(t)
}

func TestGetMetricsHistory_InvalidTimeFormat(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/metrics/history?from=invalid-time", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Invalid from time format")
}

func TestGetAverageMetrics_Success(t *testing.T) {
	router, mockRepo := setupTestRouter()

	expectedAverage := &models.AverageMetrics{
		Duration:       time.Hour,
		AvgCPUUsage:    65.5,
		AvgMemoryUsage: 70.2,
		AvgDiskUsage:   45.8,
		SampleCount:    120,
	}

	mockRepo.On("GetAverageUsage", time.Hour).Return(expectedAverage, nil)

	req, _ := http.NewRequest("GET", "/api/v1/metrics/average?duration=1h", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	mockRepo.AssertExpectations(t)
}

func TestGetAverageMetrics_InvalidDuration(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/metrics/average?duration=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Invalid duration format")
}

func TestGetTopHostsByUsage_Success(t *testing.T) {
	router, mockRepo := setupTestRouter()

	expectedHosts := []*models.HostUsage{
		{
			Hostname:       "host1",
			AvgCPUUsage:    85.5,
			AvgMemoryUsage: 70.2,
			AvgDiskUsage:   45.8,
			LastSeen:       time.Now(),
		},
	}

	mockRepo.On("GetTopHostsByUsage", "cpu", 10).Return(expectedHosts, nil)

	req, _ := http.NewRequest("GET", "/api/v1/metrics/top/cpu", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	mockRepo.AssertExpectations(t)
}

func TestGetTopHostsByUsage_InvalidType(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/metrics/top/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Invalid metric type")
}

func TestCreateMetric_Success(t *testing.T) {
	router, mockRepo := setupTestRouter()

	metric := createSampleMetric()
	mockRepo.On("Create", mock.AnythingOfType("*models.Metric")).Return(nil)

	jsonData, _ := json.Marshal(metric)
	req, _ := http.NewRequest("POST", "/api/v1/metrics", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	mockRepo.AssertExpectations(t)
}

func TestCreateMetric_InvalidJSON(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("POST", "/api/v1/metrics", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
}

func TestGetSystemStatus_Success(t *testing.T) {
	router, mockRepo := setupTestRouter()

	expectedStatus := []*models.SystemStatus{
		{
			Hostname:      "host1",
			CPUUsage:      75.5,
			MemoryPercent: 60.2,
			DiskPercent:   45.8,
			Timestamp:     time.Now(),
			Status:        "online",
		},
	}

	mockRepo.On("GetSystemStatus").Return(expectedStatus, nil)

	req, _ := http.NewRequest("GET", "/api/v1/system/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	mockRepo.AssertExpectations(t)
}

func TestGetStats_Success(t *testing.T) {
	router, mockRepo := setupTestRouter()

	mockRepo.On("GetTotalCount").Return(int64(1000), nil)

	req, _ := http.NewRequest("GET", "/api/v1/system/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	mockRepo.AssertExpectations(t)
}

func TestCleanupOldMetrics_Success(t *testing.T) {
	router, mockRepo := setupTestRouter()

	mockRepo.On("DeleteOldRecords", mock.AnythingOfType("time.Time")).Return(int64(50), nil)

	req, _ := http.NewRequest("DELETE", "/api/v1/admin/metrics/cleanup?days=30", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	mockRepo.AssertExpectations(t)
}

func TestCleanupOldMetrics_InvalidDays(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest("DELETE", "/api/v1/admin/metrics/cleanup?days=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "Invalid days parameter")
}

// Benchmark tests

func BenchmarkGetCurrentMetrics(b *testing.B) {
	router, mockRepo := setupTestRouter()

	expectedMetric := createSampleMetric()
	mockRepo.On("GetLatest").Return(expectedMetric, nil)

	req, _ := http.NewRequest("GET", "/api/v1/metrics/current", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkCreateMetric(b *testing.B) {
	router, mockRepo := setupTestRouter()

	metric := createSampleMetric()
	mockRepo.On("Create", mock.AnythingOfType("*models.Metric")).Return(nil)

	jsonData, _ := json.Marshal(metric)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/metrics", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
