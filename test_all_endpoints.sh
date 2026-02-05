#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8080"
ACCOUNT_ID=""
ISIN="US0378331005" # Apple stock ISIN

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Valhafin Backend API Test Suite${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to test endpoint
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    echo -e "${YELLOW}Testing:${NC} $description"
    echo -e "${BLUE}$method $endpoint${NC}"
    
    if [ -z "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X $method "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X $method "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        echo -e "${GREEN}✓ Success (HTTP $http_code)${NC}"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ Failed (HTTP $http_code)${NC}"
        echo "$body"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    echo ""
}

# 1. Health Check
echo -e "${BLUE}=== 1. Health Check ===${NC}\n"
test_endpoint "GET" "/health" "" "Health check endpoint"

# 2. Account Management
echo -e "${BLUE}=== 2. Account Management ===${NC}\n"

# Create account
test_endpoint "POST" "/api/accounts" '{
  "name": "Test Trade Republic Account",
  "platform": "traderepublic",
  "credentials": {
    "phone_number": "+33612345678",
    "pin": "1234"
  }
}' "Create Trade Republic account"

# Get all accounts
response=$(curl -s "$BASE_URL/api/accounts")
ACCOUNT_ID=$(echo "$response" | jq -r '.[0].id' 2>/dev/null)
test_endpoint "GET" "/api/accounts" "" "List all accounts"

if [ -n "$ACCOUNT_ID" ] && [ "$ACCOUNT_ID" != "null" ]; then
    # Get specific account
    test_endpoint "GET" "/api/accounts/$ACCOUNT_ID" "" "Get account details"
    
    # 3. Synchronization
    echo -e "${BLUE}=== 3. Synchronization ===${NC}\n"
    test_endpoint "POST" "/api/accounts/$ACCOUNT_ID/sync" "" "Sync account (may fail without real credentials)"
    
    # 4. Transactions
    echo -e "${BLUE}=== 4. Transactions ===${NC}\n"
    test_endpoint "GET" "/api/accounts/$ACCOUNT_ID/transactions" "" "Get account transactions"
    test_endpoint "GET" "/api/accounts/$ACCOUNT_ID/transactions?page=1&limit=10" "" "Get transactions with pagination"
    test_endpoint "GET" "/api/accounts/$ACCOUNT_ID/transactions?type=buy" "" "Filter transactions by type"
    
    # 5. Performance
    echo -e "${BLUE}=== 5. Performance ===${NC}\n"
    test_endpoint "GET" "/api/accounts/$ACCOUNT_ID/performance" "" "Get account performance"
    test_endpoint "GET" "/api/accounts/$ACCOUNT_ID/performance?period=1m" "" "Get account performance (1 month)"
    test_endpoint "GET" "/api/accounts/$ACCOUNT_ID/performance?period=3m" "" "Get account performance (3 months)"
    
    # 6. Fees
    echo -e "${BLUE}=== 6. Fees ===${NC}\n"
    test_endpoint "GET" "/api/accounts/$ACCOUNT_ID/fees" "" "Get account fees"
    test_endpoint "GET" "/api/accounts/$ACCOUNT_ID/fees?period=1m" "" "Get account fees (1 month)"
fi

# 7. Global endpoints
echo -e "${BLUE}=== 7. Global Endpoints ===${NC}\n"
test_endpoint "GET" "/api/transactions" "" "Get all transactions"
test_endpoint "GET" "/api/performance" "" "Get global performance"
test_endpoint "GET" "/api/fees" "" "Get global fees"

# 8. Asset prices
echo -e "${BLUE}=== 8. Asset Prices ===${NC}\n"
test_endpoint "GET" "/api/assets/$ISIN/price" "" "Get current asset price"
test_endpoint "GET" "/api/assets/$ISIN/history" "" "Get asset price history"
test_endpoint "GET" "/api/assets/$ISIN/performance" "" "Get asset performance"

# 9. CSV Import
echo -e "${BLUE}=== 9. CSV Import ===${NC}\n"
if [ -n "$ACCOUNT_ID" ] && [ "$ACCOUNT_ID" != "null" ]; then
    # Create a test CSV file
    cat > /tmp/test_transactions.csv << EOF
timestamp,isin,quantity,amount_value,fees,type
2024-01-15T10:00:00Z,US0378331005,10,1500.00,5.00,buy
2024-02-20T14:30:00Z,US0378331005,5,800.00,3.00,sell
EOF
    
    echo -e "${YELLOW}Testing:${NC} Import CSV transactions"
    echo -e "${BLUE}POST /api/transactions/import${NC}"
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/transactions/import" \
        -F "file=@/tmp/test_transactions.csv" \
        -F "account_id=$ACCOUNT_ID")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        echo -e "${GREEN}✓ Success (HTTP $http_code)${NC}"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ Failed (HTTP $http_code)${NC}"
        echo "$body"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    echo ""
    
    rm /tmp/test_transactions.csv
fi

# 10. Delete account (cleanup)
if [ -n "$ACCOUNT_ID" ] && [ "$ACCOUNT_ID" != "null" ]; then
    echo -e "${BLUE}=== 10. Cleanup ===${NC}\n"
    test_endpoint "DELETE" "/api/accounts/$ACCOUNT_ID" "" "Delete test account"
fi

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo -e "${BLUE}Total:  $((TESTS_PASSED + TESTS_FAILED))${NC}\n"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi
