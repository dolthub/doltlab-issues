package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

const (
	AuthPlain       = "plain"
	AuthAnonymous   = "anonymous"
	AuthExternal    = "external"
	AuthOauthBearer = "oauthbearer"
	AuthDisable     = "disable"
	AuthLogin       = "login"
)

var sender = flag.String("from", "", "email address of sender")
var recipient = flag.String("to", "", "email address of recipient")
var host = flag.String("host", "", "smtp host")
var port = flag.Int("port", 0, "smtp port")
var auth = flag.String("auth", "", fmt.Sprintf("authentication method, one of '%s', '%s', '%s', '%s', '%s', '%s'", AuthPlain, AuthLogin, AuthAnonymous, AuthExternal, AuthOauthBearer, AuthDisable))
var implicitTLSArg = flag.Bool("implicit-tls", false, "use TLS Wrapper connection instead of STARTTLS connection")
var insecure = flag.Bool("insecure", false, "if used, sets TLS Config InsecureSkipVerify to true")
var username = flag.String("username", "", "smtp username")
var password = flag.String("password", "", "smtp password")
var identity = flag.String("identity", "", "smtp identity")
var token = flag.String("token", "", "oauthbearer token")
var trace = flag.String("trace", "", "trace argument passed to anonymous smtp client")
var subject = flag.String("subject", "Testing SMTP Server Connection", "email subject")
var message = flag.String("message", "This is a test email message sent with smtp_connection_helper!", "message to send to recipient")
var clientHostnameArg = flag.String("client-hostname", "localhost", "specifies client hostname passed SMTP server during HELO or EHLO")
var help = flag.Bool("help", false, "print 'smtp_connection_helper' usage")

func main() {
	flag.Parse()

	if *help {
		printUsage()
	} else {

		if *host == "" || *port < 1 || *sender == "" || *recipient == "" || *subject == "" || *message == "" || *clientHostnameArg == "" {
			printUsage()
			log.Fatal(errors.New("one or all required arguments not supplied, must supply: --host, --port, --from, --to, --subject, --message, --client-hostname"))
		}

		var err error
		switch {
		case *auth == AuthPlain:
			err = sendEmailPlainAuth(*host, *port, *identity, *username, *password, *sender, *recipient, *subject, *message, *clientHostnameArg, *implicitTLSArg, *insecure)
		case *auth == AuthLogin:
			err = sendEmailLoginAuth(*host, *port, *username, *password, *sender, *recipient, *subject, *message, *clientHostnameArg, *implicitTLSArg, *insecure)
		case *auth == AuthAnonymous:
			err = sendEmailAnonymousAuth(*host, *port, *trace, *sender, *recipient, *subject, *message, *clientHostnameArg, *implicitTLSArg, *insecure)
		case *auth == AuthExternal:
			err = sendEmailExternalAuth(*host, *port, *identity, *sender, *recipient, *subject, *message, *clientHostnameArg, *implicitTLSArg, *insecure)
		case *auth == AuthOauthBearer:
			err = sendEmailOauthBearerAuth(*host, *port, *username, *token, *sender, *recipient, *subject, *message, *clientHostnameArg, *implicitTLSArg, *insecure)
		case *auth == AuthDisable:
			err = sendEmailDisableAuth(*host, *port, *sender, *recipient, *subject, *message, *clientHostnameArg, *implicitTLSArg, *insecure)
		default:
			err = errors.New(fmt.Sprintf("Unsupported auth method: %s", *auth))
		}

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Successfully sent email!")
	}
}

func sendEmailPlainAuth(host string, port int, identity, username, password, sender, recipient, subject, message, clientHostname string, implicitTLS, insecure bool) error {
	if username == "" || password == "" {
		return errors.New(fmt.Sprintf("username and password must not be empty for auth %s", AuthPlain))
	}

	smtpClient, err := getSmtpClient(host, port, implicitTLS, insecure)
	if err != nil {
		return err
	}

	authClient := sasl.NewPlainClient(identity, username, password)
	return sendEmail(smtpClient, authClient, sender, recipient, subject, message, clientHostname, AuthPlain)
}

func sendEmailLoginAuth(host string, port int, username, password, sender, recipient, subject, message, clientHostname string, implicitTLS, insecure bool) error {
	if username == "" || password == "" {
		return errors.New(fmt.Sprintf("username and password must not be empty for auth %s", AuthLogin))
	}

	smtpClient, err := getSmtpClient(host, port, implicitTLS, insecure)
	if err != nil {
		return err
	}

	authClient := sasl.NewLoginClient(username, password)
	return sendEmail(smtpClient, authClient, sender, recipient, subject, message, clientHostname, AuthLogin)
}

func sendEmailAnonymousAuth(host string, port int, trace, sender, recipient, subject, message, clientHostname string, implicitTLS, insecure bool) error {
	smtpClient, err := getSmtpClient(host, port, implicitTLS, insecure)
	if err != nil {
		return err
	}

	authClient := sasl.NewAnonymousClient(trace)
	return sendEmail(smtpClient, authClient, sender, recipient, subject, message, clientHostname, AuthAnonymous)
}

func sendEmailExternalAuth(host string, port int, identity, sender, recipient, subject, message, clientHostname string, implicitTLS, insecure bool) error {
	smtpClient, err := getSmtpClient(host, port, implicitTLS, insecure)
	if err != nil {
		return err
	}

	authClient := sasl.NewExternalClient(identity)
	return sendEmail(smtpClient, authClient, sender, recipient, subject, message, clientHostname, AuthExternal)
}

func sendEmailOauthBearerAuth(host string, port int, username, token, sender, recipient, subject, message, clientHostname string, implicitTLS, insecure bool) error {
	if username == "" || token == "" {
		return errors.New(fmt.Sprintf("username and token must not be empty for auth %s", AuthOauthBearer))
	}

	smtpClient, err := getSmtpClient(host, port, implicitTLS, insecure)
	if err != nil {
		return err
	}

	authClient := sasl.NewOAuthBearerClient(&sasl.OAuthBearerOptions{
		Username: username,
		Token:    token,
		Host:     host,
		Port:     port,
	})

	return sendEmail(smtpClient, authClient, sender, recipient, subject, message, clientHostname, AuthOauthBearer)
}

func sendEmailDisableAuth(host string, port int, sender, recipient, subject, message, clientHostname string, implicitTLS, insecure bool) error {
	smtpClient, err := getSmtpClient(host, port, implicitTLS, insecure)
	if err != nil {
		return err
	}
	return sendEmail(smtpClient, nil, sender, recipient, subject, message, clientHostname, AuthDisable)
}

func getSmtpClient(host string, port int, implicitTLS, insecure bool) (*smtp.Client, error) {
	if implicitTLS {
		return newImplicitTLSClient(host, port, insecure)
	}
	return newExplicitStartTLSClient(host, port)
}

func newImplicitTLSClient(host string, port int, insecure bool) (*smtp.Client, error) {
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", host, port), &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: insecure,
	})
	if err != nil {
		return nil, err
	}
	return smtp.NewClient(conn), nil
}

func newExplicitStartTLSClient(host string, port int) (*smtp.Client, error) {
	return smtp.Dial(fmt.Sprintf("%s:%d", host, port))
}

func sendEmail(smtpClient *smtp.Client, authClient sasl.Client, sender, recipient, subject, message, clientHostname string, method string) (err error) {
	defer func() {
		rerr := smtpClient.Close()
		if err == nil {
			if rerr.Error() != "use of closed network connection" {
				err = rerr
			}

		}
	}()

	body := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	body += fmt.Sprintf("From: %s\r\n", sender)
	body += fmt.Sprintf("To: %s\r\n", recipient)
	body += fmt.Sprintf("Subject: %s\r\n", subject)
	body += fmt.Sprintf("\r\n%s\r\n", message)

	fmt.Println("Sending email with auth method:", method)

	err = smtpClient.Hello(clientHostname)
	if err != nil {
		return
	}

	if ok, _ := smtpClient.Extension("STARTTLS"); ok {
		if err = smtpClient.StartTLS(nil); err != nil {
			return
		}
	}

	if authClient != nil {
		ok, _ := smtpClient.Extension("AUTH")
		if !ok {
			return errors.New("smtp: server doesn't support AUTH")
		} else {
			err = smtpClient.Auth(authClient)
			if err != nil {
				return
			}
		}
	}

	err = smtpClient.Mail(sender, nil)
	if err != nil {
		return
	}

	err = smtpClient.Rcpt(recipient, nil)
	if err != nil {
		return
	}

	var wc io.WriteCloser
	wc, err = smtpClient.Data()
	if err != nil {
		return
	}

	_, err = io.Copy(wc, strings.NewReader(body))
	if err != nil {
		return
	}

	err = wc.Close()
	if err != nil {
		return
	}

	return smtpClient.Quit()
}

func printUsage() {
	fmt.Println("")
	fmt.Println("")
	fmt.Println("'smtp_connection_helper' is a simple tool used to ensure you can successfully connect to an smtp server.")
	fmt.Println("If the connection is successful, this tool will send a test email to a single recipient from a single sender.")
	fmt.Println("By default 'smtp_connection_helper' will attempt to connect to the SMTP server with STARTTLS. To use implicit TLS, use --implicit-tls")
	fmt.Println("")
	fmt.Println(fmt.Sprintf("Usage:"))
	fmt.Println("")
	fmt.Println(fmt.Sprintf("./smtp_connection_helper \\"))
	fmt.Println("--host <smtp hostname> \\")
	fmt.Println("--port <smtp port> \\")
	fmt.Println("--from <email address> \\")
	fmt.Println("--to <email address> \\")
	fmt.Println("--message {This is a test email message sent with smtp_connection_helper!} \\")
	fmt.Println("--subject {Testing SMTP Server Connection} \\")
	fmt.Println("--client-hostname {localhost} \\")
	fmt.Println(fmt.Sprintf("--auth <%s|%s|%s|%s|%s|%s> \\", AuthPlain, AuthLogin, AuthExternal, AuthAnonymous, AuthOauthBearer, AuthDisable))
	fmt.Println("[--username smtp username] \\")
	fmt.Println("[--password smtp password] \\")
	fmt.Println("[--token smtp oauth token] \\")
	fmt.Println("[--identity smtp identity] \\")
	fmt.Println("[--trace anonymous trace] \\")
	fmt.Println("[--implicit-tls] \\")
	fmt.Println("[--insecure]")
	fmt.Println("")
	fmt.Println("")
}
