package main

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"sort"
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

func printStdout(stdout io.ReadCloser, stash *[]string) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		*stash = append(*stash, line)
		Log.Debug.Printf("-- STDOUT --: %s", line)
	}
}

func printStderr(stderr io.ReadCloser, stash *[]string) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		*stash = append(*stash, line)
		Log.Debug.Printf("-- STDERR --: %s", line)
	}
}
