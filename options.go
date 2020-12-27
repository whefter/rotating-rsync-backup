package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Options is the main options struct
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
	Verbose       bool
}

// ReportOptions is the options struct for report mail-related options
type ReportOptions struct {
	recipients   []string
	from         string
	smtpHost     string
	smtpPort     uint
	smtpUsername string
	smtpPassword string
	smtpInsecure bool
}

// SSHOptions constructs and returns a string slice containing all SSH options, including
// the target user as -l and the port as -p
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

// TargetPath returns a well-formed target path with trailing slash
func (options *Options) TargetPath() string {
	return NormalizeFolderPath(options.target)
}

// DailyFolderPath Returns the full path to the "daily" folder based on the target path
func (options *Options) DailyFolderPath() string {
	return NormalizeFolderPath(filepath.Join(options.target, DailyFolderName))
}

// DailyRelativeFolderPath Returns the relative path to the "daily" folder (relative to target folder)
func (options *Options) DailyRelativeFolderPath() string {
	return options.TargetRelativePath(options.DailyFolderPath())
}

// WeeklyFolderPath Returns the full path to the "weekly" folder based on the target path
func (options *Options) WeeklyFolderPath() string {
	return NormalizeFolderPath(filepath.Join(options.target, WeeklyFolderName))
}

// WeeklyRelativeFolderPath Returns the relative path to the "weekly" folder (relative to target folder)
func (options *Options) WeeklyRelativeFolderPath() string {
	return options.TargetRelativePath(options.WeeklyFolderPath())
}

// MonthlyFolderPath Returns the full path to the "monthly" folder based on the target path
func (options *Options) MonthlyFolderPath() string {
	return NormalizeFolderPath(filepath.Join(options.target, MonthlyFolderName))
}

// MonthlyRelativeFolderPath Returns the relative path to the "monthly" folder (relative to target folder)
func (options *Options) MonthlyRelativeFolderPath() string {
	return options.TargetRelativePath(options.MonthlyFolderPath())
}

// TargetRelativePath Returns the relative path from the target path to the passed path
func (options *Options) TargetRelativePath(fullPath string) string {
	relPath, err := filepath.Rel(options.TargetPath(), fullPath)
	if err != nil {
		panic(err)
	}
	// Log.Info.Printf("TargetRelativePath: from %s to %s = %s", options.TargetPath(), fullPath, relPath)

	return NormalizeFolderPath(relPath)
}

// IsRemoteTarget is a helper function to check if the options indicate the target folder
// is to be on a remote host
func (options *Options) IsRemoteTarget() bool {
	if options.targetHost != "" {
		// TODO Validate user/port
		return true
	}

	return false
}
