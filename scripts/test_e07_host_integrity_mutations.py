import os
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


sys.path.insert(0, str(Path(__file__).resolve().parent))

import e07_host_environment
import e07_host_integrity
import e07_host_tree


class HostIntegrityMutationTests(unittest.TestCase):
    def test_host_snapshot_fails_closed_when_codex_tree_has_symbolic_links(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            host_home = root / "home"
            repository = root / "repository"
            codex_tree = host_home / ".codex"
            codex_tree.mkdir(parents=True)
            repository.mkdir()
            external_target = root / "external-target"
            external_target.write_bytes(b"synthetic-target")
            (codex_tree / "external-link").symlink_to(external_target)

            with self.assertRaises(e07_host_tree.IntegritySnapshotError):
                e07_host_integrity.snapshot(host_home, repository)

    def test_tree_fingerprint_detects_file_and_directory_renames(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory) / ".codex"
            root.mkdir()
            file_path = root / "state.json"
            file_path.write_bytes(b"stable-state")

            before_file_rename = e07_host_integrity.fingerprint_tree(root)
            file_path.rename(root / "renamed-state.json")
            after_file_rename = e07_host_integrity.fingerprint_tree(root)

            nested_directory = root / "nested"
            nested_directory.mkdir()
            (nested_directory / "state.json").write_bytes(b"nested-state")
            before_directory_rename = e07_host_integrity.fingerprint_tree(root)
            nested_directory.rename(root / "renamed-nested")
            after_directory_rename = e07_host_integrity.fingerprint_tree(root)

            self.assertNotEqual(before_file_rename, after_file_rename)
            self.assertNotEqual(before_directory_rename, after_directory_rename)

    def test_tree_fingerprint_fails_when_file_is_replaced_during_read(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_root = Path(temporary_directory)
            root = temporary_root / ".codex"
            root.mkdir()
            file_path = root / "state.json"
            file_path.write_bytes(b"opened-state")
            replacement = temporary_root / "replacement.json"
            replacement.write_bytes(b"replacement")
            read_file = e07_host_tree.read_file_descriptor

            def replace_after_read(file_descriptor):
                digest = read_file(file_descriptor)
                os.replace(replacement, file_path)
                return digest

            with patch(
                "e07_host_tree.read_file_descriptor",
                side_effect=replace_after_read,
            ):
                with self.assertRaises(e07_host_tree.IntegritySnapshotError):
                    e07_host_integrity.fingerprint_tree(root)

    def test_tree_fingerprint_fails_when_directory_is_replaced_during_read(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_root = Path(temporary_directory)
            root = temporary_root / ".codex"
            root.mkdir()
            nested_directory = root / "nested"
            nested_directory.mkdir()
            (nested_directory / "state.json").write_bytes(b"opened-state")
            replacement = temporary_root / "replacement-directory"
            replacement.mkdir()
            (replacement / "state.json").write_bytes(b"replacement")
            moved_directory = temporary_root / "moved-directory"
            read_file = e07_host_tree.read_file_descriptor

            def replace_after_read(file_descriptor):
                digest = read_file(file_descriptor)
                os.replace(nested_directory, moved_directory)
                os.replace(replacement, nested_directory)
                return digest

            with patch(
                "e07_host_tree.read_file_descriptor",
                side_effect=replace_after_read,
            ):
                with self.assertRaises(e07_host_tree.IntegritySnapshotError):
                    e07_host_integrity.fingerprint_tree(root)

    def test_tree_fingerprint_fails_when_entry_limit_is_exceeded(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory) / ".codex"
            root.mkdir()
            (root / "first").write_bytes(b"one")
            (root / "second").write_bytes(b"two")

            with patch("e07_host_tree.MAX_ENTRIES", 1):
                with self.assertRaises(e07_host_tree.IntegritySnapshotError):
                    e07_host_integrity.fingerprint_tree(root)

    def test_environment_fingerprint_detects_metadata_change(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            environment_file = root / ".env"
            environment_file.write_bytes(b"KEY=synthetic-value\n")
            before = e07_host_integrity.fingerprint_environment_files(root)

            environment_file.chmod(0o400)
            after = e07_host_integrity.fingerprint_environment_files(root)

            self.assertNotEqual(before, after)

    def test_environment_fingerprint_fails_when_file_cannot_be_read(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            (root / ".env").write_bytes(b"KEY=synthetic-value\n")

            with patch(
                "e07_host_environment.fingerprint_file_contents",
                side_effect=e07_host_tree.IntegritySnapshotError(
                    "synthetic environment read failure"
                ),
            ):
                with self.assertRaises(e07_host_tree.IntegritySnapshotError):
                    e07_host_integrity.fingerprint_environment_files(root)


if __name__ == "__main__":
    unittest.main()
