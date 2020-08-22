package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/shlex"
)

// Helper construct to read array of values from arguments
type StringArray []string

func (i *StringArray) String() string {
	return strings.Join(*i, ",")
}
func (i *StringArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type Options struct {
	profileName       string
	sources           StringArray
	target            string
	targetHost        string
	targetUser        string
	targetPort        uint
	rsyncOptions      []string
	sshOptions        []string
	maxMain           uint
	maxDaily          uint
	maxWeekly         uint
	maxMonthly        uint
	statusMailOptions StatusMailOptions
}

type StatusMailOptions struct {
	recipients   StringArray
	from         string
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	smtpInsecure bool
}

func GetOptions() *Options {
	var options Options

	flag.StringVar(&(options.profileName), "profile-name", "", "Name for this profile, used in status values.")
	flag.Var(&(options.sources), "source", "Source path(s) passed to rsync. Specify multiple times for multiple values.")
	flag.StringVar(&(options.target), "target", "", "Target path. This should be an absolute folder path. For paths on remote hosts, --target-host must be specified. For custom SSH options, such as  target host user/port, pass the -e option to rsync using --rsync-options.")
	flag.StringVar(&(options.targetHost), "target-host", "", "Target host")
	flag.StringVar(&(options.targetUser), "target-user", "", "Target user")
	flag.UintVar(&(options.targetPort), "target-port", 22, "Target port")

	var rsyncOptions string
	flag.StringVar(&rsyncOptions, "rsync-options", "", "Extra rsync options. Note that -a and --link-dest are always prepended to these because they are central to how this tool works. -e \"ssh ...\" is also prepended; if you require custom SSH options, pass them in --ssh-options.")

	var sshOptions string
	flag.StringVar(&sshOptions, "ssh-options", "", "Extra ssh options. Used for calls to ssh and in rsync's -e option.")

	flag.UintVar(&(options.maxMain), "max-main", 1, "Max number of backups to keep in the main folder (e.g. 10 backups per day)")
	flag.UintVar(&(options.maxDaily), "max-daily", 7, "Max number of backups to keep in the daily folder (after which the oldest are moved to the weekly folder)")
	flag.UintVar(&(options.maxWeekly), "max-weekly", 52, "Max number of backups to keep in the weekly folder (after which the oldest are moved to the monthly folder)")
	flag.UintVar(&(options.maxMonthly), "max-monthly", 12, "Max number of backups to keep in the monthly folder (after which the oldest are *discarded*)")

	flag.Var(&(options.statusMailOptions.recipients), "status-mailto", "Status mail recipients. Specify multiple times for multiple values.")
	flag.StringVar(&(options.statusMailOptions.from), "status-from", "", "Status mail \"from\".")
	flag.StringVar(&(options.statusMailOptions.smtpHost), "status-smtp-host", "", "SMTP host for status mail sending.")
	flag.IntVar(&(options.statusMailOptions.smtpPort), "status-smtp-port", 0, "SMTP port for status mail sending.")
	flag.StringVar(&(options.statusMailOptions.smtpUsername), "status-smtp-username", "", "SMTP username for status mail sending.")
	flag.StringVar(&(options.statusMailOptions.smtpPassword), "status-smtp-password", "", "SMTP password for status mail sending.")
	flag.BoolVar(&(options.statusMailOptions.smtpInsecure), "status-smtp-insecure", false, "Skip verify on SMTP certificates.")

	flag.Parse()

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

	splitSshOptions, err := shlex.Split(sshOptions)
	if err != nil {
		panic(fmt.Sprintf("Invalid --ssh-options: %v", err))
	}
	options.sshOptions = splitSshOptions

	splitRsyncOptions, err := shlex.Split(rsyncOptions)
	if err != nil {
		panic(fmt.Sprintf("Invalid --rsync-options: %v", err))
	}
	options.rsyncOptions = splitRsyncOptions

	Log.Debug.Println("profileName:", options.profileName)
	Log.Debug.Println("sources:", options.sources)
	Log.Debug.Println("target:", options.target)
	Log.Debug.Println("targetHost:", options.targetHost)
	Log.Debug.Println("targetUser:", options.targetUser)
	Log.Debug.Println("targetPort:", options.targetPort)
	Log.Debug.Println("rsyncOptions:", options.rsyncOptions)
	Log.Debug.Println("sshOptions:", options.sshOptions)
	Log.Debug.Println("statusMailOptions.recipients:", options.statusMailOptions.recipients)
	Log.Debug.Println("statusMailOptions.from:", options.statusMailOptions.from)
	Log.Debug.Println("statusMailOptions.smtpHost:", options.statusMailOptions.smtpHost)
	Log.Debug.Println("statusMailOptions.smtpPort:", options.statusMailOptions.smtpPort)
	Log.Debug.Println("statusMailOptions.smtpUsername:", options.statusMailOptions.smtpUsername)
	Log.Debug.Println("statusMailOptions.smtpPassword:", options.statusMailOptions.smtpPassword)
	Log.Debug.Println("statusMailOptions.smtpInsecure:", options.statusMailOptions.smtpInsecure)
	Log.Debug.Println("maxMain:", options.maxMain)
	Log.Debug.Println("maxDaily:", options.maxDaily)
	Log.Debug.Println("maxWeekly:", options.maxWeekly)
	Log.Debug.Println("maxMonthly:", options.maxMonthly)

	return &options
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
