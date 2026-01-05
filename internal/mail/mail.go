package mail

import (
	"fmt"
	"net/smtp"
	"os"
)

const (
	smtpHost          = "smtp.mail.yahoo.com"
	smtpPort          = "587"
	registeredSubject = "Bienvenido a Aprendiendo con Otero Ediciones"
)

func SendWelcomeEmail(toEmail, fullName, password string) error {
	// Pull from environment
	from := os.Getenv("EMAIL_FROM")
	appPass := os.Getenv("EMAIL_PASSWORD")

	if from == "" || appPass == "" {
		return fmt.Errorf("email configuration missing in environment")
	}

	body := fmt.Sprintf(
		"Hola %s,\n\nTu cuenta ha sido creada con éxito.\n\nUsuario: %s\nPassword: %s\n\nPuedes ingresar aquí: https://www.aprendiendoconoteroediciones.com/login",
		fullName, toEmail, password,
	)

	message := []byte("Subject: " + registeredSubject + "\r\n" +
		"To: " + toEmail + "\r\n" +
		"From: " + from + "\r\n" +
		"MIME-version: 1.0;\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\";\r\n" +
		"\r\n" +
		body)

	auth := smtp.PlainAuth("", from, appPass, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{toEmail}, message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
