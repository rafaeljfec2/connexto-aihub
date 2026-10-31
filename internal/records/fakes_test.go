package records

import (
	"io"
	"os"
)

type hookOps struct {
	osFileOps

	openAppendHook    func(string) (appendFile, error)
	openExclusiveHook func(string) (appendFile, error)
	openTempHook      func(string) (appendFile, error)
	openReadHook      func(string) (io.ReadCloser, error)
	renameHook        func(string, string) error
	removeHook        func(string) error
	syncDirHook       func(string) error
	mkdirAllHook      func(string, os.FileMode) error
	chmodHook         func(string, os.FileMode) error
	readDirHook       func(string) ([]os.DirEntry, error)
}

func (o *hookOps) mkdirAll(path string, perm os.FileMode) error {
	if o.mkdirAllHook != nil {
		return o.mkdirAllHook(path, perm)
	}

	return o.osFileOps.mkdirAll(path, perm)
}

func (o *hookOps) chmod(path string, perm os.FileMode) error {
	if o.chmodHook != nil {
		return o.chmodHook(path, perm)
	}

	return o.osFileOps.chmod(path, perm)
}

func (o *hookOps) openAppend(path string) (appendFile, error) {
	if o.openAppendHook != nil {
		return o.openAppendHook(path)
	}

	return o.osFileOps.openAppend(path)
}

func (o *hookOps) openExclusive(path string) (appendFile, error) {
	if o.openExclusiveHook != nil {
		return o.openExclusiveHook(path)
	}

	return o.osFileOps.openExclusive(path)
}

func (o *hookOps) openTemp(path string) (appendFile, error) {
	if o.openTempHook != nil {
		return o.openTempHook(path)
	}

	return o.osFileOps.openTemp(path)
}

func (o *hookOps) openRead(path string) (io.ReadCloser, error) {
	if o.openReadHook != nil {
		return o.openReadHook(path)
	}

	return o.osFileOps.openRead(path)
}

func (o *hookOps) rename(oldPath, newPath string) error {
	if o.renameHook != nil {
		return o.renameHook(oldPath, newPath)
	}

	return o.osFileOps.rename(oldPath, newPath)
}

func (o *hookOps) remove(path string) error {
	if o.removeHook != nil {
		return o.removeHook(path)
	}

	return o.osFileOps.remove(path)
}

func (o *hookOps) syncDir(dir string) error {
	if o.syncDirHook != nil {
		return o.syncDirHook(dir)
	}

	return o.osFileOps.syncDir(dir)
}

func (o *hookOps) readDir(path string) ([]os.DirEntry, error) {
	if o.readDirHook != nil {
		return o.readDirHook(path)
	}

	return o.osFileOps.readDir(path)
}

type hookFile struct {
	*os.File

	writeHook    func([]byte) (int, error)
	syncHook     func() error
	closeHook    func() error
	truncateHook func(int64) error
}

func (f *hookFile) Write(data []byte) (int, error) {
	if f.writeHook != nil {
		return f.writeHook(data)
	}

	return f.File.Write(data)
}

func (f *hookFile) Sync() error {
	if f.syncHook != nil {
		return f.syncHook()
	}

	return f.File.Sync()
}

func (f *hookFile) Close() error {
	if f.closeHook != nil {
		return f.closeHook()
	}

	return f.File.Close()
}

func (f *hookFile) Truncate(size int64) error {
	if f.truncateHook != nil {
		return f.truncateHook(size)
	}

	return f.File.Truncate(size)
}

func wrapAppend(ops *hookOps, configure func(*hookFile)) {
	wrap := func(base func(string) (appendFile, error), fallback func(string) (appendFile, error)) func(string) (appendFile, error) {
		return func(path string) (appendFile, error) {
			var file appendFile
			var err error
			if base != nil {
				file, err = base(path)
			} else {
				file, err = fallback(path)
			}
			if err != nil {
				return nil, err
			}

			concrete, ok := file.(*os.File)
			if !ok {
				return file, nil
			}

			wrapped := &hookFile{File: concrete}
			configure(wrapped)
			return wrapped, nil
		}
	}

	ops.openAppendHook = wrap(ops.openAppendHook, ops.osFileOps.openAppend)
	ops.openExclusiveHook = wrap(ops.openExclusiveHook, ops.osFileOps.openExclusive)
	ops.openTempHook = wrap(ops.openTempHook, ops.osFileOps.openTemp)
}

func blockingWriteHook(block <-chan struct{}, entered chan<- struct{}) func(*hookFile) {
	return func(file *hookFile) {
		file.writeHook = func(data []byte) (int, error) {
			select {
			case entered <- struct{}{}:
			default:
			}
			<-block
			return file.File.Write(data)
		}
	}
}
