#!/usr/bin/env perl

# rotating-rsync-backup v1.0
#
# Usage: rotating-rsync-backup.pl /path/to/config.conf
#
# Rsync utility script that takes a configuration file path as first argument. Backup
# folders are rotated, with a configurable number of daily/weekly/monthly backup folders
# being kept. Hardlinks are used where possible.
#
# Copyright (c) 2014-2016 William Hefter <william@whefter.de>
#
# This program is free software; you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation; either version 2 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License along
# with this program; if not, write to the Free Software Foundation, Inc.,
# 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.

use warnings;
use strict;

use File::Copy;
use File::Path 'remove_tree';
use Date::Parse;
use Date::Format;
use Data::Dumper;
use B qw(svref_2object);

# Clean path to rsync
my $rsyncCmd = `which rsync`;
chomp $rsyncCmd;

my $sshCmd = `which ssh`;
chomp $sshCmd;

my $dailyFolder     = '_daily';
my $weeklyFolder    = '_weekly';
my $monthlyFolder   = '_monthly';

# Check passed configuration file path
my $configFile = $ARGV[0] || '';
if ( !( -e $configFile ) || !( -r $configFile ) ) {
    print "No valid configuration file specified.\n";
    die;
}

# Enable debug "mode"?
my $debugEnabled = $ARGV[1] || 0;

# Read config from configuration file into hash
my %CONFIG;

open(CONFIG, $configFile);
while (<CONFIG>) {
    chomp;                  # no newline
    s/#.*//;                # no comments
    s/^\s+//;               # no leading white
    s/\s+$//;               # no trailing white
    next unless length;     # anything left?
    my ($var, $value) = split(/\s*=\s*/, $_, 2);
    
    if ( my @match = $var =~ /^SOURCE(\d+)(_.*)?$/ ) {
        my $index = $match[0];
        my $var   = "SOURCE" . ($match[1] ? $match[1] : '');
        
        if ( !(exists $CONFIG{$var}) ) {
            $CONFIG{$var} = ();
        }
        $CONFIG{$var}[$index] = $value;
    } else {
        $CONFIG{$var} = $value;
    }
}
close CONFIG;

# print Dumper(\%CONFIG);

my $backupFormat        = '%Y-%m-%d_%H-%M-%S';
my $backupFormatPattern = '^(\d{4})-(\d{2})-(\d{2})_(\d{2})-(\d{2})-(\d{2})$';
my $thisBackupName      = time2str( $backupFormat, time() );

# Whether a source is on a remote server is of no interest to us; rsync will take care of it.
# Only remote targets are a concern, since we need to rotate backups on the target.
# We will use system calls to ssh and pass it a command. Using Net::OpenSSH is troublesome since
# it is not a core module, and there shouldn't be many calls to ssh under normal circumstances anyway:
#  - 3 for rotation folder creation
#  - 1 for main backup list
#  - 1 for main backup list when rotating
#  - 1 (potentially) for moving from main to daily
#  - 1 for daily backup list
#  - 1 (potentially) for moving from daily to weekly
#  - 1 for weekly backup list
#  - 1 (potentially) for moving from weekly to monthly
#  - 1 for monthly backup list
#  - 1 (potentially) for deleting from monthly
# Makes about 12 calls. If this proves to be very slow we can always aggregate commands, though that would
# potentially lead to very long commandlines.
my $remoteTarget        = $CONFIG{'TARGET_HOST'} ? 1 : 0;
# This is the call to SSH made for FS manipulations on the remote machine (not rsync's -e parameter)
my $sshCall             =   "$sshCmd "
                            . ($CONFIG{'SSH_IDENTITY'}  ? '-i "' . $CONFIG{'SSH_IDENTITY'}  . '"' : '') . ' '
                            . ($CONFIG{'SSH_PORT'}      ? '-p "' . $CONFIG{'SSH_PORT'}      . '"' : '') . ' '
                            . $CONFIG{'TARGET_USER'} . '@' . $CONFIG{'TARGET_HOST'};

# Debug purposes
my $cmd = "";

# Get name of last backup
my @mainBackupList = listBackupsInPath($CONFIG{'TARGET'});
my $lastBackupName = pop @mainBackupList;

# Command building
# Start with sources; this will tell us if we need to use SSH
my $remoteSources = 0;
my $sourceCmdline = "";
for my $i (0 .. scalar(@{$CONFIG{'SOURCE'}})) {
    my $source = $CONFIG{'SOURCE'}[$i];
    next unless length($source); # Deals with bad indexing by the user
    
    my $host = exists($CONFIG{'SOURCE_HOST'}[$i]) ? $CONFIG{'SOURCE_HOST'}[$i] : undef;
    my $user = exists($CONFIG{'SOURCE_USER'}[$i]) ? $CONFIG{'SOURCE_USER'}[$i] : undef;
    
    if ( $host ) {
        $remoteSources = 1;
    }
    
    $sourceCmdline .=  " \"" . ($host ? ($user ? $user . '@' : '') . $host . ':' : '') . $source . "\" ";
}

# This is for rsync's -e parameter
my $sshParameter = "-ze 'ssh";
if ( $CONFIG{'SSH_IDENTITY'} ) {
    $sshParameter .= ' -i "' . $CONFIG{'SSH_IDENTITY'} . '" ';
}
if ( $CONFIG{'SSH_PORT'} ) {
    $sshParameter .= ' -p "' . $CONFIG{'SSH_PORT'} . '" ';
}
$sshParameter .= "'";

# Basic
my $rsyncCmdline =  "$rsyncCmd "
                    . " -av "
                    . (($remoteSources || $remoteTarget) ? " $sshParameter " : "")
                    . ($CONFIG{'RELATIVE'} ? " -R " : "")
                    . " --delete --no-perms --no-owner --no-group "
                    . ($CONFIG{'RSYNC_PARAMS'} ? " " . $CONFIG{'RSYNC_PARAMS'} . " " : "")
                    . ""; # Cosmetic

# --link-dest must be relative to the TARGET FOLDER. It does not take user:host@ before the relative path, but figures that out itself
if ( $lastBackupName ) {
    $rsyncCmdline   .= " --link-dest=\"../$lastBackupName\" "
                    . ""; # Cosmetic
}

# Add sources
$rsyncCmdline .= $sourceCmdline;

# Add target
$rsyncCmdline   .= " \"" . ($CONFIG{'TARGET_HOST'} ? ($CONFIG{'TARGET_USER'} ? $CONFIG{'TARGET_USER'} . '@' : '') . $CONFIG{'TARGET_HOST'} . ':' : '') . $CONFIG{'TARGET'} . "/$thisBackupName\" "
                . ""; # Cosmetic

# Execute rsync
print "Executing rsync\n";
debugOut($rsyncCmdline);
system($rsyncCmdline);

# Create rotation folders. Do this AFTER executing the main rsync command; this way, if for some reason the creation
# fails (ssh error or something), at least we'll have the backup.
if ( !$remoteTarget ) {
    mkdir($CONFIG{'TARGET'} . "/$dailyFolder") if !( -d $CONFIG{'TARGET'} . "/$dailyFolder" );
    mkdir($CONFIG{'TARGET'} . "/$weeklyFolder") if !( -d $CONFIG{'TARGET'} . "/$weeklyFolder" );
    mkdir($CONFIG{'TARGET'} . "/$monthlyFolder") if !( -d $CONFIG{'TARGET'} . "/$monthlyFolder" );
} else {
    $cmd = $sshCall . " 'if [[ ! -d \"" . $CONFIG{'TARGET'} . '/' . $dailyFolder . "\" ]]; then mkdir -p \"" . $CONFIG{'TARGET'} . '/' . $dailyFolder . "\" ; fi'";
    debugOut($cmd);
    system($cmd);
        # or die 'Failed to create ' . $CONFIG{'TARGET_USER'} . '@' . $CONFIG{'TARGET_HOST'} . ':' . $CONFIG{'TARGET'} . '/' . $dailyFolder;
    
    $cmd = $sshCall . " 'if [[ ! -d \"" . $CONFIG{'TARGET'} . '/' . $weeklyFolder . "\" ]]; then mkdir -p \"" . $CONFIG{'TARGET'} . '/' . $weeklyFolder . "\" ; fi'";
    debugOut($cmd);
    system($cmd);
        # or die 'Failed to create ' . $CONFIG{'TARGET_USER'} . '@' . $CONFIG{'TARGET_HOST'} . ':' . $CONFIG{'TARGET'} . '/' . $weeklyFolder;
    
    $cmd = $sshCall . " 'if [[ ! -d \"" . $CONFIG{'TARGET'} . '/' . $monthlyFolder . "\" ]]; then mkdir -p \"" . $CONFIG{'TARGET'} . '/' . $monthlyFolder . "\" ; fi'";
    debugOut($cmd);
    system($cmd);
        # or die 'Failed to create ' . $CONFIG{'TARGET_USER'} . '@' . $CONFIG{'TARGET_HOST'} . ':' . $CONFIG{'TARGET'} . '/' . $monthlyFolder;
}

# Execute the rotation
rotateBackups();

sub rotateBackups {
    # Move excess from main to daily according to MAIN_MAX
    moveExcessBackups( $CONFIG{'TARGET'}, $CONFIG{'MAIN_MAX'}, $CONFIG{'TARGET'} . "/$dailyFolder" );
    
    # Delete excess in daily (keep oldest from each day), needs no limit
    groupBackups( $CONFIG{'TARGET'} . "/$dailyFolder", \&getBackupDay );
    
    # Move excess from daily to weekly according to DAILY_MAX
    moveExcessBackups( $CONFIG{'TARGET'} . "/$dailyFolder", $CONFIG{'DAILY_MAX'}, $CONFIG{'TARGET'} . "/$weeklyFolder" );
    
    # Delete excess in weekly (keep oldest from each week), needs no limit
    groupBackups( $CONFIG{'TARGET'} . "/$weeklyFolder", \&getBackupWeek );
    
    # Move excess from weekly to monthly according to WEEKLY_MAX
    moveExcessBackups( $CONFIG{'TARGET'} . "/$weeklyFolder", $CONFIG{'WEEKLY_MAX'}, $CONFIG{'TARGET'} . "/$monthlyFolder" );
    
    # Delete excess in monthly (keep oldest from each month), needs no limit
    groupBackups( $CONFIG{'TARGET'} . "/$monthlyFolder", \&getBackupMonth );
    
    # Delete excess from monthly according to MONTHLY_MAX
    moveExcessBackups( $CONFIG{'TARGET'} . "/$monthlyFolder", $CONFIG{'MONTHLY_MAX'}, '' );
}

# Return the list of backups in the specified path (can be remote, will check for
# $remoteTarget and use $sshCall) and checks entry format to return only backup folders
#
# Args:
# $path
sub listBackupsInPath {
    my $path = $_[0];
    my @backupList = ();

    if ( !$remoteTarget ) {
        opendir(D, $path);
        my @items = readdir(D);
        closedir(D);

        foreach (@items) {
            # Is it a directory?
            if ( -d "$path/$_" ) {
                # Is it a backup folder (excludes junk and rotation folders)
                if ( /$backupFormatPattern/ ) {
                    push(@backupList, $_);
                }
            }
        }
    } else {
        # Only list directories. Use find as the wildcard used with ls will not be expanded unless
        # we spawn a new shell, which results in all kinds of pain with quotes. Don't forget -maxdepth!
        # Use backticks here as system doesn't capture output
        $cmd = $sshCall . " 'find \"" . $path . "/\" -type d -maxdepth 1'";
        debugOut($cmd);
        my @items = `$cmd`;
        
        # The first result using find is the directory itself, we don't want that.
        shift @items;
        
        foreach (@items) {
            # Remove any ending characters
            chomp;
            
            # Basename and remove ending slash added by ls
            $_ =~ s/^.*?\/([^\/]+)\/?$/$1/;
            
            # Is it a backup folder (excludes junk and rotation folders)
            if ( /$backupFormatPattern/ ) {
                push(@backupList, $_);
            }
        }
    }
 
    @backupList = sort @backupList;

    return @backupList;
}

# Count all backups in $source. If count exceeds $sourceMax, move excess backups to $target, if
# specified, or delete them
#
# Args:
# $source
# $sourceMax
# $target
sub moveExcessBackups {
    my ( $source, $sourceMax, $target ) = @_;

    print( "Handling excess backups (> $sourceMax) in '$source'\n" );

    my @backupList  = listBackupsInPath($source);
    @backupList     = sort @backupList;

    if ( scalar(@backupList) > $sourceMax ) {
        for ( my $i = 0; $i < (scalar(@backupList) - $sourceMax); $i++ ) {
            my $currentBackup = $backupList[$i];

            # If a target folder has been specified, move excess backups to that folder. If not, delete them
            if ( $target ) {
                print( "Moving $currentBackup to '$target'\n" );
                
                if ( !$remoteTarget ) {
                    move( "$source/$currentBackup", "$target/$currentBackup" );
                } else {
                    $cmd = $sshCall . " 'mv \"" . $source . '/' . $currentBackup . "\" \"" . $target . '/' . $currentBackup . "\"'";
                    debugOut($cmd);
                    system($cmd);
                }
            } else {
                print( "Deleting $currentBackup\n" );
                
                if ( !$remoteTarget ) {
                    remove_tree( "$source/$currentBackup", {error => \my $err}  );
                } else {
                    $cmd = $sshCall . " 'rm -rf \"" . $source . '/' . $currentBackup . "\"'";
                    debugOut($cmd);
                    system($cmd);
                }
            }
        }
    }
}

# Group backups in $source so that only one backup remains. The second argument is a reference to
# the function that returns the second identifier for the backup folder name; e.g. month or year.
# This can be hugely improved, no doubt, and has edgy cases where it breaks down, no doubt, but 
# was quick to implement and works so far.
#
# Args:
# $source
# $secondIdentifierCall
sub groupBackups {
    my ( $source, $secondIdentifierCall ) = @_;

    print( "Grouping excess backups in '$source' by return value of '" . svref_2object($secondIdentifierCall)->GV->NAME . "'\n" );

    my @backupList  = listBackupsInPath($source);
    @backupList     = sort @backupList;
    
    # This works as follows:
    # Walk through the backup list in reverse, e.g. the oldest backup will be on top.
    # Get the backup year, and the second identifier with the passed function reference
    # (day, week, month). Set a high beginning year and second identifier. For each
    # entry, check if either the year or second identifier are inferior to the previous entry's
    # value(s). If not, the current backup is for the same year and <second identifier> as the
    # previous backup, so delete it. If so, we have move to the previous <whatever>; set
    # our control variables to the current values and repeat.
    my $highYear = 999999;
    my $highNum  = 999999;
 
    foreach my $currentBackup (reverse @backupList) {
        my $currentYear = getBackupYear($currentBackup);
        my $currentNum  = $secondIdentifierCall->($currentBackup);

        if ( $currentYear < $highYear ) {
            $highYear   = $currentYear;
            $highNum    = $currentNum;
        } else {
            if ( $currentNum < $highNum ) {
                $highNum  = $currentNum;
            } else {
                print( "Deleting excess backup $currentBackup\n" );
                
                if ( !$remoteTarget ) {
                    remove_tree( "$source/$currentBackup" );
                } else {
                    $cmd = $sshCall . " 'rm -rf \"" . $source . '/' . $currentBackup . "\"'";
                    debugOut($cmd);
                    system($cmd);
                }
            }
        }
    }
}

# Unused
#
# Args:
# $left
# $right
sub compareBackupTime {
    my ($left, $right) = @_;

    if ( backupNameToTime($left) > backupNameToTime($right) ) {
        return 1;
    } elsif (backupNametoTime($left) < backupNameToTime($right) ) {
        return -1;
    } else {
        return 0;
    }
}

# Return the day of month of the passed backup folder name
#
# Args:
# $backupName
sub getBackupDay {
    my ($backupName) = @_;

    if ( $backupName =~ m/$backupFormatPattern/ ) {
        my $time = backupNameToTime($backupName);
        return time2str( "%j", $time );
    }

    return undef;
}

# Return the week of year of the passed backup folder name
#
# Args:
# $backupName
sub getBackupWeek {
    my ($backupName) = @_;

    if ( $backupName =~ m/$backupFormatPattern/ ) {
        my $time = backupNameToTime($backupName);
        return time2str( "%W", $time );
    }

    return undef;
}

# Return the month number of the passed backup folder name
#
# Args:
# $backupName
sub getBackupMonth {
    my ($backupName) = @_;

    if ( $backupName =~ m/$backupFormatPattern/ ) {
        return $2;
    }

    return undef;
}

# Return the year of the passed backup folder name
#
# Args:
# $backupName
sub getBackupYear {
    my ($backupName) = @_;

    if ( $backupName =~ m/$backupFormatPattern/ ) {
        return $1;
    }

    return undef;
}

# Converts the backup folder name to one readable by str2time, then calls str2time
# on it and returns the resulting timestamp
#
# Args:
# $backupName
sub backupNameToTime {
    my ($backupName) = @_;

    $backupName =~ s/$backupFormatPattern/$1-$2-$3 $4:$5:$6/;

    return str2time($backupName);
}

sub debugOut {
    my ($msg) = @_;
    
    if ( $debugEnabled ) {
        print "DEBUG: $msg\n";
    }
}
