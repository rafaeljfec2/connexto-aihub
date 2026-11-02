import hashlib
import os
import stat
from pathlib import Path


MAX_ENTRIES = 250_000
READ_FLAGS = os.O_RDONLY | getattr(os, "O_CLOEXEC", 0) | getattr(os, "O_NOATIME", 0)
DIRECTORY_FLAGS = READ_FLAGS | getattr(os, "O_DIRECTORY", 0) | getattr(os, "O_NOFOLLOW", 0)
FILE_FLAGS = READ_FLAGS | getattr(os, "O_NOFOLLOW", 0)
CHUNK_SIZE = 1024 * 1024


class IntegritySnapshotError(Exception):
    pass


def require_read_flags():
    required = ("O_NOATIME", "O_DIRECTORY", "O_NOFOLLOW")
    if any(not getattr(os, flag, 0) for flag in required):
        raise IntegritySnapshotError("required read-only fingerprint flags are unavailable")


def stat_signature(info):
    return (
        info.st_dev,
        info.st_ino,
        info.st_mode,
        info.st_nlink,
        info.st_uid,
        info.st_gid,
        info.st_size,
        info.st_atime_ns,
        info.st_mtime_ns,
        info.st_ctime_ns,
    )


def integrity_signature(info):
    return (
        info.st_dev,
        info.st_ino,
        info.st_mode,
        info.st_nlink,
        info.st_uid,
        info.st_gid,
        info.st_size,
        info.st_atime_ns,
        info.st_mtime_ns,
        info.st_ctime_ns,
    )


def record(hasher, kind, relative, info, detail=b""):
    signature = b"" if info is None else repr(integrity_signature(info)).encode()
    fields = (kind, os.fsencode(relative), signature, detail)
    for field in fields:
        hasher.update(len(field).to_bytes(8, "big"))
        hasher.update(field)


def read_file_descriptor(file_descriptor):
    hasher = hashlib.sha256()
    while True:
        chunk = os.read(file_descriptor, CHUNK_SIZE)
        if not chunk:
            return hasher.digest()
        hasher.update(chunk)


def fingerprint_file_contents(path):
    require_read_flags()
    file_path = Path(path)
    try:
        before = file_path.lstat()
        if not stat.S_ISREG(before.st_mode):
            raise IntegritySnapshotError("fingerprint target is not a regular file")
        descriptor = os.open(str(file_path), FILE_FLAGS)
        try:
            opened = os.fstat(descriptor)
            if stat_signature(before) != stat_signature(opened):
                raise IntegritySnapshotError("fingerprint target changed while opening")
            digest = read_file_descriptor(descriptor)
            after = os.fstat(descriptor)
            if stat_signature(opened) != stat_signature(after):
                raise IntegritySnapshotError("fingerprint target changed while reading")
        finally:
            os.close(descriptor)
        if stat_signature(file_path.lstat()) != stat_signature(after):
            raise IntegritySnapshotError("fingerprint target changed after reading")
        return digest.hex()
    except IntegritySnapshotError:
        raise
    except OSError as error:
        raise IntegritySnapshotError("fingerprint target is unavailable") from error


def _fingerprint_directory(
    hasher, descriptor, relative, entries, require_symlink_target_coverage
):
    before = os.fstat(descriptor)
    names = sorted(os.listdir(descriptor), key=os.fsencode)
    for name in names:
        entries[0] += 1
        if entries[0] > MAX_ENTRIES:
            raise IntegritySnapshotError("fingerprint entry limit was exceeded")
        child_relative = name if not relative else relative + "/" + name
        info = os.stat(name, dir_fd=descriptor, follow_symlinks=False)
        mode = info.st_mode
        if stat.S_ISDIR(mode):
            child_descriptor = os.open(name, DIRECTORY_FLAGS, dir_fd=descriptor)
            try:
                opened = os.fstat(child_descriptor)
                if stat_signature(info) != stat_signature(opened):
                    raise IntegritySnapshotError("directory changed while opening")
                record(hasher, b"directory", child_relative, opened)
                _fingerprint_directory(
                    hasher,
                    child_descriptor,
                    child_relative,
                    entries,
                    require_symlink_target_coverage,
                )
                if stat_signature(os.fstat(child_descriptor)) != stat_signature(opened):
                    raise IntegritySnapshotError("directory changed while reading")
            finally:
                os.close(child_descriptor)
        elif stat.S_ISREG(mode):
            child_descriptor = os.open(name, FILE_FLAGS, dir_fd=descriptor)
            try:
                opened = os.fstat(child_descriptor)
                if stat_signature(info) != stat_signature(opened):
                    raise IntegritySnapshotError("file changed while opening")
                content_digest = read_file_descriptor(child_descriptor)
                after = os.fstat(child_descriptor)
                if stat_signature(opened) != stat_signature(after):
                    raise IntegritySnapshotError("file changed while reading")
                record(hasher, b"file", child_relative, opened, content_digest)
            finally:
                os.close(child_descriptor)
        elif stat.S_ISLNK(mode):
            if require_symlink_target_coverage:
                raise IntegritySnapshotError(
                    "symbolic-link target integrity is not verifiable"
                )
            record(hasher, b"link", child_relative, info)
        else:
            record(hasher, b"special", child_relative, info)
        after_entry = os.stat(name, dir_fd=descriptor, follow_symlinks=False)
        if stat_signature(info) != stat_signature(after_entry):
            raise IntegritySnapshotError("directory entry changed while fingerprinting")
    if stat_signature(before) != stat_signature(os.fstat(descriptor)):
        raise IntegritySnapshotError("directory changed during fingerprinting")


def fingerprint_tree(root, require_symlink_target_coverage=False):
    require_read_flags()
    root_path = Path(root)
    hasher = hashlib.sha256()
    try:
        info = root_path.lstat()
    except FileNotFoundError:
        hasher.update(b"missing-root")
        return hasher.hexdigest()
    except OSError as error:
        raise IntegritySnapshotError("fingerprint root is unavailable") from error
    if not stat.S_ISDIR(info.st_mode):
        raise IntegritySnapshotError("fingerprint root is not a directory")
    try:
        descriptor = os.open(str(root_path), DIRECTORY_FLAGS)
        try:
            opened = os.fstat(descriptor)
            if stat_signature(info) != stat_signature(opened):
                raise IntegritySnapshotError("fingerprint root changed while opening")
            record(hasher, b"root", ".", opened)
            _fingerprint_directory(
                hasher,
                descriptor,
                "",
                [0],
                require_symlink_target_coverage,
            )
            after = os.fstat(descriptor)
            if stat_signature(opened) != stat_signature(after):
                raise IntegritySnapshotError("fingerprint root changed while reading")
        finally:
            os.close(descriptor)
        if stat_signature(root_path.lstat()) != stat_signature(after):
            raise IntegritySnapshotError("fingerprint root changed after reading")
        return hasher.hexdigest()
    except IntegritySnapshotError:
        raise
    except OSError as error:
        raise IntegritySnapshotError("fingerprint tree could not be traversed") from error
