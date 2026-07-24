# WakeUp

> 终端里的 Wake-on-LAN 工具 —— 快速、纯键盘、单文件。

WakeUp 是一个基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 的 TUI 应用，用于管理设备并发送网络唤醒魔术包。设备数据本地存储，按键操作遵循 Vim 习惯，界面主题自动适配终端亮/暗模式。

[English](README.md)

## 功能

- **网络唤醒** — 选中设备，`Enter` 发送魔术包。结果异步回传，不阻塞界面。
- **设备管理** — 支持新增、编辑、删除设备。每条记录包含名称、MAC 地址、端口和广播地址。
- **Vim 式操作** — `j`/`k` 浏览、`dd` 删除、`a` 新增、`e` 编辑。`Esc` 始终回退到上一级。
- **自适应主题** — 全部样式使用 `lipgloss.AdaptiveColor` + 16 色 ANSI，自动跟随终端亮/暗模式。
- **单文件部署** — 一个静态编译的二进制文件。无运行时依赖、无守护进程、无配置文件。

## 安装

### 前置要求

[Go](https://go.dev/dl/) 1.26 或更高版本。

### 从源码构建

```bash
git clone https://github.com/rainLyn/WakeUp.git
cd WakeUp
make
```

编译产物为 `wakeup`。安装到系统路径：

```bash
make install
```

### Makefile 目标

| 命令 | 说明 |
|--------|-------------|
| `make` | 编译 `wakeup` |
| `make run` | 编译并启动 |
| `make clean` | 清理编译产物 |
| `make install` | 安装到 `/usr/local/bin` |

## 使用

```bash
./wakeup
```

### 按键绑定

| 上下文 | 按键 | 操作 |
|---------|-----|--------|
| 设备列表 | `j` / `k` / `↑` / `↓` | 移动光标 |
| 设备列表 | `Enter` | 唤醒所选设备 |
| 设备列表 | `a` | 新增设备 |
| 设备列表 | `e` | 编辑所选设备 |
| 设备列表 | `dd` | 删除所选设备 |
| 全局 | `Esc` | 关闭弹窗 / 返回上一页 |
| 全局 | `Ctrl+C` | 退出程序 |

## 数据存储

设备数据保存在 `~/.wakeup/devices.json`：

```json
[
  {
    "name": "NAS",
    "mac": "AA:BB:CC:DD:EE:FF",
    "port": 9,
    "address": "255.255.255.255"
  }
]
```

- 首次运行自动创建。数据文件损坏时自动重置为空。
- **异步写入** — 文件 I/O 不阻塞界面渲染。
- **内存缓存** — 读取瞬间完成，写入后台落盘。

## MAC 地址格式

WakeUp 将所有 MAC 地址标准化为 `XX:XX:XX:XX:XX:XX`。以下输入格式均可识别：

| 输入格式 | 标准化结果 |
|-------|-----------|
| `AA:BB:CC:DD:EE:FF` | `AA:BB:CC:DD:EE:FF` |
| `aa-bb-cc-dd-ee-ff` | `AA:BB:CC:DD:EE:FF` |
| `AABB.CCDD.EEFF` | `AA:BB:CC:DD:EE:FF` |
| `AABBCCDDEEFF` | `AA:BB:CC:DD:EE:FF` |

## 项目结构

```
WakeUp/
├── main.go          # 入口，顶层 Bubble Tea 模型
├── Makefile         # 构建自动化
├── store/
│   └── store.go     # 设备 CRUD、JSON 持久化、字段校验
├── wol/
│   └── wol.go       # 魔术包构造与 UDP 发送
└── ui/
    ├── ui.go        # 共享样式、弹窗、帮助栏、布局辅助
    ├── list.go      # 设备列表（主界面），内嵌唤醒/删除弹窗
    ├── form.go      # 新增/编辑表单（四字段）
    └── menu.go      # 经典菜单页（可选入口）
```

### 架构

- **页面自治路由** — 每个页面的 `Update()` 直接返回下一页模型。无中心路由器，消除 `switch pageType`。
- **Vim 模态系统** — 各页面独立管理模态状态（Normal、Confirm、WakeConfirm、Result）。`Esc` 是统一的回退键。
- **`tea.Cmd` 异步** — WOL 发包和文件写入以 goroutine 执行，结果通过类型化消息回传。
- **自适应主题** — 全部样式使用 `lipgloss.AdaptiveColor` + 终端 ANSI 色，跟随终端亮/暗模式自动切换。
