package main

import "regexp"

// DailyFolderName is a helper constant holding the name of the daily backup grouping folder
const DailyFolderName string = "_daily"

// WeeklyFolderName is a helper constant holding the name of the weekly backup grouping folder
const WeeklyFolderName string = "_weekly"

// MonthlyFolderName is a helper constant holding the name of the monthly backup grouping folder
const MonthlyFolderName string = "_monthly"

// BackupFolderTimeFormat is the time format used to format backup folder names and parse
// them back into a time instance
const BackupFolderTimeFormat string = "2006-01-02_15-04-05"

// BackupFolderNameRegex will match a correct backup folder name (with no suffixes)
var BackupFolderNameRegex = regexp.MustCompile("^(\\d{4})-(\\d{2})-(\\d{2})_(\\d{2})-(\\d{2})-(\\d{2})$")
