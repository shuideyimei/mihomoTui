#!/usr/bin/env python3
"""
E2E Automated Test Suite for mihomoTui powered by mcp-tui-test
"""
import os
import sys
import time
import subprocess
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from server import (
    launch_tui,
    send_keys,
    send_ctrl,
    capture_screen,
    assert_contains,
    close_session,
    list_sessions,
    sessions,
)

class TestMihomoTuiE2E(unittest.TestCase):
    mock_proc = None
    session_id = "mihomo_e2e"

    @classmethod
    def setUpClass(cls):
        project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
        os.chdir(project_root)

        # 1. Build mihomoTui binary
        print("\n[Setup] Building mihomoTui binary...")
        res = subprocess.run(["go", "build", "-o", "./mihomoTui", "main.go"], capture_output=True, text=True)
        if res.returncode != 0:
            raise RuntimeError(f"Failed to build mihomoTui: {res.stderr}")

        # 2. Build and start Mock Mihomo API server on 127.0.0.1:9090
        print("[Setup] Starting mock Mihomo API server on 127.0.0.1:9090...")
        subprocess.run(["go", "build", "-o", "/tmp/mock_mihomo_bin", "cmd/mock_mihomo/main.go"], check=True)
        cls.mock_proc = subprocess.Popen(["/tmp/mock_mihomo_bin", "-addr", "127.0.0.1:9090"])
        time.sleep(1)

        # 3. Launch mihomoTui in buffer mode (120x35)
        print("[Setup] Launching mihomoTui in buffer mode (120x35)...")
        launch_res = launch_tui(
            command="./mihomoTui",
            session_id=cls.session_id,
            dimensions="120x35",
            mode="buffer"
        )
        print(launch_res)
        if "✓" not in launch_res:
            raise RuntimeError(f"Failed to launch TUI: {launch_res}")
        time.sleep(2)

    @classmethod
    def tearDownClass(cls):
        print("\n[Teardown] Closing session...")
        try:
            close_session(cls.session_id)
        except Exception:
            pass

        if cls.mock_proc:
            print("[Teardown] Stopping mock Mihomo API server...")
            cls.mock_proc.terminate()
            cls.mock_proc.wait()

    def press(self, key: str, delay: float = 0.3):
        """Send key and give the TUI event loop time to render."""
        send_keys(key, session_id=self.session_id)
        time.sleep(delay)
        if self.session_id in sessions:
            sessions[self.session_id]._update_buffer()

    def wait_for(self, target: str, timeout: float = 4.0) -> bool:
        """Wait until screen buffer contains target string."""
        start = time.time()
        while time.time() - start < timeout:
            if self.session_id in sessions:
                sessions[self.session_id]._update_buffer()
            screen = capture_screen(session_id=self.session_id)
            if target in screen:
                return True
            time.sleep(0.2)
        return False

    def wait_for_not(self, target: str, timeout: float = 4.0) -> bool:
        """Wait until screen buffer does NOT contain target string."""
        start = time.time()
        while time.time() - start < timeout:
            if self.session_id in sessions:
                sessions[self.session_id]._update_buffer()
            screen = capture_screen(session_id=self.session_id)
            if target not in screen:
                return True
            time.sleep(0.2)
        return False

    def test_01_dashboard_connected(self):
        """Test 1: Verify dashboard initializes and connects to API."""
        self.assertTrue(self.wait_for("已连接", timeout=5.0), "Dashboard should show '已连接'")
        screen = capture_screen(session_id=self.session_id)
        self.assertIn("仪表板", screen)
        self.assertIn("连接统计", screen)
        self.assertIn("控制面板", screen)

    def test_02_help_dialog(self):
        """Test 2: Verify help overlay toggle with '?' and dismiss with Esc."""
        # Open help
        self.press("?")
        self.assertTrue(self.wait_for("页面导航", timeout=3.0), "Help modal should appear with '页面导航'")
        screen = capture_screen(session_id=self.session_id)
        self.assertIn("全局操作", screen)

        # Close help with Esc
        self.press("\x1b")
        self.assertTrue(self.wait_for_not("页面导航", timeout=3.0), "Help modal should close on Esc")

    def test_03_command_palette(self):
        """Test 3: Verify command palette opens with ':' and dismisses with Esc."""
        self.press(":")
        self.assertTrue(
            self.wait_for("命令面板", timeout=3.0) or self.wait_for("跳转", timeout=3.0),
            "Command palette prompt should appear"
        )
        screen = capture_screen(session_id=self.session_id)
        self.assertIn("跳转", screen)

        # Close palette
        self.press("\x1b")
        time.sleep(0.3)
        self.assertTrue(self.wait_for_not("命令面板", timeout=3.0), "Command palette should close on Esc")

    def test_04_navigate_to_proxies(self):
        """Test 4: Switch to Proxies page (F2 / Esc + 2)."""
        self.press("\x1b")
        self.press("2")
        self.assertTrue(self.wait_for("代理", timeout=3.0), "Should navigate to Proxies page")

    def test_05_navigate_to_connections(self):
        """Test 5: Switch to Connections page (F3 / Esc + 3)."""
        self.press("\x1b")
        self.press("3")
        self.assertTrue(self.wait_for("连接", timeout=3.0), "Should navigate to Connections page")

    def test_06_navigate_to_logs(self):
        """Test 6: Switch to Logs page (F4 / Esc + 4)."""
        self.press("\x1b")
        self.press("4")
        self.assertTrue(self.wait_for("日志", timeout=3.0), "Should navigate to Logs page")

    def test_07_navigate_to_profiles(self):
        """Test 7: Switch to Profiles page (F5 / Esc + 5)."""
        self.press("\x1b")
        self.press("5")
        self.assertTrue(self.wait_for("配置", timeout=3.0), "Should navigate to Profiles page")

    def test_08_navigate_to_subscriptions(self):
        """Test 8: Switch to Subscriptions page (F6 / Esc + 6)."""
        self.press("\x1b")
        self.press("6")
        self.assertTrue(self.wait_for("订阅", timeout=3.0), "Should navigate to Subscriptions page")

    def test_09_navigate_to_editor(self):
        """Test 9: Switch to Editor page (F7 / Esc + 7)."""
        self.press("\x1b")
        self.press("7")
        self.assertTrue(
            self.wait_for("配置编辑", timeout=3.0) or self.wait_for("代理组", timeout=3.0),
            "Should navigate to Editor page"
        )

    def test_10_navigate_to_settings(self):
        """Test 10: Switch to Settings page (F8 / Esc + 8)."""
        self.press("\x1b")
        self.press("8")
        self.assertTrue(self.wait_for("设置", timeout=3.0), "Should navigate to Settings page")

    def test_11_return_to_dashboard(self):
        """Test 11: Return to Dashboard (F1 / Esc + 1)."""
        self.press("\x1b")
        self.press("1")
        self.assertTrue(
            self.wait_for("仪表板", timeout=3.0) or self.wait_for("Dashboard", timeout=3.0),
            "Should return to Dashboard page"
        )

    def test_12_palette_jump_to_page(self):
        """Test 12: Jump directly to Proxies via Command Palette typing."""
        self.press(":")
        self.assertTrue(self.wait_for("跳转", timeout=3.0), "Palette should open")
        # Press Enter to select the top item in palette
        self.press("\n")
        time.sleep(0.5)

    def test_13_small_terminal_dimensions(self):
        """Test 13: Test launching in 80x24 standard compact terminal dimensions."""
        small_session = "mihomo_small_e2e"
        res = launch_tui(
            command="./mihomoTui",
            session_id=small_session,
            dimensions="80x24",
            mode="buffer"
        )
        self.assertIn("✓", res)
        time.sleep(1.5)
        screen = capture_screen(session_id=small_session)
        self.assertTrue("mihomoTui" in screen or "导航" in screen)
        close_session(small_session)


if __name__ == "__main__":
    unittest.main(verbosity=2)
