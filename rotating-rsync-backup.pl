#!/usr/bin/env perl

# rotating-rsync-backup v0.1
#
# Usage: rotating-rsync-backup.pl /path/to/config.conf
#
# Rsync utility script that takes a configuration file path as first argument. Backup
# folders are rotated, with a configurable number of daily/weekly/monthly backup folders
# being kept. Hardlinks are used where possible.
#
# The MIT License (MIT)
#
# Copyright (c) 2014-2015 William Hefter <william@whefter.de>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

use warnings;
use strict;

use File::Copy;
use File::Path 'remove_tree';
use Date::Parse;
use Date::Format;
use Data::Dumper;

my $rsyncCmd = `which rsync`;
chomp $rsyncCmd;

my $dailyFolder = '_daily';
my $weeklyFolder = '_weekly';
my $monthlyFolder = '_monthly';

# Read config
my $configFile = $ARGV[0] || '';
if ( !( -e $configFile ) || !( -r $configFile ) ) {
    print "No valid configuration file specified.\n";
    die;
}

my %CONFIG;
open(CONFIG, $configFile);
while (<CONFIG>) {
    chomp;                  # no newline
    s/#.*//;                # no comments
    s/^\s+//;               # no leading white
    s/\s+$//;               # no trailing white
    next unless length;     # anything left?
    my ($var, $value) = split(/\s*=\s*/, $_, 2);

    if ( exists $CONFIG{$var} ) {
        if ( ref $CONFIG{$var} eq 'ARRAY' ) {
            push @{$CONFIG{$var}}, $value;
        } else {
            $CONFIG{$var} = [($CONFIG{$var},  $value)];
        }
    } else {
        $CONFIG{$var} = $value;
    }
}
close CONFIG;

# print Dumper(\%CONFIG);

my $backupFormat = '%Y-%m-%d_%H-%M-%S';
my $backupFormatPattern = '^(\d{4})-(\d{2})-(\d{2})_(\d{2})-(\d{2})-(\d{2})$';

mkdir($CONFIG{'TARGET'} . "/$dailyFolder") if !( -d $CONFIG{'TARGET'} . "/$dailyFolder" );
mkdir($CONFIG{'TARGET'} . "/$weeklyFolder") if !( -d $CONFIG{'TARGET'} . "/$weeklyFolder" );
mkdir($CONFIG{'TARGET'} . "/$monthlyFolder") if !( -d $CONFIG{'TARGET'} . "/$monthlyFolder" );

my @mainBackupList = listBackupsInPath($CONFIG{'TARGET'});
my $lastBackupName = pop @mainBackupList;

if ( ref $CONFIG{'SOURCES'} ne 'ARRAY' ) {
    $CONFIG{'SOURCES'} = [($CONFIG{'SOURCES'})];
}

my $thisBackupName = time2str( $backupFormat, time() );
 
my $remoteParams = "";
if ( $CONFIG{'SSH'} ) {
    $remoteParams = "-ze ssh";
}

my $cmd =   "rsync "
            . " -av "
            . $remoteParams
            . ($CONFIG{'RELATIVE'} ? " -R " : "" )
            . " --delete --no-perms --no-owner --no-group "
            . " --link-dest=\"" . $CONFIG{'TARGET'} . "/$lastBackupName\" "
            . "";

foreach my $source (@{$CONFIG{'SOURCES'}}) {
    $cmd .=  " \"$source\" ";
}

$cmd .= " \"" . $CONFIG{'TARGET'} . "/$thisBackupName\" "
        . "";

#print "$cmd\n"; 
system($cmd);

rotateBackups();

sub rotateBackups {
    moveExcessBackups( $CONFIG{'TARGET'}, $CONFIG{'MAIN_MAX'}, $CONFIG{'TARGET'} . "/$dailyFolder" );
    groupBackups( $CONFIG{'TARGET'} . "/$dailyFolder", \&getBackupDay );
    moveExcessBackups( $CONFIG{'TARGET'} . "/$dailyFolder", $CONFIG{'DAILY_MAX'}, $CONFIG{'TARGET'} . "/$weeklyFolder" );
    groupBackups( $CONFIG{'TARGET'} . "/$weeklyFolder", \&getBackupWeek );
    moveExcessBackups( $CONFIG{'TARGET'} . "/$weeklyFolder", $CONFIG{'WEEKLY_MAX'}, $CONFIG{'TARGET'} . "/$monthlyFolder" );
    groupBackups( $CONFIG{'TARGET'} . "/$monthlyFolder", \&getBackupMonth );
    moveExcessBackups( $CONFIG{'TARGET'} . "/$monthlyFolder", $CONFIG{'MONTHLY_MAX'}, '' );
}

sub listBackupsInPath {
    my $path = $_[0];
    my @backupList = ();

    opendir(D, $path);
    my @items = readdir(D);
    closedir(D);

    foreach (@items) {
        if ( -d "$path/$_" ) {
            if ( /$backupFormatPattern/ ) {
                push(@backupList, $_);
            }
        }
    }
 
    @backupList = sort @backupList;

    return @backupList;
}

sub moveExcessBackups {
    my ( $source, $sourceMax, $target ) = @_;

    print( "Handling excess backups (> $sourceMax) from '$source' to '$target'\n" );

    my @backupList = listBackupsInPath($source);

    if ( scalar(@backupList) > $sourceMax ) {
        for ( my $i = 0; $i < (scalar(@backupList) - $sourceMax); $i++ ) {
            my $currentBackup = $backupList[$i];

            if ( $target ) {
                print( "Moving $currentBackup]\n" );
                move( "$source/$currentBackup", "$target/$currentBackup" );
            } else {
                print( "Deleting $currentBackup\n" );
                remove_tree( "$source/$currentBackup", {error => \my $err}  );
                print $err."\n";
            }
        }
    }
}

sub groupBackups {
    my ( $source, $secondIdentifierCall ) = @_;

    print( "Grouping excess backups in '$source'\n" );

    my @backupList = listBackupsInPath($source);
    my $highYear = 999999;
    my $highNum  = 999999;
 
    foreach my $currentBackup (reverse @backupList) {
        my $currentYear = getBackupYear($currentBackup);
        my $currentNum = $secondIdentifierCall->($currentBackup);

        if ( $currentYear < $highYear ) {
            $highYear = $currentYear;
            $highNum = $currentNum;
        } else {
            if ( $currentNum < $highNum ) {
                $highNum = $currentNum;
            } else {
                print( "Deleting excess backup $currentBackup\n" );
                remove_tree( "$source/$currentBackup" );
            }
        }
    }
}

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

sub getBackupDay {
    my ($backupName) = @_;

    if ( $backupName =~ m/$backupFormatPattern/ ) {
        my $time = backupNameToTime($backupName);
        return time2str( "%j", $time );
    }

    return undef;
}

sub getBackupWeek {
    my ($backupName) = @_;

    if ( $backupName =~ m/$backupFormatPattern/ ) {
        my $time = backupNameToTime($backupName);
        return time2str( "%W", $time );
    }

    return undef;
}

sub getBackupMonth {
    my ($backupName) = @_;

    if ( $backupName =~ m/$backupFormatPattern/ ) {
        return $2;
    }

    return undef;
}

sub getBackupYear {
    my ($backupName) = @_;

    if ( $backupName =~ m/$backupFormatPattern/ ) {
        return $1;
    }

    return undef;
}

sub backupNameToTime {
    my ($backupName) = @_;

    $backupName =~ s/$backupFormatPattern/$1-$2-$3 $4:$5:$6/;

    return str2time($backupName);
}
