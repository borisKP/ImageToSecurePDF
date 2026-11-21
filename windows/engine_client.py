# -*- coding: utf-8 -*-
"""共享 Go 引擎的 Python 客户端。

只负责启动 helper、按行读取 JSON 事件、取消任务。
不依赖 PySide6，便于单独测试。
"""
from __future__ import annotations

import json
import os
import subprocess
import threading
from typing import Callable


class EngineClient:
    def __init__(self, helper_path: str, on_event: Callable[[dict], None],
                 on_done: Callable[[int], None] | None = None):
        self.helper_path = helper_path
        self.on_event = on_event
        self.on_done = on_done
        self.proc: subprocess.Popen | None = None
        self._thread: threading.Thread | None = None

    def is_running(self) -> bool:
        return self.proc is not None and self.proc.poll() is None

    def run(self, args: list[str]) -> None:
        if not os.path.isfile(self.helper_path):
            raise FileNotFoundError(f"找不到共享引擎：{self.helper_path}")
        if self.is_running():
            raise RuntimeError("已有任务在运行")
        self.proc = subprocess.Popen(
            [self.helper_path, *args],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            encoding="utf-8",
            errors="replace",
            bufsize=1,
        )
        self._thread = threading.Thread(target=self._read, daemon=True)
        self._thread.start()

    def _read(self) -> None:
        proc = self.proc
        if proc is None or proc.stdout is None:
            return
        try:
            for line in proc.stdout:
                line = line.strip()
                if not line:
                    continue
                try:
                    event = json.loads(line)
                except json.JSONDecodeError:
                    event = {"type": "log", "message": line}
                self.on_event(event)
        finally:
            code = proc.wait()
            if self.on_done:
                self.on_done(code)

    def cancel(self) -> None:
        if self.proc and self.proc.stdin and self.is_running():
            try:
                self.proc.stdin.write(json.dumps({"type": "cancel"}) + "\n")
                self.proc.stdin.flush()
            except (BrokenPipeError, OSError):
                pass

    def terminate(self) -> None:
        if self.is_running() and self.proc:
            self.proc.terminate()
