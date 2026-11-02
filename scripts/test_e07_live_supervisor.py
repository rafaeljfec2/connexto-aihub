import unittest
from pathlib import Path


class LiveSupervisorTests(unittest.TestCase):
    def test_supervisor_does_not_fingerprint_host_codex_or_environment_files(self):
        script_path = Path(__file__).with_name("e07-live-supervisor.sh")
        script = script_path.read_text(encoding="utf-8")

        self.assertNotIn("e07_host_integrity.py", script)
        self.assertNotIn("host_codex_before", script)
        self.assertNotIn("host_env_before", script)

    def test_supervisor_keeps_private_codex_mount_and_runtime(self):
        script_path = Path(__file__).with_name("e07-live-supervisor.sh")
        script = script_path.read_text(encoding="utf-8")

        self.assertIn("--tmpfs /home", script)
        self.assertIn("--tmpfs /home/e07-homologation/.codex", script)
        self.assertIn('CODEX_HOME="$private_root/codex-home"', script)


if __name__ == "__main__":
    unittest.main()
