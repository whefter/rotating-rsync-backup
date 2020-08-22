package main

import (
	"crypto/tls"
	"fmt"

	"gopkg.in/gomail.v2"
)

func SendStatusMail(options *Options) {
	logContent := Log.String()

	if options.statusMailOptions.smtpHost == "" ||
		options.statusMailOptions.smtpPort == 0 ||
		options.statusMailOptions.smtpUsername == "" ||
		options.statusMailOptions.smtpPassword == "" {
		if len(options.statusMailOptions.recipients) > 0 {
			Log.Warn.Println("Status mail recipients given, but SMTP configuration is incomplete.")
		} else {
			Log.Debug.Println("No SMTP configuration given.")
		}

		return
	}

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
	m.SetHeader("From", options.statusMailOptions.from)
	m.SetHeader("To", options.statusMailOptions.recipients...)
	m.SetHeader("Subject", fmt.Sprintf("rotating-rsync-backup [%s]: %s", logLevel, options.profileName))
	m.SetBody("text/plain", logContent)

	d := gomail.NewDialer(
		options.statusMailOptions.smtpHost,
		options.statusMailOptions.smtpPort,
		options.statusMailOptions.smtpUsername,
		options.statusMailOptions.smtpPassword,
	)
	if options.statusMailOptions.smtpInsecure {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if err := d.DialAndSend(m); err != nil {
		panic(fmt.Sprintf("Error while sending status mail: %v", err))
	}
}
