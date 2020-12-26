package main

import (
	"crypto/tls"
	"fmt"
	"os/user"

	"github.com/Showmax/go-fqdn"
	"gopkg.in/gomail.v2"
)

// SendReportMail sends a report mail to the recipients configured in the options using the
// configured SMTP server, containing the full log output up until the function call
func SendReportMail(options *Options) {
	logContent := Log.String()

	if options.ReportOptions.smtpHost == "" ||
		options.ReportOptions.smtpPort == 0 {
		if len(options.ReportOptions.recipients) > 0 {
			Log.Warn.Println("Status mail recipients given, but SMTP configuration is incomplete (host/port missing/invalid).")
		} else {
			Log.Debug.Println("No SMTP configuration given.")
		}

		return
	}

	var from string
	if options.ReportOptions.from == "" {
		user, err := user.Current()
		if err != nil {
			panic(fmt.Sprintf("Error obtaining current user (for From: value): %v", err))
		}
		fqdn, err := fqdn.FqdnHostname()
		if err != nil {
			panic(fmt.Sprintf("Error obtaining FqdnHostname (for From: value): %v", err))
		}
		from = fmt.Sprintf("%s@%s", user.Username, fqdn)
	} else {
		from = options.ReportOptions.from
	}

	Log.Info.Printf("Sending report mail to: %v", options.ReportOptions.recipients)

	logLevel := Log.MaxLogLevel()

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", options.ReportOptions.recipients...)
	m.SetHeader("Subject", fmt.Sprintf("rotating-rsync-backup [%s]: %s", logLevel, options.profileName))
	m.SetBody("text/plain", logContent)

	d := gomail.NewDialer(
		options.ReportOptions.smtpHost,
		int(options.ReportOptions.smtpPort),
		options.ReportOptions.smtpUsername,
		options.ReportOptions.smtpPassword,
	)
	if options.ReportOptions.smtpInsecure {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if err := d.DialAndSend(m); err != nil {
		panic(fmt.Sprintf("Error while sending report mail: %v", err))
	}
}
