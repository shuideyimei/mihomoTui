//go:build !windows

package config

import (
	"fmt"
	"os"
	"syscall"
)

type fileOwner struct {
	uid int
	gid int
	ok  bool
}

func ownerFromFileInfo(info os.FileInfo) fileOwner {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileOwner{}
	}
	return fileOwner{uid: int(stat.Uid), gid: int(stat.Gid), ok: true}
}

func applyOwner(path string, owner fileOwner) error {
	if !owner.ok {
		return nil
	}
	return os.Chown(path, owner.uid, owner.gid)
}

func appendOwnerInstallArgs(args []string, owner fileOwner) []string {
	if !owner.ok {
		return args
	}
	return append(args, "-o", fmt.Sprint(owner.uid), "-g", fmt.Sprint(owner.gid))
}
