/*
 * store/store.go — 设备数据持久化层
 *
 * 职责：
 *   1. 设备数据的内存缓存（slice + 读写锁）
 *   2. JSON 文件异步落盘（写操作异步，读操作启动时同步）
 *   3. 设备字段校验（MAC、端口、IP、名称唯一性）
 *
 * 设计原则：
 *   - 内存缓存是唯一真相源（single source of truth）
 *   - 所有写操作先更新内存，再异步落盘
 *   - 校验函数是纯函数，无副作用
 */

package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

/* ---------------------------------------------------------------------------
 * 数据模型
 * ---------------------------------------------------------------------------*/

// Device 表示一个可唤醒的网络设备
type Device struct {
	Name    string `json:"name"`    // 设备别名，唯一
	MAC     string `json:"mac"`     // MAC 地址，标准化格式 XX:XX:XX:XX:XX:XX
	Port    int    `json:"port"`    // WOL 端口，1-65535
	Address string `json:"address"` // 广播地址，默认 255.255.255.255
}

/* ---------------------------------------------------------------------------
 * 预定义错误
 * ---------------------------------------------------------------------------*/

var (
	ErrNameDuplicate = errors.New("设备名称已存在，请重新输入")
	ErrNameEmpty     = errors.New("设备名称不能为空")
	ErrMACInvalid    = errors.New("MAC 地址格式错误，请使用 XX:XX:XX:XX:XX:XX 格式")
	ErrMACPartial    = errors.New("MAC 地址输入未完成") // 实时校验时静默，Enter 时提示
	ErrPortInvalid   = errors.New("端口号非法，请输入 1-65535 范围内的数值")
	ErrIPInvalid     = errors.New("广播地址格式错误，请输入合法的 IPv4 地址")
	ErrNotFound      = errors.New("未查询到对应设备")
)

/* ---------------------------------------------------------------------------
 * Store — 设备数据仓库
 * ---------------------------------------------------------------------------*/

// Store 管理设备数据的持久化存储
type Store struct {
	mu      sync.RWMutex
	devices []Device
	path    string // JSON 文件绝对路径
}

// New 创建 Store 实例并加载已有数据
func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	s := &Store{
		devices: make([]Device, 0),
		path:    filepath.Join(dataDir, "devices.json"),
	}

	if err := s.load(); err != nil {
		// 文件损坏或不存在：初始化空数据，不报错
		s.devices = make([]Device, 0)
	}

	return s, nil
}

/* ---------------------------------------------------------------------------
 * 文件 I/O（私有）
 * ---------------------------------------------------------------------------*/

// load 从 JSON 文件加载设备数据
func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.flush() // 创建空文件
		}
		return err
	}

	if len(data) == 0 {
		return s.flush()
	}

	return json.Unmarshal(data, &s.devices)
}

// flush 将当前内存数据写入磁盘（同步，用于启动初始化）
func (s *Store) flush() error {
	data, err := json.MarshalIndent(s.devices, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化设备数据失败: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("写入数据文件失败: %w", err)
	}
	return nil
}

// Save 异步落盘（供外部通过 goroutine/cmd 调用）
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.flush()
}

/* ---------------------------------------------------------------------------
 * CRUD 操作
 * ---------------------------------------------------------------------------*/

// Add 新增设备，校验名称唯一性
func (s *Store) Add(d Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.devices {
		if strings.EqualFold(existing.Name, d.Name) {
			return ErrNameDuplicate
		}
	}

	s.devices = append(s.devices, d)
	return nil
}

// Update 按索引更新设备数据
func (s *Store) Update(idx int, d Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idx < 0 || idx >= len(s.devices) {
		return ErrNotFound
	}

	// 检查名称唯一性（排除自身）
	for i, existing := range s.devices {
		if i != idx && strings.EqualFold(existing.Name, d.Name) {
			return ErrNameDuplicate
		}
	}

	s.devices[idx] = d
	return nil
}

// Delete 按索引删除设备
func (s *Store) Delete(idx int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idx < 0 || idx >= len(s.devices) {
		return ErrNotFound
	}

	s.devices = append(s.devices[:idx], s.devices[idx+1:]...)
	return nil
}

// FindByName 按名称查找设备（大小写不敏感）
func (s *Store) FindByName(name string) (Device, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i, d := range s.devices {
		if strings.EqualFold(d.Name, name) {
			return d, i, nil
		}
	}
	return Device{}, -1, ErrNotFound
}

// FindByIndex 按序号查找设备（1-based 显示序号）
func (s *Store) FindByIndex(idx int) (Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if idx < 0 || idx >= len(s.devices) {
		return Device{}, ErrNotFound
	}
	return s.devices[idx], nil
}

// List 返回所有设备的只读副本
func (s *Store) List() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cp := make([]Device, len(s.devices))
	copy(cp, s.devices)
	return cp
}

// Count 返回设备数量
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.devices)
}

/* ---------------------------------------------------------------------------
 * 字段校验（纯函数，无副作用）
 * ---------------------------------------------------------------------------*/

// macHexRe 匹配 MAC 地址中的十六进制字符
var macHexRe = regexp.MustCompile(`[^0-9A-Fa-f]`)

// ValidateMAC 校验并标准化 MAC 地址
// 支持格式：XX:XX:XX:XX:XX:XX、XX-XX-XX-XX-XX-XX、XXXX.XXXX.XXXX、XXXXXXXXXXXX
// 返回标准格式 XX:XX:XX:XX:XX:XX
func ValidateMAC(s string) (string, error) {
	// 提取所有十六进制字符
	hex := macHexRe.ReplaceAllString(s, "")
	if len(hex) < 12 {
		return "", ErrMACPartial // 输入未完成，实时校验时不报错
	}
	if len(hex) > 12 {
		return "", ErrMACInvalid // 超出长度，格式错误
	}

	// 标准化为 XX:XX:XX:XX:XX:XX
	parts := make([]string, 6)
	for i := 0; i < 6; i++ {
		parts[i] = strings.ToUpper(hex[i*2 : i*2+2])
	}
	return strings.Join(parts, ":"), nil
}

// ValidatePort 校验端口号范围
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return ErrPortInvalid
	}
	return nil
}

// ValidateIP 校验 IPv4 地址格式
func ValidateIP(ip string) error {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ErrIPInvalid
	}
	if parsed.To4() == nil {
		return ErrIPInvalid
	}
	return nil
}

// ValidateName 校验设备名称
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameEmpty
	}
	return nil
}
