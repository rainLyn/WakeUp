# ⚡ WakeUp

**终端里的网络唤醒工具。** 极速、纯键盘、Vim 范儿。

WakeUp 是一个基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 的 TUI 网络唤醒（Wake-on-LAN）工具。支持设备管理、一键唤醒、Vim 风格按键导航、终端自适应配色，编译出来就一个二进制，零运行时依赖。

## ✨ 功能

- **⚡ 一键唤醒** — 列表里选中设备，`Enter` 确认弹窗，异步发包，结果实时弹窗反馈
- **📋 设备管理** — 新增/编辑/删除设备，名称、MAC、端口、广播地址四个字段。数据持久化到 `~/.wakeup/devices.json`
- **🖖 Vim 模态操作** — 列表 `j`/`k` 浏览、`dd` 删除、`Enter` 唤醒、`a` 新增、`e` 编辑。符合 Vim 用户直觉
- **📦 单文件部署** — `go build` 完事儿。无配置文件，无守护进程，启动 < 200ms

## 🗂 数据存储

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

- 首次运行**自动创建**。数据文件损坏时自动重置为空
- **异步写入**——文件 I/O 不阻塞界面渲染
- **内存缓存**——读取瞬间完成，增删改后异步落盘

## 🏗 项目结构

```
WakeUp/
├── main.go             # 入口，顶层 Bubble Tea 模型（页面路由）
├── store/
│   └── store.go        # 设备 CRUD、JSON 持久化、字段校验
├── wol/
│   └── wol.go          # WOL 魔术包构造 + UDP 发送
└── ui/
    ├── ui.go           # 共享样式、弹窗、帮助栏、PinBottom
    ├── list.go         # 设备列表（主界面），内嵌唤醒/删除弹窗
    ├── form.go         # 新增/编辑表单（四字段一屏展示）
    └── menu.go         # 经典菜单页（可选入口）
```

**架构要点：**

- **页面自治路由**——每个页面的 `Update()` 返回下一页 Model，无需中心路由器，消除了 `switch pageType`
- **Vim 模态系统**——各页面独立管理模态（Normal / Confirm / WakeConfirm / Result），ESC 始终回退
- **异步通过 `tea.Cmd`**——WOL 发包和文件写入以 goroutine 运行，通过类型化消息回传结果
- **零硬编码色值**——所有样式使用 `lipgloss.AdaptiveColor` + 终端 16 色，自动跟随终端深浅主题

## 📋 MAC 格式兼容

WakeUp 自动将 MAC 地址标准化为 `XX:XX:XX:XX:XX:XX`：

| 输入 | 标准化结果 |
|------|-----------|
| `AA:BB:CC:DD:EE:FF` | `AA:BB:CC:DD:EE:FF` |
| `aa-bb-cc-dd-ee-ff` | `AA:BB:CC:DD:EE:FF` |
| `AABB.CCDD.EEFF` | `AA:BB:CC:DD:EE:FF` |
| `AABBCCDDEEFF` | `AA:BB:CC:DD:EE:FF` |