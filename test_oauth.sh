#!/bin/bash

# Test script for GitHub OAuth implementation

echo "=== Testing MCP Server with GitHub OAuth ==="
echo ""

# Test 1: Health check (no auth required)
echo "Test 1: Health check endpoint (should work without auth)"
curl -s http://localhost:8080/health
echo -e "\n"

# Test 2: MCP endpoint without auth (should fail)
echo "Test 2: MCP endpoint without authentication (should fail with 401)"
curl -s -w "\nHTTP Status: %{http_code}\n" http://localhost:8080/mcp
echo ""

# Test 3: MCP endpoint with invalid token (should fail)
echo "Test 3: MCP endpoint with invalid token (should fail with 401)"
curl -s -w "\nHTTP Status: %{http_code}\n" \
  -H "Authorization: Bearer invalid_token_here" \
  http://localhost:8080/mcp
echo ""

# Test 4: Instructions for OAuth flow
echo "Test 4: To get a valid token, visit:"
echo "http://localhost:8080/oauth/login"
echo ""
echo "After authenticating with GitHub, copy the bearer token and test with:"
echo 'export TOKEN="your_token_here"'
echo 'curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/mcp'
echo ""

