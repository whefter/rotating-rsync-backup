package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/google/shlex"
	"github.com/google/uuid"
)

const DAILY_FOLDER_NAME string = "_daily"
const WEEKLY_FOLDER_NAME string = "_weekly"
const MONTHLY_FOLDER_NAME string = "_monthly"

const BACKUP_FOLDER_FORMAT string = "2006-01-02_15-04-05"

var BACKUP_FOLDER_PATTERN = regexp.MustCompile("^(\\d{4})-(\\d{2})-(\\d{2})_(\\d{2})-(\\d{2})-(\\d{2})$")

// Helper construct to read array of values from arguments
type StringArray []string

func (i *StringArray) String() string {
	return strings.Join(*i, ",")
}
func (i *StringArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func describe(i interface{}) {
	fmt.Printf("(%v, %T)\n", i, i)
}

type Options struct {
	profileName          string
	sources              StringArray
	target               string
	targetHost           string
	targetUser           string
	targetPort           uint
	rsyncOptions         []string
	sshOptions           []string
	statusMailRecipients StringArray
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
	defer recovery()

	options := getOptions()

	logInfo.Print("Starting up")

	currentTime := time.Now()
	thisBackupName := currentTime.Format(BACKUP_FOLDER_FORMAT)
	logInfo.Printf("New backup will be called: %s", thisBackupName)

	// TODO Validate user/port

	prepareTargetFolder(&options)

	lastBackupRelativePath := determineLastBackup(&options)
	if lastBackupRelativePath == "" {
		logInfo.Println("No existing backup detected.")
	} else {
		logInfo.Printf("Last backup: %s", lastBackupRelativePath)
	}

	createBackup(&options, thisBackupName, lastBackupRelativePath)
}

func recovery() {
	if recoveryMessage := recover(); recoveryMessage != nil {
		logFatal.Printf("Uncaught error: %v", recoveryMessage)
	}
}

func createBackup(options *Options, thisBackupName string, lastBackupRelativePath string) {
	// Add target, check for existence and create if necessary
	// Use a temporary folder and rename to an error folder if anything fails. That way, if the script is interrupted
	// or ends in an error, the temporary/error folders won't crowd out the actual folders during groups/excess deletes.
	// targetPath := filepath.Join(options.target, thisBackupName)
	progressTargetPath := filepath.Join(options.target, thisBackupName+"_progress")
	// errorTargetPath := filepath.Join(options.target, thisBackupName+"_error")

	args := []string{"-a", "--delete"}

	if lastBackupRelativePath != "" {
		// --link-dest must be relative to the TARGET FOLDER. It does not take user:host@ before the relative path,
		// but figures that out itself
		args = append(args, "--link-dest", lastBackupRelativePath)
	}

	args = append(args, options.rsyncOptions...)
	args = append(args, "-e", fmt.Sprintf("ssh %s", strings.Join(getSSHOptions(options), " ")))

	for _, source := range options.sources {
		args = append(args, source)
	}

	if isRemoteTarget(options) {
		args = append(args, fmt.Sprintf("%s:%s", options.targetHost, progressTargetPath))
	} else {
		args = append(args, progressTargetPath)
	}

	logDebug.Printf("createBackup: cmdLine: rsync %s", strings.Join(args, " "))

	// _, _, err := call("printenv", []string{})
	_, _, err := call("rsync", args)
	if err != nil {
		panic(fmt.Sprintf("Error executing rsync command: %v", err))
	}
}

func isRemoteTarget(options *Options) bool {
	if options.targetHost != "" {
		// TODO Validate user/port
		return true
	} else {
		return false
	}
}

func getSSHOptions(options *Options) []string {
	sshOptions := append(options.sshOptions)
	if isRemoteTarget(options) {
		if strings.TrimSpace(options.targetUser) != "" {
			sshOptions = append(sshOptions, "-l", strings.TrimSpace(options.targetUser))
		}

		if options.targetPort != 22 {
			sshOptions = append(sshOptions, "-p", fmt.Sprintf("%d", options.targetPort))
		}
	}

	return sshOptions
}

func prepareTargetFolder(options *Options) {
	if isRemoteTarget(options) {
		ensureRemoteFolderExists(options, options.target)
		ensureRemoteFolderExists(options, filepath.Join(options.target, DAILY_FOLDER_NAME))
		ensureRemoteFolderExists(options, filepath.Join(options.target, WEEKLY_FOLDER_NAME))
		ensureRemoteFolderExists(options, filepath.Join(options.target, MONTHLY_FOLDER_NAME))
	} else {
		ensureLocalFolderExists(options, options.target)
		ensureLocalFolderExists(options, filepath.Join(options.target, DAILY_FOLDER_NAME))
		ensureLocalFolderExists(options, filepath.Join(options.target, WEEKLY_FOLDER_NAME))
		ensureLocalFolderExists(options, filepath.Join(options.target, MONTHLY_FOLDER_NAME))
	}
}

func ensureRemoteFolderExists(options *Options, absPathOnRemote string) {
	logDebug.Printf("ensureRemoteFolderExists(%s)", absPathOnRemote)

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
		logDebug.Println("ensureRemoteFolderExists: remote folder exists")
		return
	} else if stdout[0] != notExistsCode.String() {
		panic("ensureRemoteFolderExists: unexpected output checking for remote folder existence (was waiting for one of existCode/notExistsCode")
	}

	logDebug.Println("ensureRemoteFolderExists: remote folder does not exist, creating")

	_, _, err = sshCall(options, fmt.Sprintf("mkdir -p -m 0700 %s", shellescape.Quote(absPathOnRemote)))
	if err != nil {
		panic(fmt.Sprintf("ensureRemoteFolderExists: unexpected error while creating remote folder %s: %v", absPathOnRemote, err))
	}
}

func ensureLocalFolderExists(options *Options, absPath string) {
	logDebug.Printf("ensureLocalFolderExists(%s)", absPath)

	if stat, err := os.Stat(options.target); err == nil {
		logDebug.Printf("ensureLocalFolderExists: target %s exists", absPath)

		if !stat.IsDir() {
			panic(fmt.Sprintf("ensureLocalFolderExists: %s is not a folder", absPath))
		} else {
			logDebug.Printf("ensureLocalFolderExists: %s is a folder, as expected", absPath)
			return
		}
	} else if os.IsNotExist(err) {
		logDebug.Printf("ensureLocalFolderExists: %s does not exist, creating", absPath)
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
		sort.Strings(backups)
		// Sort by actual date of the backup folder (its basename)
		sort.SliceStable(backups, func(i, j int) bool {
			iBasename := filepath.Base(backups[i])
			iDate, err := time.Parse(BACKUP_FOLDER_FORMAT, iBasename)
			if err != nil {
				panic(fmt.Sprintf("determineLastBackup: error parsing backup folder %s into time: %v", iBasename, err))
			}

			jBasename := filepath.Base(backups[j])
			jDate, err := time.Parse(BACKUP_FOLDER_FORMAT, jBasename)
			if err != nil {
				panic(fmt.Sprintf("determineLastBackup: error parsing backup folder %s into time: %v", jBasename, err))
			}

			if iDate.Before(jDate) {
				return true
			}

			return false
		})

		return backups[len(backups)-1]
	} else {
		return ""
	}
}

func listBackupsInPath(options *Options, basePath string, absPath string) []string {
	logDebug.Printf("listBackupsInPath(%s)", absPath)
	backups := []string{}

	if isRemoteTarget(options) {
		stdout, _, err := sshCall(options, fmt.Sprintf("find %s -type d -maxdepth 1", shellescape.Quote(absPath)))
		if err != nil {
			panic(fmt.Sprintf("listBackupsInPath: unexpected error while listing remote target folder %s: %v", options.target, err))
		}

		for _, folderPath := range stdout {
			folderRelativePath, err := filepath.Rel(basePath, folderPath)
			if err != nil {
				panic(fmt.Sprintf("listBackupsInPath: unexpected error while computing relative path on folder %s: %v", folderPath, err))
			}

			folderName := path.Base(folderPath)
			logDebug.Printf("listBackupsInPath: candidate folder: %s", folderRelativePath)

			if BACKUP_FOLDER_PATTERN.MatchString(folderName) {
				logDebug.Printf("listBackupsInPath: matched folder: %s", folderRelativePath)
				backups = append(backups, folderRelativePath)
			}
		}
	} else {
		files, err := ioutil.ReadDir(absPath)
		if err != nil {
			panic(fmt.Sprintf("listBackupsInPath: unexpected error while listing local target folder %s: %v", options.target, err))
		}

		for _, f := range files {
			if !f.IsDir() {
				continue
			}

			folderRelativePath, err := filepath.Rel(basePath, filepath.Join(absPath, f.Name()))
			if err != nil {
				panic(fmt.Sprintf("listBackupsInPath: unexpected error while computing relative path on folder %s: %v", filepath.Join(absPath, f.Name()), err))
			}

			folderName := path.Base(f.Name())
			logDebug.Printf("listBackupsInPath: candidate folder: %s", folderName)

			if BACKUP_FOLDER_PATTERN.MatchString(folderName) {
				logDebug.Printf("listBackupsInPath: matched folder: %s", folderRelativePath)
				backups = append(backups, folderRelativePath)
			}
		}
	}

	return backups
}

func sshCall(options *Options, sshCmd string) ([]string, []string, error) {
	args := []string{}

	args = append(args, getSSHOptions(options)...)
	args = append(args, options.targetHost)
	args = append(args, sshCmd)

	return call("ssh", args)
}

func call(command string, args []string) ([]string, []string, error) {
	logDebug.Printf("call: Full command line: %s %v", command, args)

	cmd := exec.Command(command, args...)
	logDebug.Printf("test? %s %v", command, cmd.Args)

	// newCommand := "bash"
	// newArgs := []string{"-c", shellescape.Quote(command + " " + strings.Join(args, " "))}
	// logDebug.Printf("FOOOOO %s %s", newCommand, strings.Join(newArgs, " "))
	// cmd := exec.Command(newCommand, newArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(fmt.Sprintf("call: Could not get StdoutPipe: %v", err))
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(fmt.Sprintf("call: Could not get Stderr: %v", err))
	}

	if err := cmd.Start(); err != nil {
		panic(fmt.Sprintf("call: could not Start() cmd: %v", err))
	}

	var fullStdout []string
	var fullStderr []string

	go printStdout(stdout, &fullStdout)
	go printStderr(stderr, &fullStderr)

	err = cmd.Wait()

	logDebug.Printf("call: Command finished with error: %v", err)
	// if err != nil {
	// 	panic(fmt.Sprintf("sshCall: returned with error %v", err))
	// }

	logDebug.Println("call: All stdout lines", fullStdout)
	logDebug.Println("call: All stderr lines", fullStderr)

	return fullStdout, fullStderr, err
}

func printStdout(stdout io.ReadCloser, stash *[]string) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		*stash = append(*stash, line)
		logDebug.Printf("-- STDOUT --: %s", line)
	}
}

func printStderr(stderr io.ReadCloser, stash *[]string) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		*stash = append(*stash, line)
		logDebug.Printf("-- STDERR --: %s", line)
	}
}

func getOptions() Options {
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

	flag.Var(&(options.statusMailRecipients), "mailto", "Status mail recipients. Specify multiple times for multiple values.")
	flag.UintVar(&(options.maxMain), "max-main", 1, "Max number of backups to keep in the main folder (e.g. 10 backups per day)")
	flag.UintVar(&(options.maxDaily), "max-daily", 7, "Max number of backups to keep in the daily folder (after which the oldest are moved to the weekly folder)")
	flag.UintVar(&(options.maxWeekly), "max-weekly", 52, "Max number of backups to keep in the weekly folder (after which the oldest are moved to the monthly folder)")
	flag.UintVar(&(options.maxMonthly), "max-monthly", 12, "Max number of backups to keep in the monthly folder (after which the oldest are *discarded*)")

	flag.Parse()

	// Validate sources
	if len(options.sources) == 0 {
		panic("No sources specified")
	}
	for _, source := range options.sources {
		var invalidSource bool
		if !filepath.IsAbs(source) {
			invalidSource = true
			logFatal.Printf("Source must be absolute: %s", source)
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

	logDebug.Println("profileName:", options.profileName)
	logDebug.Println("sources:", options.sources)
	logDebug.Println("target:", options.target)
	logDebug.Println("targetHost:", options.targetHost)
	logDebug.Println("targetUser:", options.targetUser)
	logDebug.Println("targetPort:", options.targetPort)
	logDebug.Println("rsyncOptions:", options.rsyncOptions)
	logDebug.Println("sshOptions:", options.sshOptions)
	logDebug.Println("statusMailRecipients:", options.statusMailRecipients)
	logDebug.Println("maxMain:", options.maxMain)
	logDebug.Println("maxDaily:", options.maxDaily)
	logDebug.Println("maxWeekly:", options.maxWeekly)
	logDebug.Println("maxMonthly:", options.maxMonthly)

	return options
}
