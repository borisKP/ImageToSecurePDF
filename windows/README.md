# Windows 前端

这一版是骨架原型：PySide6 界面通过 `engine_client.py` 调用共享 Go 引擎，不再使用原 `ImageToPDF_Studio` 的 Python 转换核心。

## 源码运行

1. 先把 Go 引擎交叉编译为 `windows/imgtopdf-helper.exe`：

```bat
set GOOS=windows
set GOARCH=amd64
go build -o ..\windows\imgtopdf-helper.exe .
```

2. 安装依赖并运行：

```bat
python -m pip install -r requirements.txt
python app.py
```

## 已接通

- 选择加密 ZIP / 7z / RAR 或图片文件夹
- 读取图片清单、上下移顺序
- 目标体积、压缩模式、页面模式、纸张、书签
- PDF 打开密码、取消、进度和日志

## 待做

- 用 PyInstaller 打成单 EXE，把 helper 作为二进制资源打进去
- 把原 Studio 的明暗主题、中英切换、九列详情视图、右键菜单合入
- Windows CI 与自动化测试
