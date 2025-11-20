import os
from smtplib import SMTP, SMTP_SSL
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from dotenv import load_dotenv

load_dotenv()

def send_email(to_email: str, subject: str, body: str):
    """
    Send an email using SMTP with support for multiple connection methods.
    
    Supports:
    - Port 587 (STARTTLS) - Default
    - Port 465 (SSL/TLS) - Alternative
    - Port 25 (Standard SMTP) - Fallback
    
    Args:
        to_email: Recipient email address
        subject: Email subject
        body: Email body (can be HTML)
    
    Raises:
        Exception: If email fails to send
    """
    # SMTP configuration
    smtp_host = os.getenv("SMTP_HOST", "smtp.gmail.com")
    smtp_port = int(os.getenv("SMTP_PORT", "587"))
    smtp_user = os.getenv("SMTP_USER")
    smtp_pass = os.getenv("SMTP_PASS")
    sender = os.getenv("EMAIL_SENDER", "noreply@example.com")
    smtp_use_ssl = os.getenv("SMTP_USE_SSL", "false").lower() == "true"
    smtp_use_tls = os.getenv("SMTP_USE_TLS", "true").lower() == "true"
    
    # Create email message
    msg = MIMEMultipart('alternative')
    msg['Subject'] = subject
    msg['From'] = sender
    msg['To'] = to_email
    
    # Add HTML body
    html_part = MIMEText(body, 'html')
    msg.attach(html_part)

    # Send email with fallback options
    print(f"[smtp] Connecting to {smtp_host}:{smtp_port}...")
    
    # Try different connection methods based on port and configuration
    connection_methods = []
    
    if smtp_port == 465 or smtp_use_ssl:
        # Port 465 uses SSL/TLS from the start
        connection_methods.append(("ssl", smtp_port))
    elif smtp_port == 587:
        # Port 587 uses STARTTLS
        connection_methods.append(("starttls", 587))
        # Fallback to SSL if STARTTLS fails
        connection_methods.append(("ssl", 465))
    else:
        # Other ports (e.g., 25)
        connection_methods.append(("starttls", smtp_port))
        if smtp_use_tls:
            connection_methods.append(("ssl", 465))
    
    last_error = None
    
    for method, port in connection_methods:
        try:
            print(f"[smtp] Trying {method.upper()} on port {port}...")
            
            if method == "ssl":
                # Use SMTP_SSL for direct SSL/TLS connection
                server = SMTP_SSL(smtp_host, port, timeout=30)
            else:
                # Use regular SMTP with STARTTLS
                import socket
                sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                sock.settimeout(30)
                sock.connect((smtp_host, port))
                
                server = SMTP()
                server.set_debuglevel(0)
                server.sock = sock
                
                if smtp_use_tls:
                    server.starttls()
            
            if smtp_user and smtp_pass:
                print(f"[smtp] Logging in as {smtp_user}...")
                server.login(smtp_user, smtp_pass)
            
            print(f"[smtp] Sending email to {to_email}...")
            server.sendmail(sender, [to_email], msg.as_string())
            print(f"[smtp] Email sent successfully via {method.upper()} on port {port}!")
            server.quit()
            return  # Success, exit function
            
        except Exception as e:
            last_error = e
            print(f"[smtp] Failed with {method.upper()} on port {port}: {e}")
            continue
    
    # All methods failed
    print(f"[smtp] All connection methods failed. Last error: {last_error}")
    import traceback
    traceback.print_exc()
    raise Exception(f"Failed to send email after trying all methods. Last error: {last_error}")