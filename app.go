package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/alessio/shellescape"
	"github.com/google/uuid"
)

func describe(i interface{}) {
	fmt.Printf("(%v, %T)\n", i, i)
}

// var Log.Debug = log.New(os.Stdout, "DEBUG ", log.LstdFlags|log.Lmsgprefix)
// var Log.Info = log.New(os.Stdout, " INFO ", log.LstdFlags|log.Lmsgprefix)
// var logWarn = log.New(os.Stdout, " WARN ", log.LstdFlags|log.Lmsgprefix)
// var logError = log.New(os.Stdout, "ERROR ", log.LstdFlags|log.Lmsgprefix)
// var Log.Fatal = log.New(os.Stdout, "FATAL ", log.LstdFlags|log.Lmsgprefix)

func main() {
	defer recovery()

	InitLogger()

	options := GetOptions()

	Log.Info.Print("Starting up")

	currentTime := time.Now()
	thisBackupName := currentTime.Format(BACKUP_FOLDER_FORMAT)
	Log.Info.Printf("New backup will be called: %s", thisBackupName)

	// TODO Validate user/port

	prepareTargetFolder(options)

	lastBackupRelativePath := determineLastBackup(options)
	if lastBackupRelativePath == "" {
		Log.Info.Println("No existing backup detected.")
	} else {
		Log.Info.Printf("Last backup: %s", lastBackupRelativePath)
	}

	createBackup(options, thisBackupName, lastBackupRelativePath)
	rotateBackups(options)

	SendStatusMail(options)
}

func recovery() {
	if recoveryMessage := recover(); recoveryMessage != nil {
		Log.Fatal.Printf("Uncaught error: %v", recoveryMessage)
	}
}

func prepareTargetFolder(options *Options) {
	if options.IsRemoteTarget() {
		ensureRemoteFolderExists(options, options.target)
		ensureRemoteFolderExists(options, options.DailyFolderPath())
		ensureRemoteFolderExists(options, options.WeeklyFolderPath())
		ensureRemoteFolderExists(options, options.MonthlyFolderPath())
	} else {
		ensureLocalFolderExists(options, options.target)
		ensureLocalFolderExists(options, options.DailyFolderPath())
		ensureLocalFolderExists(options, options.WeeklyFolderPath())
		ensureLocalFolderExists(options, options.MonthlyFolderPath())
	}
}

func ensureRemoteFolderExists(options *Options, absPathOnRemote string) {
	Log.Debug.Printf("ensureRemoteFolderExists(%s)", absPathOnRemote)

	existsCode, err := uuid.NewRandom()
	if err != nil {
		panic("ensureRemoteFolderExists: Could not generate okUuid for remote target check")
	}

	notExistsCode, err := uuid.NewRandom()
	if err != nil {
		panic("ensureRemoteFolderExists: Could not generate notOkUuid for remote target check")
	}

	stdout, _, err := sshCall(options, fmt.Sprintf("if [ -d %s ]; then echo %s; else echo %s; fi", shellescape.Quote(absPathOnRemote), existsCode.String(), notExistsCode.String()))
	if err != nil {
		panic(fmt.Sprintf("ensureRemoteFolderExists: error checking for remote folder existence: %v", err))
	} else if stdout[0] == existsCode.String() {
		Log.Debug.Println("ensureRemoteFolderExists: remote folder exists")
		return
	} else if stdout[0] != notExistsCode.String() {
		panic("ensureRemoteFolderExists: unexpected output checking for remote folder existence (was waiting for one of existCode/notExistsCode")
	}

	Log.Debug.Println("ensureRemoteFolderExists: remote folder does not exist, creating")

	_, _, err = sshCall(options, fmt.Sprintf("mkdir -p -m 0700 %s", shellescape.Quote(absPathOnRemote)))
	if err != nil {
		panic(fmt.Sprintf("ensureRemoteFolderExists: unexpected error while creating remote folder %s: %v", absPathOnRemote, err))
	}
}

func ensureLocalFolderExists(options *Options, absPath string) {
	Log.Debug.Printf("ensureLocalFolderExists(%s)", absPath)

	if stat, err := os.Stat(options.target); err == nil {
		Log.Debug.Printf("ensureLocalFolderExists: target %s exists", absPath)

		if !stat.IsDir() {
			panic(fmt.Sprintf("ensureLocalFolderExists: %s is not a folder", absPath))
		} else {
			Log.Debug.Printf("ensureLocalFolderExists: %s is a folder, as expected", absPath)
			return
		}
	} else if os.IsNotExist(err) {
		Log.Debug.Printf("ensureLocalFolderExists: %s does not exist, creating", absPath)
		err := os.MkdirAll(options.target, 0700)

		if err != nil {
			panic(fmt.Sprintf("ensureLocalFolderExists: %s does not exist and could not be created", absPath))
		}
	} else {
		panic(fmt.Sprintf("ensureLocalFolderExists: unexpected error while checking for %s existence: %v", absPath, err))
	}
}

func determineLastBackup(options *Options) string {
	var backups []string

	backups = append(backups, listBackupsInPath(options, options.target, options.target)...)
	backups = append(backups, listBackupsInPath(options, options.target, filepath.Join(options.target, DAILY_FOLDER_NAME))...)
	backups = append(backups, listBackupsInPath(options, options.target, filepath.Join(options.target, WEEKLY_FOLDER_NAME))...)
	backups = append(backups, listBackupsInPath(options, options.target, filepath.Join(options.target, MONTHLY_FOLDER_NAME))...)

	if len(backups) > 0 {
		// sort.Strings(backups)
		// Sort by actual date of the backup folder (its basename)
		SortBackupList(&backups, false)

		return backups[len(backups)-1]
	} else {
		return ""
	}
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
