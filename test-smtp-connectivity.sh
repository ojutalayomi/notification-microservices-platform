#!/bin/bash

echo "=========================================="
echo "SMTP CONNECTIVITY TEST"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test 1: Check SMTP environment variables
echo -e "${YELLOW}=== Test 1: SMTP Configuration ===${NC}"
echo ""

SMTP_HOST=$(docker exec hng-email-service printenv SMTP_HOST 2>/dev/null || echo "NOT SET")
SMTP_PORT=$(docker exec hng-email-service printenv SMTP_PORT 2>/dev/null || echo "NOT SET")
SMTP_USER=$(docker exec hng-email-service printenv SMTP_USER 2>/dev/null || echo "NOT SET")
SMTP_PASS=$(docker exec hng-email-service printenv SMTP_PASS 2>/dev/null || echo "NOT SET")
EMAIL_SENDER=$(docker exec hng-email-service printenv EMAIL_SENDER 2>/dev/null || echo "NOT SET")

echo "SMTP_HOST: $SMTP_HOST"
echo "SMTP_PORT: $SMTP_PORT"
echo "SMTP_USER: $SMTP_USER"
echo "SMTP_PASS: ${SMTP_PASS:0:3}*** (hidden)"
echo "EMAIL_SENDER: $EMAIL_SENDER"
echo ""

# Test 2: Test port connectivity from container
echo -e "${YELLOW}=== Test 2: Port Connectivity from Container ===${NC}"
echo ""

if [ "$SMTP_HOST" != "NOT SET" ] && [ "$SMTP_PORT" != "NOT SET" ]; then
    echo "Testing connection to $SMTP_HOST:$SMTP_PORT..."
    
    # Test with Python socket
    RESULT=$(docker exec hng-email-service python3 -c "
import socket
import sys
try:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(5)
    result = sock.connect_ex(('$SMTP_HOST', $SMTP_PORT))
    sock.close()
    if result == 0:
        print('SUCCESS')
    else:
        print(f'FAILED: Error code {result}')
    sys.exit(0 if result == 0 else 1)
except Exception as e:
    print(f'ERROR: {e}')
    sys.exit(1)
" 2>&1)
    
    if echo "$RESULT" | grep -q "SUCCESS"; then
        echo -e "${GREEN}✓ Port $SMTP_PORT is reachable${NC}"
    else
        echo -e "${RED}✗ Port $SMTP_PORT is NOT reachable${NC}"
        echo "   Result: $RESULT"
    fi
else
    echo -e "${RED}✗ SMTP configuration not set${NC}"
fi

echo ""

# Test 3: Test from host machine
echo -e "${YELLOW}=== Test 3: Port Connectivity from Host ===${NC}"
echo ""

if [ "$SMTP_HOST" != "NOT SET" ] && [ "$SMTP_PORT" != "NOT SET" ]; then
    echo "Testing connection from host to $SMTP_HOST:$SMTP_PORT..."
    
    if command -v nc >/dev/null 2>&1; then
        if timeout 5 nc -zv "$SMTP_HOST" "$SMTP_PORT" 2>&1 | grep -q "succeeded"; then
            echo -e "${GREEN}✓ Port $SMTP_PORT is reachable from host${NC}"
        else
            echo -e "${RED}✗ Port $SMTP_PORT is NOT reachable from host${NC}"
        fi
    elif command -v telnet >/dev/null 2>&1; then
        if timeout 5 bash -c "echo > /dev/tcp/$SMTP_HOST/$SMTP_PORT" 2>/dev/null; then
            echo -e "${GREEN}✓ Port $SMTP_PORT is reachable from host${NC}"
        else
            echo -e "${RED}✗ Port $SMTP_PORT is NOT reachable from host${NC}"
        fi
    else
        echo "Skipping host test (nc/telnet not available)"
    fi
fi

echo ""

# Test 4: Test SMTP handshake
echo -e "${YELLOW}=== Test 4: SMTP Handshake Test ===${NC}"
echo ""

if [ "$SMTP_HOST" != "NOT SET" ] && [ "$SMTP_PORT" != "NOT SET" ]; then
    echo "Testing SMTP handshake..."
    
    RESULT=$(docker exec hng-email-service python3 << 'PYTHON'
import socket
import sys
import os

smtp_host = os.getenv('SMTP_HOST', 'smtp.gmail.com')
smtp_port = int(os.getenv('SMTP_PORT', '587'))

try:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(10)
    sock.connect((smtp_host, smtp_port))
    
    # Read SMTP greeting
    response = sock.recv(1024).decode('utf-8', errors='ignore')
    print(f"SMTP Server Response: {response.strip()}")
    
    # Send EHLO
    sock.send(b'EHLO test\r\n')
    response = sock.recv(1024).decode('utf-8', errors='ignore')
    print(f"EHLO Response: {response.strip()[:100]}")
    
    sock.close()
    print("SUCCESS")
    sys.exit(0)
except socket.timeout:
    print("ERROR: Connection timeout")
    sys.exit(1)
except Exception as e:
    print(f"ERROR: {e}")
    sys.exit(1)
PYTHON
2>&1)
    
    if echo "$RESULT" | grep -q "SUCCESS"; then
        echo -e "${GREEN}✓ SMTP handshake successful${NC}"
        echo "$RESULT" | grep -E "SMTP Server|EHLO"
    else
        echo -e "${RED}✗ SMTP handshake failed${NC}"
        echo "$RESULT"
    fi
fi

echo ""

# Test 5: Test full SMTP authentication (if credentials are set)
echo -e "${YELLOW}=== Test 5: SMTP Authentication Test ===${NC}"
echo ""

if [ "$SMTP_USER" != "NOT SET" ] && [ "$SMTP_PASS" != "NOT SET" ]; then
    echo "Testing SMTP authentication..."
    echo "Note: This will attempt to authenticate (may fail if credentials are incorrect)"
    
    RESULT=$(docker exec hng-email-service python3 << 'PYTHON'
import os
import sys
from smtplib import SMTP

smtp_host = os.getenv('SMTP_HOST', 'smtp.gmail.com')
smtp_port = int(os.getenv('SMTP_PORT', '587'))
smtp_user = os.getenv('SMTP_USER')
smtp_pass = os.getenv('SMTP_PASS')

try:
    import socket
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(10)
    sock.connect((smtp_host, smtp_port))
    
    server = SMTP()
    server.sock = sock
    server.starttls()
    
    server.login(smtp_user, smtp_pass)
    print("SUCCESS: Authentication successful")
    server.quit()
    sys.exit(0)
except Exception as e:
    print(f"ERROR: {e}")
    sys.exit(1)
PYTHON
2>&1)
    
    if echo "$RESULT" | grep -q "SUCCESS"; then
        echo -e "${GREEN}✓ SMTP authentication successful${NC}"
    else
        echo -e "${RED}✗ SMTP authentication failed${NC}"
        echo "$RESULT"
    fi
else
    echo "Skipping authentication test (credentials not set)"
fi

echo ""
echo "=========================================="
echo "TEST COMPLETE"
echo "=========================================="
echo ""
echo "Common Issues:"
echo "1. Port 587 blocked by firewall/security group"
echo "2. Docker network configuration"
echo "3. ISP blocking SMTP ports"
echo "4. EC2 security group needs outbound rule for port 587"
echo ""
echo "Alternative SMTP Options:"
echo "- Port 465 (SSL/TLS)"
echo "- Port 25 (may be blocked)"
echo "- Use different SMTP provider (SendGrid, Mailgun, etc.)"

