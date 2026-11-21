# 当前状态与待办

固定日期：2026-11-21 18:20 GMT+8。改天继续时从这里开始。

## 已完成

- 仓库骨架、唯一版本源 `VERSION = 3.0.0-alpha.1`
- 共享 Go 引擎：加密 ZIP/7z/RAR 直读、目标体积压缩、AES-256 + 权限、书签、元数据、DPI/纸张页面、无损嵌入、并行、JPEG 修复、错误报告、JSON 行协议 `list/thumb/build/cancel/version`
- `go test ./...` 全部通过
- macOS 完整 AppKit 界面：复用真机 2711 行源码，品牌/版本/helper 已换，编译、签名、自检通过
- Windows 前端骨架：PySide6 界面 + `engine_client.py`，能调协议但未打包 EXE
- `shared/job.schema.json`、`shared/messages.json`
- 方案/决策/实测说明文档

## 实测数据

- CLI：24 张图片 / 加密 ZIP 835 KB → 995 KB / 24 页 AES-256 PDF / 1.4 s
- 完整界面自检：24 页 / 3.4 MB / 400 ms（目标体积 20 MB）
- 错误压缩包密码：生成失败，不留下 PDF
- 内置 helper 身份：`ImageToSecurePDF 3.0.0-alpha.1`

## 未完成 / 下一步（按优先级）

1. **Windows 前端正式化**：基于 `ImageToPDF_Studio/src/{app,i18n,theme}.py` 完整界面改，替换 `core.convert()` / `archive.extract_archive()` 为引擎调用；不要从当前 440 行骨架扩写。
2. **macOS 新增合并版选项**：把 DPI/纸张页面、无损模式、书签开关加进原生界面（当前引擎已支持，UI 还没暴露）。
3. **微信预览兼容性回归**：用真实要发的加密 PDF，测微信内置预览能否打开加密 PDF；必要时提供绕行方案。
4. **中文字符串加密**：书签/元数据里的中文目前未随 PDF 加密字典加密，补齐。
5. **AES-128 决策**：确认是否加入；RC4 不加。
6. **分卷 RAR**：确认是否需要外部 7-Zip 回退。
7. **打包与发版**：Windows 单 EXE（helper 内嵌）+ macOS DMG/ZIP，同一 tag `v3.0.0`。
8. **CI**：GitHub Actions 矩阵，两端构建 + 引擎测试。
9. **旧仓库归档**：加「已由新项目接替」说明，保留 Releases。

## 本机环境备忘

- Go 工具链临时在 `/tmp/gotoolchain/go/bin/go`（go1.27.4），系统 PATH 未安装 Go；继续开发时先 `export PATH=/tmp/gotoolchain/go/bin:$PATH`，或自行安装。
- macOS 真机 App：`/Applications/图片合集转加密PDF.app`
- 真机完整源码：`~/WorkBuddy/2026-11-06-22-31-25/src/macapp/App.swift`
- 原 Studio 源码：`~/WorkBuddy/2026-11-06-22-31-25/src/`（本机副本）与 GitHub `borisKP/ImageToPDF_Studio`
- 本地测试数据：`tests/fixtures/`（未入库，可重新生成）

## 重新生成测试数据

```bash
# 生成 24 张 JPEG（示例脚本在 docs 或本地 /tmp/genimgs.go）
go run /tmp/genimgs.go tests/fixtures/images 24
zip -P Test123456 -q -r tests/fixtures/photos_encrypted.zip tests/fixtures/images
```

## 构建命令

```bash
cd engine && go test ./...
cd .. && ./macos/build.sh
```

`build.sh` 会从根目录 `VERSION` 同步 Info.plist，编译共享引擎与 AppKit 前端到 `macos/ImageToSecurePDF.app`，运行后需 `codesign --force --deep -s -`。
