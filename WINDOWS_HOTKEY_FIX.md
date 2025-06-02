# Windows Hotkey Implementation Fix

## 问题描述

原始的MathReX应用在Windows平台编译时遇到以下问题：

1. **gohook库不兼容Windows**: `github.com/robotn/gohook`在Windows上有未定义的函数
2. **链接错误**: `-ldl`标志在Windows上不存在（libdl是Unix/Linux特有的）
3. **CGO依赖复杂**: 多个包需要CGO，使Windows构建复杂化

## 解决方案

### 1. 平台特定的热键实现

我们创建了一个平台无关的热键管理接口，并为不同平台提供了特定实现：

#### 文件结构
- `hotkey.go` - 通用接口和管理器
- `hotkey_windows.go` - Windows特定实现（使用Windows API）
- `hotkey_unix.go` - Unix/Linux/macOS实现（使用gohook）

#### 构建标签
```go
//go:build windows
// +build windows

//go:build !windows  
// +build !windows
```

### 2. Windows实现详情

Windows实现使用原生Windows API：
- `RegisterHotKey` - 注册全局热键
- `UnregisterHotKey` - 注销热键
- `golang.org/x/sys/windows` - Windows系统调用
- 消息循环处理热键事件

### 3. 代码更改

#### 主要更改：
1. **移除直接的gohook导入** - 从main.go中移除
2. **添加平台无关接口** - 通过HotkeyManager接口
3. **更新go.mod** - 添加`golang.org/x/sys`依赖
4. **重构主程序** - 使用新的热键管理器

#### 关键函数：
```go
// 初始化热键管理器
InitializeHotkeyManager() error

// 注册全局热键
RegisterGlobalHotkey(shortcut string, callback func()) error

// 开始快捷键捕获
StartGlobalShortcutCapture() (<-chan string, error)
```

### 4. GitHub Actions更新

简化了Windows构建流程：
- 移除了复杂的fallback策略
- 直接使用新的热键实现
- 保留了CGO_LDFLAGS清理逻辑

## 技术细节

### Windows热键实现特点

1. **使用Windows API**: 直接调用`user32.dll`中的函数
2. **虚拟键码映射**: 支持常用键的映射
3. **修饰键支持**: Ctrl, Alt, Shift, Win键
4. **消息窗口**: 创建隐藏窗口接收热键消息

### Unix/Linux/macOS实现

1. **保持gohook**: 继续使用原有的gohook库
2. **条件编译**: 只在非Windows平台编译
3. **向后兼容**: 保持原有功能不变

## 预期效果

### 修复前的问题：
```
Error: C:\Users\runneradmin\go\pkg\mod\github.com\robotn\gohook@v0.42.1\event.go:51:10: undefined: addEvent
Error: cannot find -ldl: No such file or directory
```

### 修复后的预期：
- Windows构建成功完成
- 热键功能在所有平台正常工作
- 不再依赖Unix特有的库

## 测试建议

### 本地测试（Windows环境）：
```bash
# 设置Windows环境
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1

# 构建测试
go build -v .
```

### GitHub Actions测试：
推送代码到仓库，GitHub Actions将自动：
1. 构建Windows版本
2. 验证热键功能
3. 创建发布包

## 兼容性

- **Windows**: 使用原生Windows API，支持Windows 7+
- **macOS**: 继续使用gohook，保持原有功能
- **Linux**: 继续使用gohook，保持原有功能

## 依赖更新

### 新增依赖：
- `golang.org/x/sys v0.33.0` - Windows系统调用

### 条件依赖：
- `github.com/robotn/gohook` - 仅在非Windows平台使用

## 后续维护

1. **Windows特定功能**: 可以在`hotkey_windows.go`中扩展
2. **跨平台功能**: 在`hotkey.go`接口中添加
3. **测试**: 建议在实际Windows环境中测试热键功能

## 总结

这个修复彻底解决了Windows编译问题，通过：
1. 替换有问题的gohook库为平台特定实现
2. 使用Windows原生API提供热键功能
3. 保持其他平台的兼容性
4. 简化构建流程

现在MathReX应该能够在Windows平台成功编译和运行。
