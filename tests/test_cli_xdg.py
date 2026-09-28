#!/usr/bin/env python3
"""
Python E2E integration test suite for the Redis Multi-Context CLI tool.
Uses standard library unittest and temporary isolated XDG_CONFIG_HOME paths.
"""

import json
import os
import pathlib
import subprocess
import sys
import tempfile
import unittest

PROJECT_ROOT = pathlib.Path(__file__).resolve().parent.parent
BINARY_PATH = PROJECT_ROOT / "bin" / "redis"


class TestRedisCLIWithXDG(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        # Build binary if not already present
        if not BINARY_PATH.exists():
            cmd = ["go", "build", "-o", str(BINARY_PATH), "main.go"]
            res = subprocess.run(cmd, cwd=PROJECT_ROOT, capture_output=True, text=True)
            if res.returncode != 0:
                raise RuntimeError(f"Failed to build redis binary: {res.stderr}")

    def setUp(self):
        # Isolated sandbox per test
        self.tmp_dir = tempfile.TemporaryDirectory()
        self.env = os.environ.copy()
        self.env["XDG_CONFIG_HOME"] = self.tmp_dir.name

    def tearDown(self):
        self.tmp_dir.cleanup()

    def run_cli(self, args, stdin=None, check=False):
        cmd = [str(BINARY_PATH)] + args
        return subprocess.run(
            cmd,
            env=self.env,
            cwd=PROJECT_ROOT,
            input=stdin,
            text=True,
            capture_output=True,
            check=check,
        )

    def test_01_context_lifecycle(self):
        # Create contexts
        res = self.run_cli(["context", "create", "test-c1", "--host", "127.0.0.1", "--port", "6379", "--password", "secretA"])
        self.assertEqual(res.returncode, 0)
        self.assertIn('Context "test-c1" created.', res.stdout)

        res = self.run_cli(["context", "create", "test-c2", "--host", "127.0.0.1", "--port", "6380"])
        self.assertEqual(res.returncode, 0)

        # Verify initial active context
        res = self.run_cli(["context", "current"])
        self.assertEqual(res.returncode, 0)
        self.assertEqual(res.stdout.strip(), "test-c1")

        # Switch context
        res = self.run_cli(["context", "use", "test-c2"])
        self.assertEqual(res.returncode, 0)
        self.assertIn('Switched to context "test-c2".', res.stdout)

        res = self.run_cli(["context", "current"])
        self.assertEqual(res.stdout.strip(), "test-c2")

        # List contexts as JSON
        res = self.run_cli(["context", "ls", "--json"])
        self.assertEqual(res.returncode, 0)
        contexts = json.loads(res.stdout)
        self.assertEqual(len(contexts), 2)
        active_ctx = next(c for c in contexts if c["active"])
        self.assertEqual(active_ctx["name"], "test-c2")

        # Export (sanitized)
        res = self.run_cli(["context", "export"])
        self.assertEqual(res.returncode, 0)
        self.assertNotIn("secretA", res.stdout)
        self.assertIn("${TEST_C1_PASSWORD}", res.stdout)

        # Delete context
        res = self.run_cli(["context", "delete", "test-c1"])
        self.assertEqual(res.returncode, 0)
        self.assertIn('Context "test-c1" deleted.', res.stdout)

    def test_02_data_commands(self):
        # Create context
        self.run_cli(["context", "create", "local-data", "--host", "127.0.0.1", "--port", "6379"])

        # Set & Get string
        res = self.run_cli(["set", "test:py:str", "Hello from Python", "--ttl", "60s"])
        self.assertEqual(res.returncode, 0)
        self.assertIn("OK", res.stdout)

        res = self.run_cli(["get", "test:py:str"])
        self.assertEqual(res.returncode, 0)
        self.assertEqual(res.stdout.strip(), "Hello from Python")

        # Set & Get JSON
        json_payload = '{"service":"auth","active":true,"code":101}'
        res = self.run_cli(["set", "test:py:json", json_payload, "--ttl", "60s"])
        self.assertEqual(res.returncode, 0)

        res = self.run_cli(["get", "test:py:json"])
        self.assertEqual(res.returncode, 0)
        parsed = json.loads(res.stdout)
        self.assertEqual(parsed["service"], "auth")
        self.assertEqual(parsed["code"], 101)

        # Show table
        res = self.run_cli(["show", "--pattern", "test:py:*"])
        self.assertEqual(res.returncode, 0)
        self.assertIn("test:py:str", res.stdout)
        self.assertIn("test:py:json", res.stdout)

        # Show JSON compact with field filtering
        res = self.run_cli(["show", "--pattern", "test:py:*", "--json", "--compact", "--fields=key,type"])
        self.assertEqual(res.returncode, 0)
        self.assertNotIn("\n", res.stdout.strip())
        items = json.loads(res.stdout.strip())
        self.assertGreaterEqual(len(items), 2)
        for it in items:
            self.assertIn("key", it)
            self.assertIn("type", it)
            self.assertNotIn("memory_bytes", it)

        # Inspect
        res = self.run_cli(["inspect", "test:py:json"])
        self.assertEqual(res.returncode, 0)
        self.assertIn("test:py:json", res.stdout)
        self.assertIn("STRING", res.stdout)
        self.assertIn('"service": "auth"', res.stdout)

        # Clean up
        self.run_cli(["del", "test:py:*", "--force"])

    def test_03_summary_health(self):
        self.run_cli(["context", "create", "local-sum", "--host", "127.0.0.1", "--port", "6379"])

        res = self.run_cli(["summary", "--json", "--compact"])
        self.assertEqual(res.returncode, 0)

        raw_bytes = res.stdout.strip().encode("utf-8")
        self.assertLess(len(raw_bytes), 500, f"Summary JSON too large: {len(raw_bytes)} bytes")

        summary = json.loads(res.stdout)
        self.assertEqual(summary["context"], "local-sum")
        self.assertIn("version", summary)
        self.assertIn("role", summary)
        self.assertIn("memory", summary)
        self.assertIn("hit_rate_pct", summary)

    def test_04_safety_guardrails(self):
        self.run_cli(["context", "create", "local-safe", "--host", "127.0.0.1", "--port", "6379"])
        self.run_cli(["set", "test:safe:guarded", "value"])

        # Non-TTY pattern delete without --force must exit with code 4
        res = self.run_cli(["del", "test:safe:*"], stdin="")
        self.assertEqual(res.returncode, 4)
        self.assertIn("requires --force in non-interactive mode", res.stderr + res.stdout)

        # Dry run simulation
        res = self.run_cli(["del", "test:safe:*", "--dry-run"])
        self.assertEqual(res.returncode, 0)
        self.assertIn("[DRY-RUN]", res.stdout)
        self.assertIn("0 keys deleted", res.stdout)

        # Real deletion with --force
        res = self.run_cli(["del", "test:safe:*", "--force"])
        self.assertEqual(res.returncode, 0)
        self.assertIn("Successfully unlinked", res.stdout)

    def test_05_non_tty_explore_rejection(self):
        # explore command in non-interactive environment must exit with code 5
        res = self.run_cli(["explore"], stdin="")
        self.assertEqual(res.returncode, 5)
        self.assertIn("interactive TTY", res.stderr + res.stdout)


if __name__ == "__main__":
    unittest.main()
