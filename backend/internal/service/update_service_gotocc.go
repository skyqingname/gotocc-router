package service

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	archivepath "path"
	"path/filepath"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/releasechannel"
)

type stagedReleaseFile struct {
	archivePath string
	stagedPath  string
	targetPath  string
	mode        os.FileMode
}

type installedReleaseFile struct {
	targetPath  string
	backupPath  string
	hadOriginal bool
}

type rollbackReleaseFile struct {
	targetPath string
	backupPath string
	currentTmp string
	hadCurrent bool
}

func (s *UpdateService) extractReleaseFiles(archivePath, tempDir, exePath string) ([]stagedReleaseFile, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var reader io.Reader = f

	if strings.HasSuffix(archivePath, ".gz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer func() { _ = gzr.Close() }()
		reader = gzr
	}

	if !strings.Contains(archivePath, ".tar") {
		return nil, fmt.Errorf("unsupported release archive: %s", filepath.Base(archivePath))
	}

	desired := make(map[string]releasechannel.RuntimeFile, len(releasechannel.RuntimeFiles))
	for _, runtimeFile := range releasechannel.RuntimeFiles {
		desired[runtimeFile.Path] = runtimeFile
	}
	found := make(map[string]stagedReleaseFile, len(desired)+1)
	stageRoot := filepath.Join(tempDir, "release")
	tr := tar.NewReader(reader)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		name := archivepath.Clean(strings.TrimPrefix(filepath.ToSlash(hdr.Name), "./"))
		if name == "." || archivepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			return nil, fmt.Errorf("path traversal attempt detected: %s", hdr.Name)
		}

		archiveName := ""
		mode := os.FileMode(0o644)
		if archivepath.Base(name) == "sub2api" || archivepath.Base(name) == "sub2api.exe" {
			archiveName = "sub2api"
			mode = 0o755
		} else if _, ok := desired[name]; ok {
			archiveName = name
		}
		if archiveName == "" {
			continue
		}
		if _, exists := found[archiveName]; exists {
			return nil, fmt.Errorf("release archive repeats %s", archiveName)
		}
		if hdr.Size < 0 || hdr.Size > maxDownloadSize {
			return nil, fmt.Errorf("release file %s has invalid size %d", archiveName, hdr.Size)
		}

		stagedPath := filepath.Join(stageRoot, filepath.FromSlash(archiveName))
		if err := os.MkdirAll(filepath.Dir(stagedPath), 0o755); err != nil {
			return nil, err
		}
		out, err := os.OpenFile(stagedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return nil, err
		}
		written, copyErr := io.Copy(out, io.LimitReader(tr, hdr.Size+1))
		closeErr := out.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if written != hdr.Size {
			return nil, fmt.Errorf("release file %s size mismatch", archiveName)
		}

		targetPath := exePath
		if archiveName != "sub2api" {
			targetPath = filepath.Join(filepath.Dir(exePath), filepath.FromSlash(archiveName))
		}
		found[archiveName] = stagedReleaseFile{
			archivePath: archiveName,
			stagedPath:  stagedPath,
			targetPath:  targetPath,
			mode:        mode,
		}
	}

	binary, ok := found["sub2api"]
	if !ok {
		return nil, fmt.Errorf("binary not found in archive")
	}
	files := []stagedReleaseFile{binary}
	for _, runtimeFile := range releasechannel.RuntimeFiles {
		staged, ok := found[runtimeFile.Path]
		if !ok {
			if runtimeFile.Required {
				return nil, fmt.Errorf("required runtime file not found in archive: %s", runtimeFile.Path)
			}
			continue
		}
		files = append(files, staged)
	}
	return files, nil
}

func installReleaseFiles(files []stagedReleaseFile) error {
	installed := make([]installedReleaseFile, 0, len(files))
	for _, file := range files {
		if err := os.Chmod(file.stagedPath, file.mode); err != nil {
			return restoreInstalledFiles(installed, fmt.Errorf("chmod %s failed: %w", file.archivePath, err))
		}
		if err := os.MkdirAll(filepath.Dir(file.targetPath), 0o755); err != nil {
			return restoreInstalledFiles(installed, fmt.Errorf("create target directory for %s failed: %w", file.archivePath, err))
		}

		backupPath := file.targetPath + ".backup"
		if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
			return restoreInstalledFiles(installed, fmt.Errorf("remove old backup for %s failed: %w", file.archivePath, err))
		}
		hadOriginal := true
		if err := os.Rename(file.targetPath, backupPath); err != nil {
			if !os.IsNotExist(err) {
				return restoreInstalledFiles(installed, fmt.Errorf("backup %s failed: %w", file.archivePath, err))
			}
			hadOriginal = false
		}

		if err := os.Rename(file.stagedPath, file.targetPath); err != nil {
			if hadOriginal {
				_ = os.Rename(backupPath, file.targetPath)
			}
			return restoreInstalledFiles(installed, fmt.Errorf("replace %s failed: %w", file.archivePath, err))
		}
		installed = append(installed, installedReleaseFile{
			targetPath:  file.targetPath,
			backupPath:  backupPath,
			hadOriginal: hadOriginal,
		})
	}
	return nil
}

func restoreInstalledFiles(installed []installedReleaseFile, cause error) error {
	for i := len(installed) - 1; i >= 0; i-- {
		file := installed[i]
		if err := os.Remove(file.targetPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("%w; restore remove failed for %s: %v", cause, file.targetPath, err)
		}
		if file.hadOriginal {
			if err := os.Rename(file.backupPath, file.targetPath); err != nil {
				return fmt.Errorf("%w; restore failed for %s: %v", cause, file.targetPath, err)
			}
		}
	}
	return cause
}

func restoreReleaseBackups(targets []string) error {
	files := make([]rollbackReleaseFile, 0, len(targets))
	for _, targetPath := range targets {
		backupPath := targetPath + ".backup"
		if _, err := os.Stat(backupPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("inspect rollback backup failed: %w", err)
		}
		files = append(files, rollbackReleaseFile{
			targetPath: targetPath,
			backupPath: backupPath,
			currentTmp: targetPath + ".rollback-current",
		})
	}

	restored := make([]rollbackReleaseFile, 0, len(files))
	for _, file := range files {
		if err := os.Remove(file.currentTmp); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("prepare rollback failed for %s: %w", file.targetPath, err)
		}
		if err := os.Rename(file.targetPath, file.currentTmp); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("stage current file for rollback failed: %w", err)
			}
		} else {
			file.hadCurrent = true
		}
		if err := os.Rename(file.backupPath, file.targetPath); err != nil {
			if file.hadCurrent {
				_ = os.Rename(file.currentTmp, file.targetPath)
			}
			return undoRestoredBackups(restored, fmt.Errorf("rollback failed for %s: %w", file.targetPath, err))
		}
		restored = append(restored, file)
	}

	for _, file := range restored {
		if !file.hadCurrent {
			continue
		}
		if err := os.Remove(file.currentTmp); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("rollback completed but cleanup failed for %s: %w", file.targetPath, err)
		}
	}
	return nil
}

func undoRestoredBackups(restored []rollbackReleaseFile, cause error) error {
	for i := len(restored) - 1; i >= 0; i-- {
		file := restored[i]
		if err := os.Rename(file.targetPath, file.backupPath); err != nil {
			return fmt.Errorf("%w; rollback recovery failed for %s: %v", cause, file.targetPath, err)
		}
		if file.hadCurrent {
			if err := os.Rename(file.currentTmp, file.targetPath); err != nil {
				return fmt.Errorf("%w; rollback recovery failed for %s: %v", cause, file.targetPath, err)
			}
		}
	}
	return cause
}
