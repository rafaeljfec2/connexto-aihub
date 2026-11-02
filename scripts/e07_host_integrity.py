import os
import pwd
import sys
from pathlib import Path

from e07_host_environment import fingerprint_environment_files
from e07_host_tree import (
    IntegritySnapshotError,
    fingerprint_file_contents,
    fingerprint_tree,
)


def snapshot(host_home, repository_root):
    return (
        fingerprint_tree(
            Path(host_home) / ".codex",
            require_symlink_target_coverage=True,
        ),
        fingerprint_environment_files(repository_root),
    )


def main(arguments):
    if len(arguments) != 2 or arguments[0] != "snapshot":
        return 2
    try:
        account_home = pwd.getpwuid(os.getuid()).pw_dir
        codex_digest, environment_digest = snapshot(account_home, arguments[1])
    except (IntegritySnapshotError, KeyError, OSError):
        sys.stderr.write("E07 host integrity snapshot unavailable\n")
        return 1
    sys.stdout.write(codex_digest + " " + environment_digest + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
