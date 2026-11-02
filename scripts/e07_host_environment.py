import hashlib
import os
import stat
from pathlib import Path, PurePath

from e07_host_tree import (
    MAX_ENTRIES,
    DIRECTORY_FLAGS,
    IntegritySnapshotError,
    stat_signature as _stat_signature,
    record as _record,
    fingerprint_file_contents,
    require_read_flags as _require_read_flags,
)


def _is_environment_name(name):
    filename = PurePath(name).name
    return filename == ".env" or (
        filename.startswith(".env.") and not filename.endswith(".example")
    )


def _collect_environment_paths(descriptor, relative, paths, entries):
    before = os.fstat(descriptor)
    names = sorted(os.listdir(descriptor), key=os.fsencode)
    for name in names:
        entries[0] += 1
        if entries[0] > MAX_ENTRIES:
            raise IntegritySnapshotError("environment discovery entry limit was exceeded")
        info = os.stat(name, dir_fd=descriptor, follow_symlinks=False)
        child_relative = name if not relative else relative + "/" + name
        if stat.S_ISDIR(info.st_mode):
            if name == ".git":
                continue
            child_descriptor = os.open(name, DIRECTORY_FLAGS, dir_fd=descriptor)
            try:
                if _stat_signature(info) != _stat_signature(os.fstat(child_descriptor)):
                    raise IntegritySnapshotError("environment directory changed while opening")
                _collect_environment_paths(child_descriptor, child_relative, paths, entries)
                if _stat_signature(info) != _stat_signature(os.fstat(child_descriptor)):
                    raise IntegritySnapshotError("environment directory changed during discovery")
            finally:
                os.close(child_descriptor)
        elif stat.S_ISLNK(info.st_mode):
            if _is_environment_name(name):
                paths.append(child_relative)
            else:
                raise IntegritySnapshotError("project links prevent safe environment discovery")
        elif _is_environment_name(name):
            paths.append(child_relative)
    if _stat_signature(before) != _stat_signature(os.fstat(descriptor)):
        raise IntegritySnapshotError("environment directory changed during discovery")


def _fingerprint_environment_entry(repository, relative, hasher):
    path = repository.joinpath(*relative.split("/"))
    try:
        info = path.lstat()
        if stat.S_ISLNK(info.st_mode):
            raise IntegritySnapshotError("environment symbolic links cannot be fingerprinted without access-time changes")
        if not stat.S_ISREG(info.st_mode):
            raise IntegritySnapshotError("environment entry is not a regular file")
        content_digest = bytes.fromhex(fingerprint_file_contents(path))
        after = path.lstat()
        if _stat_signature(info) != _stat_signature(after):
            raise IntegritySnapshotError("environment entry changed while reading")
        _record(hasher, b"environment-file", relative, info, content_digest)
    except IntegritySnapshotError:
        raise
    except OSError as error:
        raise IntegritySnapshotError("environment entry is unavailable") from error


def fingerprint_environment_files(repository_root):
    _require_read_flags()
    repository = Path(repository_root)
    try:
        root_info = repository.lstat()
        if not stat.S_ISDIR(root_info.st_mode):
            raise IntegritySnapshotError("repository root is unavailable")
        root_descriptor = os.open(str(repository), DIRECTORY_FLAGS)
        try:
            if _stat_signature(root_info) != _stat_signature(os.fstat(root_descriptor)):
                raise IntegritySnapshotError("repository root changed while opening")
            paths = []
            _collect_environment_paths(root_descriptor, "", paths, [0])
        finally:
            os.close(root_descriptor)
        if _stat_signature(root_info) != _stat_signature(repository.lstat()):
            raise IntegritySnapshotError("repository root changed during discovery")
        hasher = hashlib.sha256()
        for relative in sorted(paths, key=os.fsencode):
            _record(hasher, b"environment-path", relative, None)
            _fingerprint_environment_entry(repository, relative, hasher)
        return hasher.hexdigest()
    except IntegritySnapshotError:
        raise
    except OSError as error:
        raise IntegritySnapshotError("environment files could not be fingerprinted") from error
