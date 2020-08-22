package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Options struct {
	profileName   string
	sources       []string
	target        string
	targetHost    string
	targetUser    string
	targetPort    uint
	rsyncOptions  []string
	sshOptions    []string
	maxMain       uint
	maxDaily      uint
	maxWeekly     uint
	maxMonthly    uint
	ReportOptions ReportOptions
}

type ReportOptions struct {
	recipients   []string
	from         string
	smtpHost     string
	smtpPort     uint
	smtpUsername string
	smtpPassword string
	smtpInsecure bool
}

func (options *Options) SSHOptions() []string {
	sshOptions := append(options.sshOptions)
	if options.IsRemoteTarget() {
		if strings.TrimSpace(options.targetUser) != "" {
			sshOptions = append(sshOptions, "-l", strings.TrimSpace(options.targetUser))
		}

		if options.targetPort != 22 {
			sshOptions = append(sshOptions, "-p", fmt.Sprintf("%d", options.targetPort))
		}
	}

	return sshOptions
}

func (options *Options) DailyFolderPath() string {
	return filepath.Join(options.target, DAILY_FOLDER_NAME)
}

func (options *Options) WeeklyFolderPath() string {
	return filepath.Join(options.target, WEEKLY_FOLDER_NAME)
}

func (options *Options) MonthlyFolderPath() string {
	return filepath.Join(options.target, MONTHLY_FOLDER_NAME)
}

func (options *Options) IsRemoteTarget() bool {
	if options.targetHost != "" {
		// TODO Validate user/port
		return true
	} else {
		return false
	}
}
