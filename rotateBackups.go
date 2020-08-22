package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alessio/shellescape"
)

func rotateBackups(options *Options) {
	// Move excess from main to daily according to MAIN_MAX
	HandleExcessBackups(options, options.target, options.DailyFolderPath(), options.maxMain)

	// Delete excess in daily (keep oldest from each day), needs no limit
	groupBackups(options, options.DailyFolderPath(), BackupGroupTypeDay)

	// Move excess from daily to weekly according to DAILY_MAX
	HandleExcessBackups(options, options.DailyFolderPath(), options.WeeklyFolderPath(), options.maxDaily)

	// Delete excess in weekly (keep oldest from each week), needs no limit
	groupBackups(options, options.WeeklyFolderPath(), BackupGroupTypeWeek)

	// Move excess from weekly to monthly according to WEEKLY_MAX
	HandleExcessBackups(options, options.WeeklyFolderPath(), options.MonthlyFolderPath(), options.maxWeekly)

	// Delete excess in monthly (keep oldest from each month), needs no limit
	groupBackups(options, options.MonthlyFolderPath(), BackupGroupTypeMonth)

	// Delete excess from monthly according to MONTHLY_MAX
	HandleExcessBackups(options, options.MonthlyFolderPath(), "", options.maxMonthly)
}

// HandleExcessBackups moves excess - as defined by the "maxFrom" parameter - backups from fromPath
// to toPath, if toPath is not empty, and deletes them if toPath is empty.
func HandleExcessBackups(options *Options, fromPath string, toPath string, maxFrom uint) {
	Log.Info.Printf("> Handling excess backups (> %d) in %s to %s", maxFrom, fromPath, toPath)

	backupList := listBackupsInPath(options, fromPath, fromPath)
	SortBackupList(&backupList, false)

	if uint(len(backupList)) > maxFrom {
		for i := maxFrom; i < uint(len(backupList)); i++ {
			currentBackup := backupList[i]

			if toPath == "" {
				Log.Info.Printf("Removing %s", currentBackup)
			} else {
				Log.Info.Printf("Moving %s to %s", currentBackup, toPath)
			}

			currentFrom := filepath.Join(fromPath, currentBackup)
			currentTo := filepath.Join(toPath, currentBackup)

			if options.IsRemoteTarget() {
				fromQuoted := shellescape.Quote(currentFrom)
				toQuoted := shellescape.Quote(currentTo)

				cmd := ""
				if toPath == "" {
					cmd = fmt.Sprintf("rm -rf %s", fromQuoted)
				} else {
					cmd = fmt.Sprintf("mv %s %s", fromQuoted, toQuoted)

				}

				_, _, err := sshCall(options, cmd)
				if err != nil {
					panic(fmt.Sprintf("HandleExcessBackups(): Remote: could not execute %s", cmd))
				}
			} else {
				if toPath == "" {
					err := os.RemoveAll(currentFrom)
					if err != nil {
						panic(fmt.Sprintf("HandleExcessBackups(): could not remove %s", currentFrom))
					}
				} else {
					err := os.Rename(currentFrom, currentTo)
					if err != nil {
						panic(fmt.Sprintf("HandleExcessBackups(): could not rename %s to %s", currentFrom, currentTo))
					}
				}

			}
		}
	} else {
		Log.Debug.Printf("HandleExcessBackups(): no excess backups (< %d) in %s, nothing to do", maxFrom, fromPath)
	}
}

type BackupGroupType string

const (
	BackupGroupTypeDay   BackupGroupType = "Day"
	BackupGroupTypeWeek                  = "Week"
	BackupGroupTypeMonth                 = "Month"
)

func groupBackups(options *Options, sourcePath string, groupBy BackupGroupType) {
	Log.Info.Printf("> Grouping excess backups in %s by %s", sourcePath, groupBy)

	backupList := listBackupsInPath(options, sourcePath, sourcePath)
	SortBackupList(&backupList, true)

	currentOverallGroup := 0

	for _, currentBackup := range backupList {
		Log.Debug.Printf("groupBackups: current backup: %s", currentBackup)

		backupTime, err := BackupNameToTime(currentBackup)
		if err != nil {
			panic(fmt.Sprintf("groupBackups: error parsing backup folder %s into time: %v", currentBackup, err))
		}

		thisBackupGroup := 0

		if groupBy == BackupGroupTypeDay {
			thisBackupGroup = backupTime.Year()*10000 + int(backupTime.Month())*100 + backupTime.Day()
		} else if groupBy == BackupGroupTypeWeek {
			year, week := backupTime.ISOWeek()
			thisBackupGroup = year*10000 + week*100
		} else if groupBy == BackupGroupTypeMonth {
			thisBackupGroup = backupTime.Year()*10000 + int(backupTime.Month())*100
		} else {
			panic(fmt.Sprintf("groupBackups: invalid BackupGroupType %s", groupBy))
		}

		Log.Debug.Printf("groupBackups: current backup group: %d", thisBackupGroup)

		keepBackup := false

		if currentOverallGroup == 0 {
			// Current backup is first overall and thus by definition first of current
			// group, since most recent in current group.
			Log.Debug.Printf("groupBackups: first backup in list, keeping")

			keepBackup = true
			currentOverallGroup = thisBackupGroup
		} else if thisBackupGroup == currentOverallGroup {
			// Current backup's "group" has already occured; the first occurence was kept (by definition),
			// so we can discard this one
			Log.Debug.Printf("groupBackups: group reoccurence, discarding")

			keepBackup = false
		} else if thisBackupGroup > currentOverallGroup {
			// This should never happen, since we loop through our backup list in DESCENDING order
			// and the case for currentOverallGroup == 0 was handled first
			panic(fmt.Sprintf("groupBackups: unexpected case of thisBackupGroup > currentOverallGroup on backup %s, list: %v", currentBackup, backupList))
		} else if thisBackupGroup < currentOverallGroup {
			Log.Debug.Printf("groupBackups: new group, keeping")

			keepBackup = true
			currentOverallGroup = thisBackupGroup
		} else {
			panic(fmt.Sprintf("groupBackups: unexpected fallthrough to edge case, current backup: %s, list: %v", currentBackup, backupList))
		}

		if keepBackup {
			continue
		} else {
			fullPath := filepath.Join(sourcePath, currentBackup)
			if options.IsRemoteTarget() {
				pathQuoted := shellescape.Quote(fullPath)
				cmd := fmt.Sprintf("rm -rf %s", pathQuoted)

				_, _, err := sshCall(options, cmd)
				if err != nil {
					panic(fmt.Sprintf("groupBackups(): Remote: could not execute %s", cmd))
				}
			} else {
				err := os.RemoveAll(fullPath)
				if err != nil {
					panic(fmt.Sprintf("groupBackups(): could not remove folder %s", fullPath))
				}
			}
		}
	}
}
