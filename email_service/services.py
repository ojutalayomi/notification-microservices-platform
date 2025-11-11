import os
from smtplib import SMTP
from email.mime.text import MIMEText

def send_email(to: str, subject: str, body: str):
    """
    Minimal SMTP sender for demo. Replace with SendGrid/AWS SES/other provider in prod.
    """
    smtp_host = os.getenv("SMTP_HOST", "smtp.gmail.com")
    smtp_port = int(os.getenv("SMTP_PORT", 587))
    smtp_user = os.getenv("SMTP_USER")
    smtp_pass = os.getenv("SMTP_PASS")
    sender = os.getenv("EMAIL_SENDER", "noreply@example.com")

    msg = MIMEText(body, "html")
    msg["Subject"] = subject
    msg["From"] = sender
    msg["To"] = to

    with SMTP(smtp_host, smtp_port) as smtp:
        smtp.starttls()
        if smtp_user and smtp_pass:
            smtp.login(smtp_user, smtp_pass)
        smtp.sendmail(sender, [to], msg.as_string())
