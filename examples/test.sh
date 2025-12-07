#!/bin/bash

# Example script to test the DynDNS Cloudflare Proxy
# This script demonstrates how to use the service

# Configuration
SERVER="http://localhost:8080"
USERNAME="your_username"
PASSWORD="your_password"
HOSTNAME="home.example.com"
IP="1.2.3.4"

echo "Testing DynDNS Cloudflare Proxy"
echo "================================"
echo ""

# Test 1: Health check
echo "1. Testing health endpoint..."
curl -s "$SERVER/health"
echo -e "\n"

# Test 2: Update with specific IP
echo "2. Testing update with specific IP..."
curl -s -u "$USERNAME:$PASSWORD" "$SERVER/nic/update?hostname=$HOSTNAME&myip=$IP"
echo -e "\n"

# Test 3: Update with auto-detected IP
echo "3. Testing update with auto-detected IP..."
curl -s -u "$USERNAME:$PASSWORD" "$SERVER/nic/update?hostname=$HOSTNAME"
echo -e "\n"

# Test 4: Test without authentication (should fail if auth is enabled)
echo "4. Testing without authentication (should fail if auth is enabled)..."
curl -s "$SERVER/nic/update?hostname=$HOSTNAME&myip=$IP"
echo -e "\n"

echo "Testing complete!"
