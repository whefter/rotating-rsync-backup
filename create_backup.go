package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alessio/shellescape"
)

// CreateBackup runs all necessary commands to create a new backup based on the passed
// backup name thisBackupName and the relative path lastBackupRelativePath to the last backup
// to use as hard link destination. Note that lastBackupRelativePath is relative to the MAIn
// target folder
func CreateBackup(options *Options, thisBackupName string, lastBackupRelativePath string) {
	Log.Info.Printf("Backing up sources: %v", options.sources)

	// Add target, check for existence and create if necessary
	// Use a temporary folder and rename to an error folder if anything fails. That way, if the script is interrupted
	// or ends in an error, the temporary/error folders won't crowd out the actual folders during groups/excess deletes.
	targetPath := filepath.Join(options.target, thisBackupName)
	progressTargetPath := filepath.Join(options.target, thisBackupName+"_progress")
	errorTargetPath := filepath.Join(options.target, thisBackupName+"_error")

	args := []string{"-a", "--delete"}

	if lastBackupRelativePath != "" {
		// --link-dest must be relative to the TARGET FOLDER, which means the NEWLY created backup folder
		// (not the "main" folder). Hence the "../".
		// It does not take user:host@ before the relative path, but figures that out itself
		args = append(args, "--link-dest", filepath.Join("../", lastBackupRelativePath))
	}

	args = append(args, options.rsyncOptions...)
	args = append(args, "-e", fmt.Sprintf("ssh %s", strings.Join(options.SSHOptions(), " ")))

	for _, source := range options.sources {
		args = append(args, source)
	}

	if options.IsRemoteTarget() {
		args = append(args, fmt.Sprintf("%s:%s", options.targetHost, progressTargetPath))
	} else {
		args = append(args, progressTargetPath)
	}

	Log.Debug.Printf("createBackup: cmdLine: rsync %s", strings.Join(args, " "))

	// _, _, _, err := call("printenv", []string{})
	_, _, exitCode, err := call("rsync", args, "rsync", true)
	if err != nil {
		if exitCode == 23 || exitCode == 24 || exitCode == 25 {
			Log.Warn.Printf("Rsync exited with exit code %v; indicating that some files could not be transfered/deleted.", exitCode)
		} else {
			Log.Fatal.Printf("Error executing rsync command: %v", err)
			Log.Debug.Printf("Renaming progress folder %s to %s", progressTargetPath, errorTargetPath)

			mvErr := os.Rename(progressTargetPath, errorTargetPath)
			if mvErr != nil {
				Log.Fatal.Printf("Could not rename progress folder %s to error folder %s", progressTargetPath, errorTargetPath)
			}

			panic(fmt.Sprintf("Error executing rsync command: %v", err))
		}

	}

	Log.Debug.Printf("Renaming temporary folder %s to %s", progressTargetPath, targetPath)
	if options.IsRemoteTarget() {
		_, _, _, err := sshCall(
			options,
			fmt.Sprintf("mv %s %s", shellescape.Quote(progressTargetPath), shellescape.Quote(targetPath)),
			options.Verbose,
		)
		if err != nil {
			panic(fmt.Sprintf("Could not rename remote progress folder %s to final target folder %s", progressTargetPath, targetPath))
		}
	} else {
		mvErr := os.Rename(progressTargetPath, targetPath)
		if mvErr != nil {
			panic(fmt.Sprintf("Could not rename progress folder %s to final target folder %s", progressTargetPath, targetPath))
		}
	}
}
