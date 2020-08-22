package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/alessio/shellescape"
	"github.com/google/shlex"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

func main() {
	defer recovery()

	InitLogger()

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "profile-name",
				Value:    "missing-profile-name",
				Usage:    "Name for this profile, used in status values.",
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:     "source",
				Usage:    "Source path(s) passed to rsync. Specify multiple times for multiple values.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "target",
				Usage:    "Target path. This should be an absolute folder path. For paths on remote hosts, --target-host must be specified. For custom SSH options, such as  target host user/port, pass the -e option to rsync using --rsync-options.",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "target-host",
				Usage:    "Target host",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "target-user",
				Usage:    "Target user",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "target-port",
				Value:    22,
				Usage:    "Target port",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "rsync-options",
				Value:    "",
				Usage:    "Extra rsync options. Note that -a and --link-dest are always prepended to these because they are central to how this tool works. -e \"ssh ...\" is also prepended; if you require custom SSH options, pass them in --ssh-options.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "ssh-options",
				Value:    "",
				Usage:    "Extra ssh options. Used for calls to ssh and in rsync's -e option.",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "max-main",
				Value:    1,
				Usage:    "Max number of backups to keep in the main folder (e.g. 10 backups per day)",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "max-daily",
				Value:    7,
				Usage:    "Max number of backups to keep in the daily folder (after which the oldest are moved to the weekly folder)",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "max-weekly",
				Value:    52,
				Usage:    "Max number of backups to keep in the weekly folder (after which the oldest are moved to the monthly folder)",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "max-monthly",
				Value:    12,
				Usage:    "Max number of backups to keep in the monthly folder (after which the oldest are *discarded*)",
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:     "report-recipient",
				Usage:    "Report mail recipients. Specify multiple times for multiple values.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "report-from",
				Usage:    "Report mail \"From\" header field.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "report-smtp-host",
				Usage:    "SMTP host to use for sending report mails.",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "report-smtp-port",
				Value:    587,
				Usage:    "SMTP port to use for sending report mails.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "report-smtp-username",
				Usage:    "SMTP username to use for sending report mails.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "report-smtp-password",
				Usage:    "SMTP password to use for sending report mails.",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "report-smtp-insecure",
				Value:    false,
				Usage:    "Skip verification of SMTP server certificates.",
				Required: false,
			},
		},
		Action: func(c *cli.Context) error {
			var options Options

			options.profileName = c.String("profile-name")

			options.sources = c.StringSlice("source")
			// Validate sources
			if len(options.sources) == 0 {
				panic("No sources specified")
			}
			for _, source := range options.sources {
				var invalidSource bool
				if !filepath.IsAbs(source) {
					invalidSource = true
					Log.Fatal.Printf("Source must be absolute: %s", source)
				}

				if invalidSource {
					panic("Invalid sources")
				}
			}

			options.target = c.String("target")
			options.targetHost = c.String("target-host")
			options.targetUser = c.String("target-user")
			options.targetPort = c.Uint("target-port")

			rsyncOptionsRaw := c.String("rsync-options")
			splitRsyncOptions, err := shlex.Split(rsyncOptionsRaw)
			if err != nil {
				panic(fmt.Sprintf("Invalid --rsync-options: %v", err))
			}
			options.rsyncOptions = splitRsyncOptions

			sshOptionsRaw := c.String("ssh-options")
			splitSshOptions, err := shlex.Split(sshOptionsRaw)
			if err != nil {
				panic(fmt.Sprintf("Invalid --ssh-options: %v", err))
			}
			options.sshOptions = splitSshOptions

			options.maxMain = c.Uint("max-main")
			options.maxDaily = c.Uint("max-daily")
			options.maxWeekly = c.Uint("max-weekly")
			options.maxMonthly = c.Uint("max-monthly")

			options.ReportOptions.recipients = c.StringSlice("report-recipient")
			options.ReportOptions.from = c.String("report-from")
			options.ReportOptions.smtpHost = c.String("report-smtp-host")
			options.ReportOptions.smtpPort = c.Uint("report-smtp-port")
			options.ReportOptions.smtpUsername = c.String("report-smtp-username")
			options.ReportOptions.smtpPassword = c.String("report-smtp-password")
			options.ReportOptions.smtpInsecure = c.Bool("report-smtp-insecure")

			Log.Debug.Println("profileName:", options.profileName)
			Log.Debug.Println("sources:", options.sources)
			Log.Debug.Println("target:", options.target)
			Log.Debug.Println("targetHost:", options.targetHost)
			Log.Debug.Println("targetUser:", options.targetUser)
			Log.Debug.Println("targetPort:", options.targetPort)
			Log.Debug.Println("rsyncOptions:", options.rsyncOptions)
			Log.Debug.Println("sshOptions:", options.sshOptions)
			Log.Debug.Println("ReportOptions.recipients:", options.ReportOptions.recipients)
			Log.Debug.Println("ReportOptions.from:", options.ReportOptions.from)
			Log.Debug.Println("ReportOptions.smtpHost:", options.ReportOptions.smtpHost)
			Log.Debug.Println("ReportOptions.smtpPort:", options.ReportOptions.smtpPort)
			Log.Debug.Println("ReportOptions.smtpUsername:", options.ReportOptions.smtpUsername)
			Log.Debug.Println("ReportOptions.smtpPassword:", options.ReportOptions.smtpPassword)
			Log.Debug.Println("ReportOptions.smtpInsecure:", options.ReportOptions.smtpInsecure)
			Log.Debug.Println("maxMain:", options.maxMain)
			Log.Debug.Println("maxDaily:", options.maxDaily)
			Log.Debug.Println("maxWeekly:", options.maxWeekly)
			Log.Debug.Println("maxMonthly:", options.maxMonthly)

			// TODO Validate user/port

			run(&options)

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(options *Options) {
	Log.Info.Print("Starting up")

	currentTime := time.Now()
	thisBackupName := currentTime.Format(BACKUP_FOLDER_FORMAT)
	Log.Info.Printf("New backup will be called: %s", thisBackupName)

	prepareTargetFolder(options)

	lastBackupRelativePath := determineLastBackup(options)
	if lastBackupRelativePath == "" {
		Log.Info.Println("No existing backup detected.")
	} else {
		Log.Info.Printf("Last backup: %s", lastBackupRelativePath)
	}

	CreateBackup(options, thisBackupName, lastBackupRelativePath)
	RotateBackups(options)

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

	backups = append(backups, ListBackupsInPath(options, options.target, options.target)...)
	backups = append(backups, ListBackupsInPath(options, options.target, filepath.Join(options.target, DAILY_FOLDER_NAME))...)
	backups = append(backups, ListBackupsInPath(options, options.target, filepath.Join(options.target, WEEKLY_FOLDER_NAME))...)
	backups = append(backups, ListBackupsInPath(options, options.target, filepath.Join(options.target, MONTHLY_FOLDER_NAME))...)

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
