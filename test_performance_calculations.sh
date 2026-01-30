#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8080"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Performance Calculation Test${NC}"
echo -e "${BLUE}========================================${NC}\n"

# 1. Create a test account
echo -e "${YELLOW}Step 1: Creating test account...${NC}"
response=$(curl -s -X POST "$BASE_URL/api/accounts" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "Performance Test Account",
      "platform": "traderepublic",
      "credentials": {
        "phone_number": "+33612345678",
        "pin": "1234"
      }
    }')

ACCOUNT_ID=$(echo "$response" | jq -r '.id')
echo -e "${GREEN}✓ Account created: $ACCOUNT_ID${NC}\n"

# 2. Import test transactions with known values
echo -e "${YELLOW}Step 2: Importing test transactions...${NC}"
cat > /tmp/perf_test.csv << EOF
timestamp,isin,quantity,amount_value,fees,type,title
2024-01-15T10:00:00Z,US0378331005,10,1500.00,5.00,buy,Buy Apple
2024-02-20T14:30:00Z,US0378331005,5,800.00,3.00,sell,Sell Apple
2024-03-10T09:00:00Z,FR0000120271,20,2000.00,10.00,buy,Buy Total
EOF

response=$(curl -s -X POST "$BASE_URL/api/transactions/import" \
    -F "file=@/tmp/perf_test.csv" \
    -F "account_id=$ACCOUNT_ID")

echo "$response" | jq '.'
echo ""

# 3. Get transactions to verify import
echo -e "${YELLOW}Step 3: Verifying imported transactions...${NC}"
response=$(curl -s "$BASE_URL/api/accounts/$ACCOUNT_ID/transactions")
echo "$response" | jq '.transactions[] | {timestamp, isin, quantity, amount_value, fees, type}'
echo ""

# 4. Get current prices for assets
echo -e "${YELLOW}Step 4: Getting current asset prices...${NC}"
echo "Apple (US0378331005):"
curl -s "$BASE_URL/api/assets/US0378331005/price" | jq '{isin, price, currency}'
echo ""
echo "Total (FR0000120271):"
curl -s "$BASE_URL/api/assets/FR0000120271/price" | jq '{isin, price, currency}'
echo ""

# 5. Calculate performance
echo -e "${YELLOW}Step 5: Calculating account performance...${NC}"
response=$(curl -s "$BASE_URL/api/accounts/$ACCOUNT_ID/performance")
echo "$response" | jq '.'
echo ""

# 6. Verify calculations
echo -e "${YELLOW}Step 6: Verifying calculations...${NC}"
total_invested=$(echo "$response" | jq -r '.total_invested')
total_fees=$(echo "$response" | jq -r '.total_fees')
total_value=$(echo "$response" | jq -r '.total_value')
performance_pct=$(echo "$response" | jq -r '.performance_pct')

echo -e "Total Invested: ${BLUE}$total_invested${NC}"
echo -e "Total Fees: ${BLUE}$total_fees${NC}"
echo -e "Total Value: ${BLUE}$total_value${NC}"
echo -e "Performance: ${BLUE}$performance_pct%${NC}"
echo ""

# Expected calculations:
# Buy 10 Apple @ 1500 = 1500 invested, 5 fees
# Sell 5 Apple @ 800 = -800 invested (realized), 3 fees
# Buy 20 Total @ 2000 = 2000 invested, 10 fees
# Net invested = 1500 - 800 + 2000 = 2700
# Total fees = 5 + 3 + 10 = 18
# Current holdings: 5 Apple + 20 Total
# Current value = 5 * current_apple_price + 20 * current_total_price

echo -e "${YELLOW}Expected calculations:${NC}"
echo "Net invested: 2700 (1500 - 800 + 2000)"
echo "Total fees: 18 (5 + 3 + 10)"
echo "Current holdings: 5 Apple + 20 Total"
echo ""

# 7. Get fees metrics
echo -e "${YELLOW}Step 7: Getting fees metrics...${NC}"
response=$(curl -s "$BASE_URL/api/accounts/$ACCOUNT_ID/fees")
echo "$response" | jq '.'
echo ""

# 8. Get asset-specific performance
echo -e "${YELLOW}Step 8: Getting asset-specific performance...${NC}"
echo "Apple performance:"
curl -s "$BASE_URL/api/assets/US0378331005/performance" | jq '{isin, name, current_price, total_quantity, total_value, performance_pct}'
echo ""

# 9. Cleanup
echo -e "${YELLOW}Step 9: Cleaning up...${NC}"
curl -s -X DELETE "$BASE_URL/api/accounts/$ACCOUNT_ID" | jq '.'
rm /tmp/perf_test.csv
echo ""

echo -e "${GREEN}✓ Performance calculation test completed!${NC}"
