# Changelog

## [3.0.0-alpha.1] - 2026-11-21

### Added

- 新项目骨架：共享 Go 引擎、macOS SwiftUI 原型、Windows PySide6 前端骨架、统一配置契约与消息表。
- 引擎合入 DPI/纸张页面模式、文件名书签、PDF 元数据、所有者密码与权限位、无损嵌入模式、取消协议。
- 统一 JSON 行协议新增 `cancel` 与 `version`。
- macOS 原型可直接打开加密 ZIP / 7z / RAR，设置目标体积与 PDF 密码并生成 PDF。

### Notes

- 第一版仅支持 AES-256；RC4/AES-128 暂不提供。
- Windows 前端尚未打包 EXE。
- 加密 PDF 在微信内置预览中的兼容性仍需真实场景回归。
