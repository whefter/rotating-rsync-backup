package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SortBackupList sorts a passed backup folder names slice in ASC (most recent one last)
// or DESC (oldest one last) direction, depending on the second parameter
func SortBackupList(backups *[]string, desc bool) {
	sort.SliceStable(*backups, func(i, j int) bool {
		iBasename := filepath.Base((*backups)[i])
		iDate, err := BackupNameToTime(iBasename)
		if err != nil {
			panic(fmt.Sprintf("DetermineLastBackup: error parsing backup folder %s into time: %v", iBasename, err))
		}

		jBasename := filepath.Base((*backups)[j])
		jDate, err := BackupNameToTime(jBasename)
		if err != nil {
			panic(fmt.Sprintf("DetermineLastBackup: error parsing backup folder %s into time: %v", jBasename, err))
		}

		if desc {
			if iDate.After(jDate) {
				return true
			}
		} else {
			if iDate.Before(jDate) {
				return true
			}
		}

		return false
	})
}

// BackupNameToTime takes a backup name as string and returns the corresponding time instance
// Returns error if name could not be parsed
func BackupNameToTime(backupName string) (time.Time, error) {
	iDate, err := time.Parse(BackupFolderTimeFormat, backupName)
	if err != nil {
		return time.Now(), err
	}

	return iDate, nil
}

// NormalizeFolderPath ensures a folder path is well-formed and ends with a slash
func NormalizeFolderPath(dirtyPath string) string {
	path := filepath.Clean(dirtyPath)

	if !strings.HasSuffix(path, "/") {
		path = fmt.Sprintf("%v/", path)
	}

	return path
}

// DetermineNewestBackupInFolder fetches all backup folder names in the target path and determines the most
// most recent one, returning its relative path relative to the target folder
func DetermineNewestBackupInFolder(options *Options, targetPath string) string {
	backups := ListBackupsInPath(options, targetPath, targetPath)

	if len(backups) > 0 {
		SortBackupList(&backups, false)
		return backups[len(backups)-1]
	}

	return ""
}
