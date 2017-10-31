# rotating-rsync-backup

Usage: rotating-rsync-backup.pl /path/to/config.conf

Rsync utility script that takes a configuration file path as first argument. Backup
folders are rotated, with a configurable number of daily/weekly/monthly backup folders
being kept. Hardlinks are used where possible.

### Required perl modules

Optional, if you want to send status reports via mail:
* Email::Sender::Simple (package: libemail-sender-perl)