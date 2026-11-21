# ImageToSecurePDF

把加密压缩包里的几百张图片，合并成一个体积很小、可加密码的 PDF。

这是 `borisKP/ImageToPDF_Studio`（Windows）与 `borisKP/image-to-encrypted-pdf`（macOS）的合并版，采用「一个共享 Go 引擎 + 两个原生前端」路线。

> **更换智能体 / 重新接手请先读 [docs/CONVERSATION.md](docs/CONVERSATION.md)**（对话全流程交接），再读 [docs/PLAN.md](docs/PLAN.md)、[docs/DECISIONS.md](docs/DECISIONS.md)、[docs/STATUS.md](docs/STATUS.md)。

## 架构

```
Windows 前端 (PySide6)  ─┐
                        ├─ JSON 行协议 ─ 共享 Go 引擎 engine/
macOS 前端 (AppKit)    ─┘                 list / thumb / build / cancel / version
```

- 统一版本源：根目录 `VERSION`（当前 `3.0.0-alpha.1`）
- 统一配置契约：[shared/job.schema.json](shared/job.schema.json)
- 统一消息表：[shared/messages.json](shared/messages.json)

## 功能

- 直接读取加密 ZIP / 7z / RAR / RAR5，密码只在内存中使用
- 目标体积 MB 驱动的 JPEG 二分压缩，必要时多轮重压
- 有界并行处理，清单顺序即 PDF 页序
- 截断 JPEG 自动修复；坏图只跳过自己那一页并输出诊断报告
- PDF 2.0 AES-256（R=6）打开密码、所有者密码与权限位
- 文件名书签、PDF 元数据
- DPI / 统一纸张页面模式、无损嵌入模式
- 后台转换与取消、进度与日志
- macOS 端：原生分栏、拖拽排序、预览、明暗主题、中英切换、任务队列

## 目录

```
engine/          共享 Go 引擎
macos/           macOS AppKit 前端与构建脚本
windows/         PySide6 前端骨架与引擎客户端
shared/          配置契约与消息表
tests/           单元测试与样例生成脚本
docs/            方案、决策、状态、实测说明
```

## 从源码构建

需要 Go 1.27+；macOS 构建还需要 Swift/Xcode 命令行工具。

```bash
# 引擎测试
cd engine && go test ./...

# 构建 macOS App（会把 VERSION 同步进 Info.plist）
cd .. && ./macos/build.sh
codesign --force --deep -s - macos/ImageToSecurePDF.app
open macos/ImageToSecurePDF.app
```

Windows 前端：

```bat
cd engine
set GOOS=windows
set GOARCH=amd64
go build -o ..\windows\imgtopdf-helper.exe .

cd ..\windows
python -m pip install -r requirements.txt
python app.py
```

## CLI 用法

```bash
imgtopdf-helper build \
  --src photos.rar --archive-pass '包密码' \
  --pdf-pass '打开密码' --out out.pdf \
  --target-mb 20 --max-edge 1600 --page fit
```

向 stdin 写 `{"type":"cancel"}\n` 可取消正在运行的任务。

## 重新生成测试数据

```bash
# 生成 24 张 JPEG，再打成带密码 ZIP（需要 go 与 zip）
go run tests/gen_images.go tests/fixtures/images 24
cd tests/fixtures && zip -P Test123456 -q -r photos_encrypted.zip images
```

## 实测

- 24 张加密 ZIP 图片 → 995 KB / 24 页 AES-256 PDF，1.4 秒
- macOS 完整界面自检：`SELFTEST PASS: 24 pages / 3.4 MB / 400ms`
- `go test ./...` 全过

## 隐私

无遥测、无统计、无网络请求。密码不会写入日志或错误报告。

## 许可

MIT © 2026 borisKP
