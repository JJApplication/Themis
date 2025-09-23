package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AppPortStorage APP端口存储管理器
type AppPortStorage struct {
	mu           sync.RWMutex
	appPorts     map[string]int // APP名称到端口的映射
	dataFile     string         // 数据文件路径
	syncInterval time.Duration  // 同步间隔
	stopChan     chan struct{}  // 停止信号
	wg           sync.WaitGroup // 等待组
}

// PortData 端口数据结构（用于JSON序列化）
type PortData struct {
	AppPorts  map[string]int `json:"app_ports"`  // APP端口映射
	Timestamp int64          `json:"timestamp"`  // 时间戳
	Version   string         `json:"version"`    // 版本信息
}

// NewAppPortStorage 创建新的APP端口存储管理器
func NewAppPortStorage(dataFile string, syncInterval time.Duration) *AppPortStorage {
	return &AppPortStorage{
		appPorts:     make(map[string]int),
		dataFile:     dataFile,
		syncInterval: syncInterval,
		stopChan:     make(chan struct{}),
	}
}

// LoadFromFile 从文件加载数据
func (s *AppPortStorage) LoadFromFile() error {
	// 确保目录存在
	dir := filepath.Dir(s.dataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %v", err)
	}

	// 如果文件不存在，创建空文件
	if _, err := os.Stat(s.dataFile); os.IsNotExist(err) {
		return s.saveToFile() // 创建空的数据文件
	}

	data, err := os.ReadFile(s.dataFile)
	if err != nil {
		return fmt.Errorf("读取数据文件失败: %v", err)
	}

	// 如果文件为空，初始化为空数据
	if len(data) == 0 {
		return nil
	}

	var portData PortData
	if err := json.Unmarshal(data, &portData); err != nil {
		return fmt.Errorf("解析数据文件失败: %v", err)
	}

	s.mu.Lock()
	s.appPorts = portData.AppPorts
	if s.appPorts == nil {
		s.appPorts = make(map[string]int)
	}
	s.mu.Unlock()

	return nil
}

// saveToFile 保存数据到文件
func (s *AppPortStorage) saveToFile() error {
	s.mu.RLock()
	portData := PortData{
		AppPorts:  make(map[string]int),
		Timestamp: time.Now().Unix(),
		Version:   "1.0",
	}
	// 复制数据避免竞态条件
	for k, v := range s.appPorts {
		portData.AppPorts[k] = v
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(portData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化数据失败: %v", err)
	}

	// 确保目录存在
	dir := filepath.Dir(s.dataFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %v", err)
	}

	if err := os.WriteFile(s.dataFile, data, 0644); err != nil {
		return fmt.Errorf("写入数据文件失败: %v", err)
	}

	return nil
}

// StartAutoSync 启动自动同步
func (s *AppPortStorage) StartAutoSync() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.syncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.saveToFile(); err != nil {
					fmt.Printf("自动同步失败: %v\n", err)
				}
			case <-s.stopChan:
				return
			}
		}
	}()
}

// StopAutoSync 停止自动同步
func (s *AppPortStorage) StopAutoSync() {
	close(s.stopChan)
	s.wg.Wait()
	// 最后同步一次
	s.saveToFile()
}

// SetAppPort 设置APP端口
func (s *AppPortStorage) SetAppPort(appName string, port int) error {
	if appName == "" {
		return fmt.Errorf("APP名称不能为空")
	}
	if port <= 0 || port > 65535 {
		return fmt.Errorf("无效的端口号: %d", port)
	}

	s.mu.Lock()
	s.appPorts[appName] = port
	s.mu.Unlock()

	return nil
}

// GetAppPort 获取APP端口
func (s *AppPortStorage) GetAppPort(appName string) (int, error) {
	if appName == "" {
		return 0, fmt.Errorf("APP名称不能为空")
	}

	s.mu.RLock()
	port, exists := s.appPorts[appName]
	s.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("APP '%s' 不存在", appName)
	}

	return port, nil
}

// DeleteAppPort 删除APP端口
func (s *AppPortStorage) DeleteAppPort(appName string) error {
	if appName == "" {
		return fmt.Errorf("APP名称不能为空")
	}

	s.mu.Lock()
	_, exists := s.appPorts[appName]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("APP '%s' 不存在", appName)
	}
	delete(s.appPorts, appName)
	s.mu.Unlock()

	// 立即同步到文件
	return s.saveToFile()
}

// HasApp 检查APP是否存在
func (s *AppPortStorage) HasApp(appName string) bool {
	s.mu.RLock()
	_, exists := s.appPorts[appName]
	s.mu.RUnlock()
	return exists
}

// GetAllApps 获取所有APP列表
func (s *AppPortStorage) GetAllApps() map[string]int {
	s.mu.RLock()
	result := make(map[string]int)
	for k, v := range s.appPorts {
		result[k] = v
	}
	s.mu.RUnlock()
	return result
}

// SyncToFile 手动同步到文件
func (s *AppPortStorage) SyncToFile() error {
	return s.saveToFile()
}