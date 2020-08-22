package main

import "regexp"

const DAILY_FOLDER_NAME string = "_daily"
const WEEKLY_FOLDER_NAME string = "_weekly"
const MONTHLY_FOLDER_NAME string = "_monthly"

const BACKUP_FOLDER_FORMAT string = "2006-01-02_15-04-05"

var BACKUP_FOLDER_PATTERN = regexp.MustCompile("^(\\d{4})-(\\d{2})-(\\d{2})_(\\d{2})-(\\d{2})-(\\d{2})$")
