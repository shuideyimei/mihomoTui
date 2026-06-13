package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// WriteFileAtomic replaces path atomically while preserving the existing file
// mode and ownership. When the destination is not writable, it falls back to a
// short sudo install+rename sequence.
func WriteFileAtomic(path string, data []byte, defaultPerm os.FileMode) error {
	realPath, err := resolveWritePath(path)
	if err != nil {
		return err
	}

	perm := defaultPerm
	var owner fileOwner
	if info, statErr := os.Stat(realPath); statErr == nil {
		perm = info.Mode().Perm()
		owner = ownerFromFileInfo(info)
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("读取目标文件状态失败: %w", statErr)
	}

	if err := writeDirectAtomic(realPath, data, perm, owner); err == nil {
		return nil
	} else if runtime.GOOS == "windows" || !errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("原子写入文件失败: %w", err)
	}

	if err := writeSudoAtomic(realPath, data, perm, owner); err != nil {
		return fmt.Errorf("原子写入文件失败 (需要 sudo 权限): %w", err)
	}
	return nil
}

func resolveWritePath(path string) (string, error) {
	realPath, err := filepath.EvalSymlinks(path)
	if err == nil {
		return realPath, nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("解析文件路径失败: %w", err)
	}

	dir, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", fmt.Errorf("解析文件目录失败: %w", err)
	}
	return filepath.Join(dir, filepath.Base(path)), nil
}

func writeDirectAtomic(path string, data []byte, perm os.FileMode, owner fileOwner) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := applyOwner(tmpPath, owner); err != nil {
		return err
	}
	if err := replaceFile(tmpPath, path); err != nil {
		return err
	}

	// Best effort: persist the directory entry as well as the file contents.
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

func writeSudoAtomic(path string, data []byte, perm os.FileMode, owner fileOwner) error {
	source, err := os.CreateTemp("", "mihomoTui-write-*")
	if err != nil {
		return err
	}
	sourcePath := source.Name()
	defer os.Remove(sourcePath)

	if _, err := source.Write(data); err != nil {
		source.Close()
		return err
	}
	if err := source.Sync(); err != nil {
		source.Close()
		return err
	}
	if err := source.Close(); err != nil {
		return err
	}

	stage := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".mihomoTui-"+randomSuffix())
	defer exec.Command("sudo", "rm", "-f", "--", stage).Run()

	args := []string{"install", "-m", fmt.Sprintf("%o", perm)}
	args = appendOwnerInstallArgs(args, owner)
	args = append(args, "--", sourcePath, stage)
	if output, err := exec.Command("sudo", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("安装临时文件失败: %s", string(output))
	}
	if output, err := exec.Command("sudo", "mv", "-f", "--", stage, path).CombinedOutput(); err != nil {
		return fmt.Errorf("替换目标文件失败: %s", string(output))
	}
	return nil
}

func randomSuffix() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(buf[:])
}
