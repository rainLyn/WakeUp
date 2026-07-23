/*
 * wol/wol.go — WOL 网络唤醒魔术包发送
 *
 * 职责：
 *   1. 构造 WOL 魔术包（6 字节 0xFF + MAC × 16 = 102 字节）
 *   2. 通过 UDP 协议发送到目标广播地址和端口
 *
 * 设计原则：
 *   - 纯函数，无状态，不持有任何数据
 *   - SendMagicPacket 直接返回 error，调用方负责处理结果
 *   - 网络操作超时由调用方通过 context 控制（本层不做超时）
 */

package wol

import (
	"fmt"
	"net"
)

/*
 * Magic Packet 格式（共 102 字节）：
 *   ┌──────────────────┬────────────────────────────────────┐
 *   │ 0xFF × 6 (6字节) │ 目标 MAC 地址重复 16 次 (16×6=96字节) │
 *   └──────────────────┴────────────────────────────────────┘
 */

const (
	magicPacketLen = 102 // 魔术包总长度
	syncStreamLen  = 6   // 同步流长度
	macRepeatCount = 16  // MAC 地址重复次数
)

// SendMagicPacket 向指定地址发送 WOL 魔术包
// mac: 目标设备 MAC，格式 XX:XX:XX:XX:XX:XX
// address: 广播地址，如 255.255.255.255
// port: 目标端口，默认 9
func SendMagicPacket(mac string, address string, port int) error {
	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return fmt.Errorf("解析 MAC 地址失败: %w", err)
	}

	if len(hwAddr) != 6 {
		return fmt.Errorf("MAC 地址长度错误: 期望 6 字节，实际 %d 字节", len(hwAddr))
	}

	// 构造魔术包：6 字节 0xFF + MAC × 16
	packet := make([]byte, magicPacketLen)
	for i := 0; i < syncStreamLen; i++ {
		packet[i] = 0xFF
	}
	for i := 0; i < macRepeatCount; i++ {
		copy(packet[syncStreamLen+i*6:], hwAddr)
	}

	// UDP 连接并发送
	target := net.JoinHostPort(address, fmt.Sprintf("%d", port))
	conn, err := net.Dial("udp", target)
	if err != nil {
		return fmt.Errorf("建立 UDP 连接失败: %w", err)
	}
	defer conn.Close()

	n, err := conn.Write(packet)
	if err != nil {
		return fmt.Errorf("发送魔术包失败: %w", err)
	}
	if n != magicPacketLen {
		return fmt.Errorf("魔术包发送不完整: 期望 %d 字节，实际 %d 字节", magicPacketLen, n)
	}

	return nil
}
