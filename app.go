package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/shlex"
	"github.com/robfig/cron/v3"
	"github.com/urfave/cli/v2"
)

func main() {
	defer recovery()

	// Free up "-h"
	cli.HelpFlag = &cli.BoolFlag{
		Name:  "help",
		Usage: "Show help",
	}
	// Free up "-v"
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "print only the version",
	}

	app := &cli.App{
		Name:    "rotating-rsync-backup",
		Version: "v3.0.0-beta.3",
		Usage:   "Create hardlinked backups using rsync and rotate them",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "profile-name",
				Aliases:  []string{"pn", "n"},
				Value:    "missing-profile-name",
				Usage:    "Name for this profile, used in status values.",
				Required: false,
			},
			// A note about cron: an alternative would be to leave the scheduled execution to the user
			// and not implement any cron-related functionality (separation of concerns). However,
			// this quickly leads to some uid/gid-related issues when building a Docker image around
			// this utility, because building an Alpine-based image with cron that allows for both
			// easy and correct handling of uid/gid is not straightforward. Moving the cron feature
			// to the application means it will always run in the context of the application's uid/gid.
			&cli.StringFlag{
				Name:     "cron",
				Aliases:  []string{"c"},
				Value:    "",
				Usage:    "Cron expression. When specified, the profile is not run immediately followed by the program exiting. Rather, it is run according to the passed cron schedule.",
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:     "source",
				Aliases:  []string{"s"},
				Usage:    "Source path(s) passed to rsync. Specify multiple times for multiple values.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "target",
				Aliases:  []string{"t"},
				Usage:    "Required. Target path. This should be an absolute folder path. For paths on remote hosts, --target-host must be specified. For custom SSH options, such as  target host user/port, pass the -e option to rsync using --rsync-options.",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "target-host",
				Aliases:  []string{"th"},
				Usage:    "Target host",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "target-user",
				Aliases:  []string{"tu"},
				Usage:    "Target user",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "target-port",
				Aliases:  []string{"tp"},
				Value:    22,
				Usage:    "Target port",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "rsync-options",
				Aliases:  []string{"r"},
				Value:    "",
				Usage:    "Extra rsync options. Note that -a and --link-dest are always prepended to these because they are central to how this tool works. -e \"ssh ...\" is also prepended; if you require custom SSH options, pass them in --ssh-options.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "ssh-options",
				Aliases:  []string{"S"},
				Value:    "",
				Usage:    "Extra ssh options. Used for calls to ssh and in rsync's -e option.",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "max-main",
				Aliases:  []string{"mM", "M"},
				Value:    1,
				Usage:    "Max number of backups to keep in the main folder (e.g. 10 backups per day)",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "max-daily",
				Aliases:  []string{"md", "d"},
				Value:    7,
				Usage:    "Max number of backups to keep in the daily folder (after which the oldest are moved to the weekly folder)",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "max-weekly",
				Aliases:  []string{"mw", "w"},
				Value:    52,
				Usage:    "Max number of backups to keep in the weekly folder (after which the oldest are moved to the monthly folder)",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "max-monthly",
				Aliases:  []string{"mm", "m"},
				Value:    12,
				Usage:    "Max number of backups to keep in the monthly folder (after which the oldest are *discarded*)",
				Required: false,
			},
			&cli.StringSliceFlag{
				Name:     "report-recipient",
				Aliases:  []string{"rr", "R"},
				Usage:    "Report mail recipients. Specify multiple times for multiple values.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "report-from",
				Aliases:  []string{"rf"},
				Usage:    "Report mail \"From\" header field.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "report-smtp-host",
				Aliases:  []string{"rh"},
				Usage:    "SMTP host to use for sending report mails.",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "report-smtp-port",
				Aliases:  []string{"rp"},
				Value:    587,
				Usage:    "SMTP port to use for sending report mails.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "report-smtp-username",
				Aliases:  []string{"ru"},
				Usage:    "SMTP username to use for sending report mails.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "report-smtp-password",
				Aliases:  []string{"rP"},
				Usage:    "SMTP password to use for sending report mails.",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "report-smtp-insecure",
				Aliases:  []string{"ri"},
				Value:    false,
				Usage:    "Skip verification of SMTP server certificates.",
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "verbose",
				Aliases:  []string{"v"},
				Value:    false,
				Usage:    "Turn on verbose/debug logging.",
				Required: false,
			},
		},
		Action: func(c *cli.Context) error {
			var options Options

			options.Verbose = c.Bool("verbose")

			InitLogger(options.Verbose)

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
			splitSSHOptions, err := shlex.Split(sshOptionsRaw)
			if err != nil {
				panic(fmt.Sprintf("Invalid --ssh-options: %v", err))
			}
			options.sshOptions = splitSSHOptions

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

			// TODO Validate user/port

			cronExpression := c.String("cron")
			if cronExpression == "" {
				run(&options)
				SendReportMail(&options)
			} else {
				specParser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
				_, err := specParser.Parse(cronExpression)

				if err != nil {
					panic(fmt.Sprintf("Invalid cron expression for schedule: %v", err))
				}

				cronLogger := cron.PrintfLogger(log.New(os.Stderr, "cron: ", log.LstdFlags|log.Lmsgprefix))
				c := cron.New(
					cron.WithParser(specParser),
					cron.WithLogger(cronLogger),
					cron.WithChain(
						cron.Recover(cronLogger),
						cron.Recover(cron.DiscardLogger),
						cron.DelayIfStillRunning(cronLogger),
						cron.DelayIfStillRunning(cron.DiscardLogger),
					),
				)

				// Make entryID available inside func
				var entryID cron.EntryID
				entryID, err = c.AddFunc(cronExpression, func() {
					run(&options)
					Log.Info.Printf("Next execution: %s", c.Entry(entryID).Next)

					SendReportMail(&options)
					Log.Reset()
				})
				if err != nil {
					panic(fmt.Sprintf("Error adding cron job: %v", err))
				}

				c.Start()
				fmt.Printf("Started cron: %s, next execution: %s", cronExpression, c.Entry(entryID).Next)

				// Wait indefinitely while listening for SIGINT/SIGTERM
				exitSignal := make(chan os.Signal)
				signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
				<-exitSignal
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(options *Options) {
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

	Log.Info.Print("Starting up")

	currentTime := time.Now()
	thisBackupName := currentTime.Format(BackupFolderTimeFormat)
	Log.Info.Printf("New backup will be called: %s", thisBackupName)

	PrepareTargetFolder(options)

	lastBackupRelativePath := DetermineLastBackup(options)
	if lastBackupRelativePath == "" {
		Log.Info.Println("No existing backup detected.")
	} else {
		Log.Info.Printf("Last backup: %s", lastBackupRelativePath)
	}

	CreateBackup(options, thisBackupName, lastBackupRelativePath)
	RotateBackups(options)
}

func recovery() {
	if recoveryMessage := recover(); recoveryMessage != nil {
		Log.Fatal.Printf("Uncaught error: %v", recoveryMessage)
	}
}
