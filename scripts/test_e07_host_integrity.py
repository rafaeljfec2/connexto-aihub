import os
import sys
import tempfile
import unittest
from types import SimpleNamespace
from unittest.mock import patch
from pathlib import Path


sys.path.insert(0, str(Path(__file__).resolve().parent))

import e07_host_integrity
import e07_host_tree


class HostIntegrityTests(unittest.TestCase):
    def test_tree_fingerprint_is_stable_when_root_is_missing(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            missing_root = Path(temporary_directory) / ".codex"

            first = e07_host_integrity.fingerprint_tree(missing_root)
            second = e07_host_integrity.fingerprint_tree(missing_root)

            self.assertEqual(first, second)
            self.assertEqual(len(first), 64)

    def test_integrity_signature_includes_access_time(self):
        metadata = {
            "st_dev": 1,
            "st_ino": 2,
            "st_mode": 0o100600,
            "st_nlink": 1,
            "st_uid": 1000,
            "st_gid": 1000,
            "st_size": 12,
            "st_mtime_ns": 13,
            "st_ctime_ns": 14,
        }
        before = SimpleNamespace(st_atime_ns=10, **metadata)
        after = SimpleNamespace(st_atime_ns=20, **metadata)

        self.assertNotEqual(
            e07_host_tree.integrity_signature(before),
            e07_host_tree.integrity_signature(after),
        )
        self.assertNotEqual(
            e07_host_tree.stat_signature(before),
            e07_host_tree.stat_signature(after),
        )

    def test_tree_fingerprint_detects_content_change_with_same_size(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory) / ".codex"
            root.mkdir()
            file_path = root / "state.json"
            file_path.write_bytes(b"token=synthetic-one")
            original_stat = file_path.stat()
            before = e07_host_integrity.fingerprint_tree(root)
            before_content = e07_host_integrity.fingerprint_file_contents(file_path)

            file_path.write_bytes(b"token=synthetic-two")
            os.utime(file_path, ns=(original_stat.st_atime_ns, original_stat.st_mtime_ns))
            after = e07_host_integrity.fingerprint_tree(root)
            after_content = e07_host_integrity.fingerprint_file_contents(file_path)

            self.assertEqual(original_stat.st_size, file_path.stat().st_size)
            self.assertEqual(original_stat.st_mode, file_path.stat().st_mode)
            self.assertEqual(original_stat.st_uid, file_path.stat().st_uid)
            self.assertEqual(original_stat.st_gid, file_path.stat().st_gid)
            self.assertEqual(original_stat.st_atime_ns, file_path.stat().st_atime_ns)
            self.assertEqual(original_stat.st_mtime_ns, file_path.stat().st_mtime_ns)
            self.assertNotEqual(before_content, after_content)
            self.assertNotEqual(before, after)

    def test_tree_fingerprint_detects_metadata_change(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory) / ".codex"
            root.mkdir()
            file_path = root / "state.json"
            file_path.write_bytes(b"stable-content")
            before = e07_host_integrity.fingerprint_tree(root)

            file_path.chmod(0o400)
            after = e07_host_integrity.fingerprint_tree(root)

            self.assertNotEqual(before, after)

    def test_tree_fingerprint_detects_entry_and_symlink_changes(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory) / ".codex"
            root.mkdir()
            first = root / "first"
            second = root / "other"
            first.write_bytes(b"one")
            second.write_bytes(b"two")
            link = root / "active"
            link.symlink_to("first")

            before = e07_host_integrity.fingerprint_tree(root)
            link.unlink()
            link.symlink_to("second")
            changed_link = e07_host_integrity.fingerprint_tree(root)
            link.unlink()
            removed_link = e07_host_integrity.fingerprint_tree(root)
            link.symlink_to("first")
            added_link = e07_host_integrity.fingerprint_tree(root)
            second.unlink()
            removed_entry = e07_host_integrity.fingerprint_tree(root)
            added = root / "third"
            added.write_bytes(b"three")
            added_entry = e07_host_integrity.fingerprint_tree(root)

            self.assertNotEqual(before, changed_link)
            self.assertNotEqual(changed_link, removed_link)
            self.assertNotEqual(removed_link, added_link)
            self.assertNotEqual(added_link, removed_entry)
            self.assertNotEqual(removed_entry, added_entry)

    def test_environment_fingerprint_covers_existing_dotenv_contents(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            environment_file = root / ".env"
            environment_file.write_bytes(b"KEY=synthetic-one\n")
            original_stat = environment_file.stat()
            (root / ".env.example").write_bytes(b"KEY=placeholder\n")
            nested = root / "nested"
            nested.mkdir()
            (nested / ".env.production").write_bytes(b"KEY=synthetic-two\n")

            before_content = e07_host_integrity.fingerprint_file_contents(environment_file)
            before = e07_host_integrity.fingerprint_environment_files(root)
            environment_file.write_bytes(b"KEY=synthetic-xxx\n")
            os.utime(environment_file, ns=(original_stat.st_atime_ns, original_stat.st_mtime_ns))
            after_content = e07_host_integrity.fingerprint_file_contents(environment_file)
            after = e07_host_integrity.fingerprint_environment_files(root)
            environment_addition = nested / ".env.local"
            environment_addition.write_bytes(b"KEY=synthetic-three\n")
            after_addition = e07_host_integrity.fingerprint_environment_files(root)
            environment_addition.unlink()
            after_removal = e07_host_integrity.fingerprint_environment_files(root)

            self.assertEqual(original_stat.st_size, environment_file.stat().st_size)
            self.assertEqual(original_stat.st_mtime_ns, environment_file.stat().st_mtime_ns)
            self.assertNotEqual(before_content, after_content)
            self.assertNotIn("KEY=synthetic-one", before)
            self.assertNotEqual(before, after)
            self.assertNotEqual(after, after_addition)
            self.assertNotEqual(after_addition, after_removal)

    def test_environment_fingerprinting_does_not_update_access_times(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            environment_file = root / ".env"
            environment_file.write_bytes(b"KEY=synthetic-value\n")
            old_time = 1_600_000_000_000_000_000
            os.utime(environment_file, ns=(old_time, old_time))
            os.utime(root, ns=(old_time, old_time))

            e07_host_integrity.fingerprint_environment_files(root)

            self.assertEqual(old_time, environment_file.stat().st_atime_ns)
            self.assertEqual(old_time, root.stat().st_atime_ns)

    def test_environment_symbolic_link_fails_closed(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            target = root / "credentials"
            target.write_bytes(b"KEY=synthetic-value\n")
            (root / ".env").symlink_to(target.name)

            with self.assertRaises(e07_host_integrity.IntegritySnapshotError):
                e07_host_integrity.fingerprint_environment_files(root)

    def test_fingerprinting_does_not_update_access_times(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory) / ".codex"
            root.mkdir()
            file_path = root / "state.json"
            file_path.write_bytes(b"synthetic-state")
            old_time = 1_600_000_000_000_000_000
            os.utime(file_path, ns=(old_time, old_time))
            os.utime(root, ns=(old_time, old_time))

            e07_host_integrity.fingerprint_tree(root)

            self.assertEqual(old_time, file_path.stat().st_atime_ns)
            self.assertEqual(old_time, root.stat().st_atime_ns)

    def test_fingerprinting_does_not_update_symbolic_link_access_time(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory) / ".codex"
            root.mkdir()
            (root / "state.json").write_bytes(b"synthetic-state")
            link = root / "active"
            link.symlink_to("state.json")
            old_time = 1_600_000_000_000_000_000
            os.utime(link, ns=(old_time, old_time), follow_symlinks=False)
            before = link.lstat().st_atime_ns

            e07_host_integrity.fingerprint_tree(root)

            self.assertEqual(before, link.lstat().st_atime_ns)

    def test_unreadable_tree_fails_closed(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory) / ".codex"
            root.mkdir()
            with self.assertRaises(e07_host_integrity.IntegritySnapshotError):
                with patch("e07_host_integrity.os.open", side_effect=PermissionError):
                    e07_host_integrity.fingerprint_tree(root)


if __name__ == "__main__":
    unittest.main()
