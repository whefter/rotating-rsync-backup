rotating-rsync-backup
---
![publish](https://github.com/whefter/rotating-rsync-backup/workflows/publish/badge.svg)

# Usage

```shell
docker run -it --rm whefter/rotating-rsync-backup --help
```

```shell
NAME:
   rotating-rsync-backup - Create hardlinked backups using rsync and rotate them

USAGE:
   rotating-rsync-backup [global options] command [command options] [arguments...]

VERSION:
   v3.0.0

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --profile-name value, --pn value, -n value      Name for this profile, used in status values. (default: "missing-profile-name")
   --cron value, -c value                          Cron expression. When specified, the profile is not run immediately followed by the program exiting. Rather, it is run according to the passed cron schedule.
   --source value, -s value                        Source path(s) passed to rsync. Specify multiple times for multiple values.
   --target value, -t value                        Required. Target path. This should be an absolute folder path. For paths on remote hosts, --target-host must be specified. For custom SSH options, such as  target host user/port, pass the -e option to rsync using --rsync-options.
   --target-host value, --th value                 Target host
   --target-user value, --tu value                 Target user
   --target-port value, --tp value                 Target port (default: 22)
   --rsync-options value, -r value                 Extra rsync options. Note that -a and --link-dest are always prepended to these because they are central to how this tool works. -e "ssh ..." is also prepended; if you require custom SSH options, pass them in --ssh-options.
   --ssh-options value, -S value                   Extra ssh options. Used for calls to ssh and in rsync's -e option.
   --max-main value, --mM value, -M value          Max number of backups to keep in the main folder (e.g. 10 backups per day) (default: 1)
   --max-daily value, --md value, -d value         Max number of backups to keep in the daily folder (after which the oldest are moved to the weekly folder) (default: 7)
   --max-weekly value, --mw value, -w value        Max number of backups to keep in the weekly folder (after which the oldest are moved to the monthly folder) (default: 52)
   --max-monthly value, --mm value, -m value       Max number of backups to keep in the monthly folder (after which the oldest are *discarded*) (default: 12)
   --report-recipient value, --rr value, -R value  Report mail recipients. Specify multiple times for multiple values.
   --report-from value, --rf value                 Report mail "From" header field. Defaults to <username>@<hostfqdn> - this might not be a valid email address and could throw errors.
   --report-smtp-host value, --rh value            SMTP host to use for sending report mails. (default: "localhost")
   --report-smtp-port value, --rp value            SMTP port to use for sending report mails. (default: 587)
   --report-smtp-username value, --ru value        SMTP username to use for sending report mails.
   --report-smtp-password value, --rP value        SMTP password to use for sending report mails.
   --report-smtp-insecure, --ri                    Skip verification of SMTP server certificates. (default: false)
   --verbose, -v                                   Turn on verbose/debug logging. IMPORTANT NOTE: might print sensitive data; e.g. the full configuration, including passwords. (default: false)
   --help                                          Show help (default: false)
   --version, -V                                   print only the version (default: false)
```

# License

MIT License

Copyright (c) 2020 William Hefter

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
