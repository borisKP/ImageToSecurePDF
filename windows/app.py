# -*- coding: utf-8 -*-
"""ImageToSecurePDF Windows 前端原型（PySide6）。"""
from __future__ import annotations

import os
import sys

from PySide6.QtCore import QObject, Qt, Signal
from PySide6.QtWidgets import (
    QApplication, QCheckBox, QComboBox, QFileDialog, QFormLayout, QGroupBox,
    QHBoxLayout, QLabel, QLineEdit, QListWidget, QListWidgetItem,
    QMainWindow, QPlainTextEdit, QProgressBar, QPushButton, QSpinBox,
    QVBoxLayout, QWidget,
)

from engine_client import EngineClient


class SignalBridge(QObject):
    event = Signal(object)
    finished = Signal(int)


class MainWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("ImageToSecurePDF 3.0.0-alpha.1")
        self.resize(760, 820)

        self.bridge = SignalBridge()
        helper = os.path.join(os.path.dirname(os.path.abspath(__file__)), "imgtopdf-helper.exe")
        self.client = EngineClient(helper, self.bridge.event.emit, self.bridge.finished.emit)
        self.bridge.event.connect(self.on_event)
        self.bridge.finished.connect(self.on_finished)

        self.source = QLineEdit()
        self.archive_pass = QLineEdit()
        self.archive_pass.setEchoMode(QLineEdit.Password)
        self.output = QLineEdit()
        self.pdf_pass = QLineEdit()
        self.pdf_pass.setEchoMode(QLineEdit.Password)

        self.target_mb = QSpinBox()
        self.target_mb.setRange(1, 500)
        self.target_mb.setValue(20)
        self.compression = QComboBox()
        self.compression.addItems(["target-mb", "preset", "lossless"])
        self.preset = QComboBox()
        self.preset.addItems(["screen", "balanced", "print", "small"])
        self.page_mode = QComboBox()
        self.page_mode.addItems(["fit", "image", "paper"])
        self.paper = QComboBox()
        self.paper.addItems(["A4", "A3", "A5", "B5", "Letter", "Legal"])
        self.bookmarks = QCheckBox("按文件名写书签")
        self.bookmarks.setChecked(True)

        self.items = QListWidget()
        self.logs = QPlainTextEdit()
        self.logs.setReadOnly(True)
        self.progress = QProgressBar()
        self.progress.setRange(0, 100)
        self.status = QLabel("就绪")

        self.btn_build = QPushButton("生成 PDF")
        self.btn_cancel = QPushButton("取消")
        self.btn_cancel.setEnabled(False)
        self.btn_build.clicked.connect(self.build)
        self.btn_cancel.clicked.connect(self.client.cancel)

        input_box = QGroupBox("① 输入")
        form = QFormLayout()
        src_row = QHBoxLayout()
        src_row.addWidget(self.source)
        b = QPushButton("选择…")
        b.clicked.connect(self.choose_input)
        src_row.addWidget(b)
        form.addRow("压缩包/文件夹", src_row)
        form.addRow("压缩包密码", self.archive_pass)
        bl = QPushButton("读取图片清单")
        bl.clicked.connect(self.load_list)
        form.addRow("", bl)
        input_box.setLayout(form)

        output_box = QGroupBox("② 输出与加密")
        form = QFormLayout()
        out_row = QHBoxLayout()
        out_row.addWidget(self.output)
        b = QPushButton("选择…")
        b.clicked.connect(self.choose_output)
        out_row.addWidget(b)
        form.addRow("输出 PDF", out_row)
        form.addRow("PDF 打开密码", self.pdf_pass)
        output_box.setLayout(form)

        opt_box = QGroupBox("③ 压缩与页面")
        form = QFormLayout()
        form.addRow("压缩模式", self.compression)
        form.addRow("预设", self.preset)
        form.addRow("目标体积 MB", self.target_mb)
        form.addRow("页面模式", self.page_mode)
        form.addRow("纸张", self.paper)
        form.addRow("", self.bookmarks)
        opt_box.setLayout(form)

        list_box = QGroupBox("④ 图片清单（顺序即 PDF 页序）")
        lay = QVBoxLayout()
        lay.addWidget(self.items)
        row = QHBoxLayout()
        up = QPushButton("上移")
        down = QPushButton("下移")
        up.clicked.connect(lambda: self.move(-1))
        down.clicked.connect(lambda: self.move(1))
        row.addWidget(up)
        row.addWidget(down)
        row.addStretch(1)
        lay.addLayout(row)
        list_box.setLayout(lay)

        central = QWidget()
        root = QVBoxLayout(central)
        root.addWidget(input_box)
        root.addWidget(output_box)
        root.addWidget(opt_box)
        act = QHBoxLayout()
        act.addWidget(self.btn_build)
        act.addWidget(self.btn_cancel)
        act.addWidget(self.progress, 1)
        act.addWidget(self.status)
        root.addLayout(act)
        root.addWidget(list_box, 1)
        root.addWidget(QLabel("⑤ 引擎日志"))
        root.addWidget(self.logs)
        self.setCentralWidget(central)

    def choose_input(self):
        path, _ = QFileDialog.getOpenFileName(
            self, "选择压缩包", "", "Archives (*.zip *.7z *.rar);;All files (*.*)")
        if path:
            self.source.setText(path)
            base = os.path.splitext(os.path.basename(path))[0]
            self.output.setText(os.path.join(os.path.dirname(path), base + "_secure.pdf"))

    def choose_output(self):
        path, _ = QFileDialog.getSaveFileName(self, "输出 PDF", "", "PDF (*.pdf)")
        if path:
            self.output.setText(path)

    def load_list(self):
        if self.client.is_running() or not self.source.text():
            return
        self.logs.appendPlainText("读取图片清单…")
        self.client.run(["list", "-src", self.source.text(),
                         "-archive-pass", self.archive_pass.text()])

    def build(self):
        if self.client.is_running():
            return
        if not self.source.text() or not self.output.text():
            self.status.setText("请先选择输入和输出路径")
            return
        order = [int(item.data(Qt.UserRole)) for item in
                 [self.items.item(i) for i in range(self.items.count())]]
        args = [
            "build",
            "-src", self.source.text(),
            "-out", self.output.text(),
            "-archive-pass", self.archive_pass.text(),
            "-pdf-pass", self.pdf_pass.text(),
            "-target-mb", str(self.target_mb.value()),
            "-compression-mode", self.compression.currentText(),
            "-preset", self.preset.currentText(),
            "-page", self.page_mode.currentText(),
            "-paper", self.paper.currentText(),
            "-bookmarks", "true" if self.bookmarks.isChecked() else "false",
            "-lang", "zh",
        ]
        if order:
            args += ["-order", ",".join(map(str, order))]
        self.btn_cancel.setEnabled(True)
        self.client.run(args)

    def move(self, delta):
        row = self.items.currentRow()
        target = row + delta
        if row < 0 or target < 0 or target >= self.items.count():
            return
        a = self.items.takeItem(row)
        b = self.items.takeItem(target)
        self.items.insertItem(row, b)
        self.items.insertItem(target, a)
        self.items.setCurrentRow(target)

    def on_event(self, event):
        t = event.get("type")
        if t == "list":
            self.items.clear()
            for it in event.get("items", []):
                row = QListWidgetItem(f"{it['index'] + 1}. {it['name']}  {it.get('sizeText', '')}")
                row.setData(Qt.UserRole, int(it["index"]))
                self.items.addItem(row)
            self.status.setText(f"找到 {self.items.count()} 张图片")
        elif t == "progress":
            self.progress.setValue(int(event.get("percent", 0)))
            self.status.setText(event.get("message", ""))
        elif t == "log":
            self.logs.appendPlainText(event.get("message", event.get("key", "")))
        elif t == "result":
            self.status.setText(f"完成：{event.get('pages', 0)} 页 · {event.get('sizeText', '')}")
            self.progress.setValue(100)
        elif t == "error":
            self.status.setText(event.get("message", event.get("key", "错误")))
            self.logs.appendPlainText("错误：" + self.status.text())

    def on_finished(self, code):
        self.btn_cancel.setEnabled(False)
        if code != 0:
            self.status.setText(f"失败（退出码 {code}）")


def main():
    app = QApplication(sys.argv)
    w = MainWindow()
    w.show()
    sys.exit(app.exec())


if __name__ == "__main__":
    main()
