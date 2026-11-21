# ImageToSecurePDF 合并方案

> 固定日期：2026-11-21 · 状态：方案已定，第一版原型完成，继续开发前请先读本文

## 1. 为什么合并

两个现有仓库在做同一件事：**把加密压缩包里的几百张图片，合并成一个体积很小、可加密码的 PDF**。

| 仓库 | 平台 | 技术栈 | 核心能力 |
|---|---|---|---|
| `borisKP/ImageToPDF_Studio` v1.10 | Windows | PySide6 + PyMuPDF + Python | 按图片 DPI 生成页面、Acrobat 式压缩、AES-256/128/RC4、书签、明暗主题、中英切换、拖拽排序、单 EXE 52 MB |
| `borisKP/image-to-encrypted-pdf` v2.3.1 | macOS 11+ | Swift/AppKit + Go 子进程 | 加密 ZIP/7z/RAR 直读、目标体积二分压缩、4 线程并行、截断 JPEG 修复、AES-256 R6、任务队列、双语/主题、DMG/ZIP |

两者业务链路一致，但引擎完全不同；不是把文件拼在一起，而是抽成**一个共享引擎 + 两个平台前端**。

## 2. 路线决策：Go 双端

用户已明确选择 **Go 双端**：

- 共享引擎以 macOS 版的 Go 实现为底座（它已经解决加密包直读、目标体积、并行、容错这些最难的部分）。
- Windows 保留 PySide6 前端，macOS 保留 Swift/AppKit 原生前端。
- 不做单一 Qt 跨平台界面（会失去 Mac 原生体验）；不做 Windows-only 先行版。

被否定的方案：

| 方案 | 否定原因 |
|---|---|
| Python + PyMuPDF 做共享引擎 | macOS 包会变大、性能下降，RAR 仍需外部工具 |
| 单一 Qt 界面 | Mac 端失去原生体验，B 的 2711 行原生 UI 要重写 |
| 浏览器 / 本地 HTTP 界面 | B 的 v1 已验证用户不喜欢，且违反纯离线原则 |

## 3. 目标架构

```
┌─────────────────────────────────────────────┐
│  一个产品：ImageToSecurePDF                  │
│  统一版本 VERSION / 统一配置 / 统一消息表     │
└───────────────────┬─────────────────────────┘
        ┌──────────┴──────────┐
        │                     │
┌───────▼────────┐    ┌───────▼────────────────┐
│ Windows 前端   │    │ macOS 前端             │
│ PySide6        │    │ Swift/AppKit（完整源码）│
└───────┬────────┘    └───────┬────────────────┘
        │  JSON 行协议        │
        │  list / thumb / build / cancel / version
        └──────────┬──────────┘
                   ▼
        ┌──────────────────────┐
        │ 共享 Go 引擎 engine/  │
        │ 加密包直读 → 解码缩放 │
        │ → JPEG → PDF → AES-256│
        └──────────────────────┘
```

统一配置契约：`shared/job.schema.json`
统一消息表：`shared/messages.json`（引擎只发 `{key, args}`，UI 负责翻译，切语言时历史日志可整体重翻）

## 4. 分阶段实施

### 阶段 0：冻结基线（已完成）
- 旧仓库打 tag，不重写历史、不强推
- 新仓库骨架：`engine/`、`windows/`、`macos/`、`shared/`、`tests/`、`docs/`
- 唯一版本源 `VERSION`

### 阶段 1：共享 Go 引擎（第一版已完成）
- 迁 B 的 Go 引擎，扩展配置对象
- 合入 A 的能力：DPI/纸张页面、无损嵌入、书签、元数据、权限位
- 协议补 `cancel` / `version`
- 多语言消息 key 化
- 测试覆盖：DPI 页面、无损、目标体积、AES-256、错误密码、截断 JPEG、书签、权限、取消

### 阶段 2：Windows 前端（进行中，仅骨架）
- 用 `windows/engine_client.py` 调共享引擎，不再用 A 的 Python 转换核心
- **必须基于原 Studio 的完整界面改，不要重写简化 UI**（见第 6 节教训）
- 合入明暗主题、九列详情、右键菜单、设置持久化
- PyInstaller 打单 EXE，helper 作为二进制资源打进去
- Windows CI

### 阶段 3：macOS 前端（第一版已完成）
- 直接复用真机完整 AppKit 源码（2711 行），只替换品牌/版本/helper
- `build_macos.sh` 从同一 `engine/` 编译 Universal 2 helper
- 后续把合并版新选项（DPI/纸张页面、无损模式、书签开关）加进原生界面
- 保持无浏览器、无 HTTP、无网络代码

### 阶段 4：发版与迁移（待做）
- 统一 `v3.0`：Windows EXE + macOS DMG/ZIP 同一 tag
- 旧仓库 README 加「已由新项目接替」，保留 Releases
- 迁移说明：默认页模式、RAR 不再依赖外部工具、RC4 移除、目标体积字段位置
- CI 矩阵：Windows 跑 PyInstaller，macOS 跑 Swift/Go/DMG，两边都跑引擎测试

## 5. 主要风险与止损点

| 风险 | 说明 | 对策 |
|---|---|---|
| Go PDF 写入器能力缺口 | 书签、元数据、权限位需自己补 | 第一版已补；若后续兼容性出问题，退回 Python 引擎方案重新评估 |
| 微信预览兼容性 | B 的 README 已注明加密 PDF 在微信内置预览可能需绕行 | 必须用真实分享场景回归 |
| RAR 覆盖范围 | 纯 Go rardecode 不支持分卷 RAR | 用真实压缩包验证；必要时保留外部 7-Zip 回退 |
| 页面默认语义不同 | A 按 DPI 还原物理尺寸，B 是 fit 无白边 | 默认保持 fit，DPI 模式作为高级选项 |
| RC4-128 | 安全性低 | 合并版直接取消，只保留 AES-256（AES-128 待定） |
| 中文字符串加密 | 书签/元数据里的中文暂未随 PDF 加密字典一起加密 | 后续版本补齐 |
| 界面重写陷阱 | 自己写的简化 UI 会被用户判定为半成品 | 必须复用两个仓库已有的完整界面源码 |

**止损点**：若阶段 1 的 PDF 能力移植成本明显超出预期，暂停，退回「Python 引擎 + 双端调用」方案重新评估，不硬推。

## 6. 关键教训：不要重写简化 UI

第一版曾写了一个 440 行 SwiftUI 草稿，用户明确反馈「非常粗糙，像一个半成品」。

正确做法：直接复用 `borisKP/image-to-encrypted-pdf` 的 `macapp/App.swift`（2711 行完整 AppKit 实现），只改版本、品牌、helper 路径。草稿已备份为 `macos/App.swift.swiftui-draft.bak`，仅作警示。

Windows 端同理：应复用 `ImageToPDF_Studio/src/app.py` + `i18n.py` + `theme.py` 的完整界面，只把 `core.convert()` / `archive.extract_archive()` 换成引擎调用。

## 7. 第一版实测数据

- 输入：24 张 JPEG（3.9 MB），加密 ZIP 后 835 KB（测试密码 `Test123456`）
- 输出：995 KB / 24 页 AES-256 PDF
- 耗时：1.4 秒（CLI）；完整界面应用内自检 400 ms / 3.4 MB（目标体积 20 MB）
- 校验：24 个书签、`/Encrypt`、`/Outlines`、`/DCTDecode`、`/Info` 均存在
- `go test ./...` 全过；`SELFTEST PASS: 24 pages / 3.4 MB / 400ms`

## 8. 继续开发前的检查清单

1. 读本文、`STATUS.md`、`DECISIONS.md`
2. 确认 `VERSION` 与各端版本常量一致
3. `cd engine && go test ./...`
4. `./macos/build.sh` 并打开 App 看界面是否仍与真机一致
5. Windows 端从原 Studio 完整界面开始改，不从 `windows/app.py` 草稿开始
6. 任何界面改动都要截图核对，避免再次出现半成品感
