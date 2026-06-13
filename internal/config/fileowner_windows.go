//go:build windows

package config

import "os"

type fileOwner struct{}

func ownerFromFileInfo(os.FileInfo) fileOwner {
	return fileOwner{}
}

func applyOwner(string, fileOwner) error {
	return nil
}

func appendOwnerInstallArgs(args []string, _ fileOwner) []string {
	return args
}
