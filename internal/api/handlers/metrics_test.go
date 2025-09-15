package handlers

import (
	"bytes"
	"context"
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

// Added to satisfy repository.MetricsRepository interface
func (m *MockMetricsRepository) GetAverageUsageAllRecords() (*models.AverageMetrics, error) {
	args := m.Called()
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

// MockSystemCollector - Mock collector implementation
type MockSystemCollector struct {
	mock.Mock
}

func (m *MockSystemCollector) GetSystemMetrics() (*models.SystemMetrics, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SystemMetrics), args.Error(1)
}

func (m *MockSystemCollector) GetSystemInfo() (*models.SystemInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SystemInfo), args.Error(1)
}

func (m *MockSystemCollector) GetMetricsSnapshot() (*models.MetricsSnapshot, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MetricsSnapshot), args.Error(1)
}

func (m *MockSystemCollector) StartCollection(ctx context.Context, interval time.Duration) <-chan *models.SystemMetrics {
	args := m.Called(ctx, interval)
	return args.Get(0).(<-chan *models.SystemMetrics)
}

func (m *MockSystemCollector) GetTopProcesses(count int, sortBy string) ([]models.ProcessInfo, error) {
	args := m.Called(count, sortBy)
	return args.Get(0).([]models.ProcessInfo), args.Error(1)
}

func (m *MockSystemCollector) IsHealthy() (bool, []string, error) {
	args := m.Called()
	return args.Get(0).(bool), args.Get(1).([]string), args.Error(2)
}

// Test helper functions
func setupTestRouter() (*gin.Engine, *MockMetricsRepository, *MockSystemCollector) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockMetricsRepository)
	mockCollector := new(MockSystemCollector)
	handler := NewMetricsHandler(mockRepo, mockCollector)

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

	return router, mockRepo, mockCollector
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

func createSampleSystemMetrics() *models.SystemMetrics {
	return &models.SystemMetrics{
		CPU: models.CPUMetrics{
			Usage:       75.5,
			Cores:       4,
			CoreUsage:   []float64{70.0, 80.0, 75.0, 65.0},
			LoadAvg:     []float64{1.2, 1.5, 1.3},
			Frequency:   2400.0,
			Temperature: 45.0,
		},
		Memory: models.MemoryMetrics{
			Total:       8589934592,
			Used:        4294967296,
			Available:   4294967296,
			Free:        2147483648,
			Cached:      1073741824,
			Buffers:     536870912,
			Percent:     50.0,
			SwapTotal:   2147483648,
			SwapUsed:    0,
			SwapPercent: 0.0,
		},
		Disk: models.DiskMetrics{
			Total:   107374182400,
			Used:    53687091200,
			Free:    53687091200,
			Percent: 50.0,
			Partitions: []models.PartitionInfo{
				{
					Device:     "/dev/sda1",
					Mountpoint: "/",
					Fstype:     "ext4",
					Total:      107374182400,
					Used:       53687091200,
					Free:       53687091200,
					Percent:    50.0,
				},
			},
			IOStats: models.DiskIOStats{
				ReadBytes:  1048576,
				WriteBytes: 524288,
				ReadOps:    100,
				WriteOps:   50,
				ReadTime:   10,
				WriteTime:  5,
			},
			ReadSpeed:  10.5,
			WriteSpeed: 5.2,
		},
		Network: models.NetworkMetrics{
			Interfaces: []models.NetworkInterface{
				{
					Name:        "eth0",
					BytesSent:   10485760,
					BytesRecv:   52428800,
					PacketsSent: 1000,
					PacketsRecv: 5000,
					Errors:      0,
					Drops:       0,
				},
			},
			TotalSent:     10485760,
			TotalReceived: 52428800,
			UploadSpeed:   1.5,
			DownloadSpeed: 8.2,
		},
		Processes: models.ProcessActivity{
			TotalProcesses:   150,
			RunningProcesses: 120,
			StoppedProcesses: 5,
			ZombieProcesses:  0,
			TopProcesses: []models.ProcessInfo{
				{
					PID:         1234,
					Name:        "test-process",
					CPUPercent:  25.5,
					MemoryBytes: 134217728,
					Status:      "running",
				},
			},
		},
		Timestamp: time.Now(),
		Hostname:  "test-host",
		Uptime:    24 * time.Hour,
	}
}

// Test cases
func TestGetCurrentMetrics_Success(t *testing.T) {
	router, mockRepo, mockCollector := setupTestRouter()

	expectedSystemMetrics := createSampleSystemMetrics()
	mockCollector.On("GetSystemMetrics").Return(expectedSystemMetrics, nil)

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
	mockCollector.AssertExpectations(t)
}

func TestGetCurrentMetrics_NotFound(t *testing.T) {
	router, mockRepo, mockCollector := setupTestRouter()

	mockCollector.On("GetSystemMetrics").Return((*models.SystemMetrics)(nil), fmt.Errorf("no metrics found"))

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
	mockCollector.AssertExpectations(t)
}

func TestGetCurrentMetricsByHostname_Success(t *testing.T) {
	router, mockRepo, mockCollector := setupTestRouter()

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
	mockCollector.AssertExpectations(t)
}

func TestGetMetricsHistory_Success(t *testing.T) {
	router, mockRepo, mockCollector := setupTestRouter()

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
	mockCollector.AssertExpectations(t)
}

func TestCreateMetric_Success(t *testing.T) {
	router, mockRepo, mockCollector := setupTestRouter()

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
	mockCollector.AssertExpectations(t)
}

func TestGetStats_Success(t *testing.T) {
	router, mockRepo, mockCollector := setupTestRouter()

	mockRepo.On("GetTotalCount").Return(int64(1000), nil)
	mockRepo.On("GetSystemStatus").Return([]*models.SystemStatus{}, nil)
	// Updated: handler now calls GetAverageUsageAllRecords instead of GetAverageUsage(duration)
	mockRepo.On("GetAverageUsageAllRecords").Return((*models.AverageMetrics)(nil), fmt.Errorf("no data"))

	expectedSystemMetrics := createSampleSystemMetrics()
	mockCollector.On("GetSystemMetrics").Return(expectedSystemMetrics, nil)

	req, _ := http.NewRequest("GET", "/api/v1/system/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	mockRepo.AssertExpectations(t)
	mockCollector.AssertExpectations(t)
}
