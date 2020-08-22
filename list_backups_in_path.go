package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/alessio/shellescape"
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

			if BACKUP_FOLDER_PATTERN.MatchString(folderName) {
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

			if BACKUP_FOLDER_PATTERN.MatchString(folderName) {
				Log.Debug.Printf("listBackupsInPath: matched folder: %s", folderRelativePath)
				backups = append(backups, folderRelativePath)
			}
		}
	}

	return backups
}
