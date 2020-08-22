package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/alessio/shellescape"
	"github.com/google/uuid"
)

// ListBackupsInPath returns a string slice contaning relative paths to all backups
// in the passed absPath, relative to basePath
func ListBackupsInPath(options *Options, basePath string, absPath string) []string {
	Log.Debug.Printf("listBackupsInPath(%s)", absPath)
	backups := []string{}

	if options.IsRemoteTarget() {
		stdout, _, err := sshCall(options, fmt.Sprintf("find %s -type d -maxdepth 1", shellescape.Quote(absPath)))
		if err != nil {
			panic(fmt.Sprintf("listBackupsInPath: unexpected error while listing remote target folder %s: %v", options.target, err))
		}

		for _, folderPath := range stdout {
			folderRelativePath, err := filepath.Rel(basePath, folderPath)
			if err != nil {
				panic(fmt.Sprintf("listBackupsInPath: unexpected error while computing relative path on folder %s: %v", folderPath, err))
			}

			folderName := path.Base(folderPath)
			Log.Debug.Printf("listBackupsInPath: candidate folder: %s", folderRelativePath)

			if BackupFolderNameRegex.MatchString(folderName) {
				Log.Debug.Printf("listBackupsInPath: matched folder: %s", folderRelativePath)
				backups = append(backups, folderRelativePath)
			}
		}
	} else {
		files, err := ioutil.ReadDir(absPath)
		if err != nil {
			panic(fmt.Sprintf("listBackupsInPath: unexpected error while listing local target folder %s: %v", options.target, err))
		}

		for _, f := range files {
			if !f.IsDir() {
				continue
			}

			folderRelativePath, err := filepath.Rel(basePath, filepath.Join(absPath, f.Name()))
			if err != nil {
				panic(fmt.Sprintf("listBackupsInPath: unexpected error while computing relative path on folder %s: %v", filepath.Join(absPath, f.Name()), err))
			}

			folderName := path.Base(f.Name())
			Log.Debug.Printf("listBackupsInPath: candidate folder: %s", folderName)

			if BackupFolderNameRegex.MatchString(folderName) {
				Log.Debug.Printf("listBackupsInPath: matched folder: %s", folderRelativePath)
				backups = append(backups, folderRelativePath)
			}
		}
	}

	return backups
}

// PrepareTargetFolder ensures all relevant folders exist at the target location
func PrepareTargetFolder(options *Options) {
	if options.IsRemoteTarget() {
		EnsureRemoteFolderExists(options, options.target)
		EnsureRemoteFolderExists(options, options.DailyFolderPath())
		EnsureRemoteFolderExists(options, options.WeeklyFolderPath())
		EnsureRemoteFolderExists(options, options.MonthlyFolderPath())
	} else {
		EnsureLocalFolderExists(options, options.target)
		EnsureLocalFolderExists(options, options.DailyFolderPath())
		EnsureLocalFolderExists(options, options.WeeklyFolderPath())
		EnsureLocalFolderExists(options, options.MonthlyFolderPath())
	}
}

// EnsureRemoteFolderExists checks for the existence of a remote folder at absPathOnRemote
// and creates it if it does not yet exist
func EnsureRemoteFolderExists(options *Options, absPathOnRemote string) {
	Log.Debug.Printf("EnsureRemoteFolderExists(%s)", absPathOnRemote)

	existsCode, err := uuid.NewRandom()
	if err != nil {
		panic("EnsureRemoteFolderExists: Could not generate okUuid for remote target check")
	}

	notExistsCode, err := uuid.NewRandom()
	if err != nil {
		panic("EnsureRemoteFolderExists: Could not generate notOkUuid for remote target check")
	}

	stdout, _, err := sshCall(options, fmt.Sprintf("if [ -d %s ]; then echo %s; else echo %s; fi", shellescape.Quote(absPathOnRemote), existsCode.String(), notExistsCode.String()))
	if err != nil {
		panic(fmt.Sprintf("EnsureRemoteFolderExists: error checking for remote folder existence: %v", err))
	} else if stdout[0] == existsCode.String() {
		Log.Debug.Println("EnsureRemoteFolderExists: remote folder exists")
		return
	} else if stdout[0] != notExistsCode.String() {
		panic("EnsureRemoteFolderExists: unexpected output checking for remote folder existence (was waiting for one of existCode/notExistsCode")
	}

	Log.Debug.Println("EnsureRemoteFolderExists: remote folder does not exist, creating")

	_, _, err = sshCall(options, fmt.Sprintf("mkdir -p -m 0700 %s", shellescape.Quote(absPathOnRemote)))
	if err != nil {
		panic(fmt.Sprintf("EnsureRemoteFolderExists: unexpected error while creating remote folder %s: %v", absPathOnRemote, err))
	}
}

// EnsureLocalFolderExists checks for the existence of a local folder at absPath
// and creates it if it does not yet exist
func EnsureLocalFolderExists(options *Options, absPath string) {
	Log.Debug.Printf("EnsureLocalFolderExists(%s)", absPath)

	if stat, err := os.Stat(options.target); err == nil {
		Log.Debug.Printf("EnsureLocalFolderExists: target %s exists", absPath)

		if !stat.IsDir() {
			panic(fmt.Sprintf("EnsureLocalFolderExists: %s is not a folder", absPath))
		} else {
			Log.Debug.Printf("EnsureLocalFolderExists: %s is a folder, as expected", absPath)
			return
		}
	} else if os.IsNotExist(err) {
		Log.Debug.Printf("EnsureLocalFolderExists: %s does not exist, creating", absPath)
		err := os.MkdirAll(options.target, 0700)

		if err != nil {
			panic(fmt.Sprintf("EnsureLocalFolderExists: %s does not exist and could not be created", absPath))
		}
	} else {
		panic(fmt.Sprintf("EnsureLocalFolderExists: unexpected error while checking for %s existence: %v", absPath, err))
	}
}

// DetermineLastBackup fetches all backup folder names in the target path and determines the most
// most recent one
func DetermineLastBackup(options *Options) string {
	var backups []string

	backups = append(backups, ListBackupsInPath(options, options.target, options.target)...)
	backups = append(backups, ListBackupsInPath(options, options.target, filepath.Join(options.target, DailyFolderName))...)
	backups = append(backups, ListBackupsInPath(options, options.target, filepath.Join(options.target, WeeklyFolderName))...)
	backups = append(backups, ListBackupsInPath(options, options.target, filepath.Join(options.target, MonthlyFolderName))...)

	if len(backups) > 0 {
		// sort.Strings(backups)
		// Sort by actual date of the backup folder (its basename)
		SortBackupList(&backups, false)

		return backups[len(backups)-1]
	}

	return ""
}
