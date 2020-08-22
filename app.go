package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

const DAILY_FOLDER_NAME string = "_daily"
const WEEKLY_FOLDER_NAME string = "_weekly"
const MONTHLY_FOLDER_NAME string = "_monthly"

const BACKUP_FOLDER_FORMAT string = "2006-01-02_15-04-05"

var BACKUP_FOLDER_PATTERN = regexp.MustCompile("^(\\d{4})-(\\d{2})-(\\d{2})_(\\d{2})-(\\d{2})-(\\d{2})$")

type StringArrayFlag []string

func (i *StringArrayFlag) String() string {
	return strings.Join(*i, ",")
}
func (i *StringArrayFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func describe(i interface{}) {
	fmt.Printf("(%v, %T)\n", i, i)
}

type Options struct {
	profileName          string
	sources              []string
	target               string
	targetHost           string
	targetUser           string
	targetPort           uint
	rsyncOptions         string
	sshOptions           string
	statusMailRecipients []string
	maxMain              uint
	maxDaily             uint
	maxWeekly            uint
	maxMonthly           uint
}

var logDebug = log.New(os.Stdout, "DEBUG ", log.LstdFlags|log.Lmsgprefix)
var logInfo = log.New(os.Stdout, " INFO ", log.LstdFlags|log.Lmsgprefix)
var logWarn = log.New(os.Stdout, " WARN ", log.LstdFlags|log.Lmsgprefix)
var logError = log.New(os.Stdout, "ERROR ", log.LstdFlags|log.Lmsgprefix)
var logFatal = log.New(os.Stdout, "FATAL ", log.LstdFlags|log.Lmsgprefix)

func main() {
	options := getOptions()

	logInfo.Print("Starting up")
	logInfo.Printf("Starting up %s", DAILY_FOLDER_NAME)

	currentTime := time.Now()
	logInfo.Printf("New backup will be called %s:", currentTime.Format(BACKUP_FOLDER_FORMAT))

	// TODO Validate user/port

}

func isRemoteTarget(options *Options) bool {
	if options.targetHost != "" {
		// TODO Validate user/port
		return true
	} else {
		return true
	}
}

func getSshOptions(options *Options) string {
	sshOptions := strings.TrimSpace(options.sshOptions)
	if isRemoteTarget(options) {
		if strings.TrimSpace(options.targetUser) != "" {
			// TODO Quoting?
			sshOptions += " -l " + strings.TrimSpace(options.targetUser) + " "
		}

		if options.targetPort != 22 {
			sshOptions += " -p " + string(options.targetPort) + " "
		}
	}

	return sshOptions
}

func prepareTargetFolder(options *Options) {
	if isRemoteTarget((options)) {

	}
}

func sshCall(options *Options, cmd string) {

}

func getOptions() Options {
	var options Options

	var profileName string
	flag.StringVar(&profileName, "profile-name", "", "Name for this profile, used in status values.")
	options.profileName = profileName

	var sources StringArrayFlag
	flag.Var(&sources, "source", "Source path(s) passed to rsync. Specify multiple times for multiple values.")
	options.sources = sources

	var target string
	flag.StringVar(&target, "target", "", "Target path. This should be an absolute folder path. For paths on remote hosts, --target-host must be specified. For custom SSH options, such as  target host user/port, pass the -e option to rsync using --rsync-options.")
	options.target = target

	var targetHost string
	flag.StringVar(&targetHost, "target-host", "", "Target host")
	options.targetHost = targetHost

	var targetUser string
	flag.StringVar(&targetUser, "target-user", "", "Target user")
	options.targetUser = targetUser

	var targetPort uint
	flag.UintVar(&targetPort, "target-port", 22, "Target port")
	options.targetPort = targetPort

	var rsyncOptions string
	flag.StringVar(&rsyncOptions, "rsync-options", "", "Extra rsync options. Note that -a and --link-dest are always prepended to these because they are central to how this tool works. -e \"ssh ...\" is also prepended; if you require custom SSH options, pass them in --ssh-options.")
	options.rsyncOptions = rsyncOptions

	var sshOptions string
	flag.StringVar(&sshOptions, "ssh-options", "", "Extra ssh options. Used for calls to ssh and in rsync's -e option.")
	options.rsyncOptions = sshOptions

	var statusMailRecipients StringArrayFlag
	flag.Var(&statusMailRecipients, "mailto", "Status mail recipients. Specify multiple times for multiple values.")
	options.statusMailRecipients = statusMailRecipients

	var maxMain uint
	flag.UintVar(&maxMain, "max-main", 1, "Max number of backups to keep in the main folder (e.g. 10 backups per day)")
	options.maxMain = maxMain

	var maxDaily uint
	flag.UintVar(&maxDaily, "max-daily", 7, "Max number of backups to keep in the daily folder (after which the oldest are moved to the weekly folder)")
	options.maxDaily = maxDaily

	var maxWeekly uint
	flag.UintVar(&maxWeekly, "max-weekly", 52, "Max number of backups to keep in the weekly folder (after which the oldest are moved to the monthly folder)")
	options.maxWeekly = maxWeekly

	var maxMonthly uint
	flag.UintVar(&maxMonthly, "max-monthly", 12, "Max number of backups to keep in the monthly folder (after which the oldest are *discarded*)")
	options.maxMonthly = maxMonthly

	flag.Parse()

	fmt.Println("profileName:", profileName)
	fmt.Println("sources:", sources)
	fmt.Println("target:", target)
	fmt.Println("targetHost:", targetHost)
	fmt.Println("targetUser:", targetUser)
	fmt.Println("targetPort:", targetPort)
	fmt.Println("rsyncOptions:", rsyncOptions)
	fmt.Println("sshOptions:", sshOptions)
	fmt.Println("statusMailRecipients:", statusMailRecipients)
	fmt.Println("maxMain:", maxMain)
	fmt.Println("maxDaily:", maxDaily)
	fmt.Println("maxWeekly:", maxWeekly)
	fmt.Println("maxMonthly:", maxMonthly)

	return options
}
