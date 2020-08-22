package main

import (
	"crypto/tls"
	"fmt"

	"gopkg.in/gomail.v2"
)

// SendReportMail sends a report mail to the recipients configured in the options using the
// configured SMTP server, containing the full log output up until the function call
func SendReportMail(options *Options) {
	logContent := Log.String()

	if options.ReportOptions.smtpHost == "" ||
		options.ReportOptions.smtpPort == 0 ||
		options.ReportOptions.smtpUsername == "" ||
		options.ReportOptions.smtpPassword == "" {
		if len(options.ReportOptions.recipients) > 0 {
			Log.Warn.Println("Status mail recipients given, but SMTP configuration is incomplete.")
		} else {
			Log.Debug.Println("No SMTP configuration given.")
		}

		return
	}

	Log.Info.Printf("Sending report mail to: %v", options.ReportOptions.recipients)

	var logLevel string
	if fatalBuf.Len() > 0 {
		logLevel = "FATAL"
	} else if errorBuf.Len() > 0 {
		logLevel = "ERROR"
	} else if errorBuf.Len() > 0 {
		logLevel = "WARN"
	} else {
		logLevel = "INFO"
	}

	m := gomail.NewMessage()
	m.SetHeader("From", options.ReportOptions.from)
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
