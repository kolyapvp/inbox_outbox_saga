#!/bin/bash

# Definition of colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

PIDS_FILE=".project.pids"

echo -e "${BLUE}=== Starting Project (Backend + Frontend) ===${NC}"

# 1. Build Go Binaries
echo -e "${BLUE}[1/5] Building Go Services...${NC}"
mkdir -p bin logs
go build -o bin/api cmd/api/main.go || { echo -e "${RED}Failed to build API${NC}"; exit 1; }
go build -o bin/worker cmd/worker/main.go || { echo -e "${RED}Failed to build Worker${NC}"; exit 1; }
go build -o bin/consumer cmd/consumer/main.go || { echo -e "${RED}Failed to build Order Consumer${NC}"; exit 1; }
go build -o bin/payment cmd/payment/main.go || { echo -e "${RED}Failed to build Payment Service${NC}"; exit 1; }
go build -o bin/ticket cmd/ticket/main.go || { echo -e "${RED}Failed to build Ticket Service${NC}"; exit 1; }

# 2. Start Infrastructure
echo -e "${BLUE}[2/5] Starting Infrastructure (Docker)...${NC}"
# Start all, then stop the ones we replace locally to avoid port conflicts
docker-compose -p web_app up -d
docker-compose -p web_app stop api worker consumer payment ticket simulator

# Apply DB migrations for existing volumes (docker-entrypoint-initdb.d runs only on fresh volumes)
echo -e "${BLUE}Applying migrations (saga tables/columns)...${NC}"
for i in {1..30}; do
    if docker-compose -p web_app exec -T postgres pg_isready -U user -d wb_tech >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

docker-compose -p web_app exec -T postgres psql -U user -d wb_tech -f /docker-entrypoint-initdb.d/005_saga_choreography.sql >/dev/null || {
    echo -e "${RED}Failed to apply migrations (005_saga_choreography.sql)${NC}";
    exit 1;
}

# Force kill any process on port 8080 to avoid "address already in use" errors
if lsof -ti :8080 >/dev/null; then
    echo -e "${RED}Port 8080 is in use, forcing cleanup...${NC}"
    lsof -ti :8080 | xargs kill -9
    sleep 1
fi


# 3. Start Backend Services
echo -e "${BLUE}[3/5] Starting Backend Services...${NC}"

# Override Config for Local Host Execution
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5433
export REDIS_ADDR=localhost:6379
export KAFKA_BROKERS=127.0.0.1:9092
export KAFKA_TOPIC=orders-events
export KAFKA_START_OFFSET=latest

# API
nohup ./bin/api > logs/api.log 2>&1 &
API_PID=$!
echo $API_PID >> $PIDS_FILE
echo -e "${GREEN}API started (PID: $API_PID)${NC}"

# Worker
nohup ./bin/worker > logs/worker.log 2>&1 &
WORKER_PID=$!
echo $WORKER_PID >> $PIDS_FILE
echo -e "${GREEN}Worker started (PID: $WORKER_PID)${NC}"

# Consumer
nohup env KAFKA_GROUP_ID=order-service ./bin/consumer > logs/consumer.log 2>&1 &
CONSUMER_PID=$!
echo $CONSUMER_PID >> $PIDS_FILE
echo -e "${GREEN}Consumer started (PID: $CONSUMER_PID)${NC}"

# Payment Service
nohup env KAFKA_GROUP_ID=payment-service ./bin/payment > logs/payment.log 2>&1 &
PAYMENT_PID=$!
echo $PAYMENT_PID >> $PIDS_FILE
echo -e "${GREEN}Payment Service started (PID: $PAYMENT_PID)${NC}"

# Ticket Service
nohup env KAFKA_GROUP_ID=ticket-service ./bin/ticket > logs/ticket.log 2>&1 &
TICKET_PID=$!
echo $TICKET_PID >> $PIDS_FILE
echo -e "${GREEN}Ticket Service started (PID: $TICKET_PID)${NC}"

# 4. Start Frontend
echo -e "${BLUE}[4/5] Starting Frontend...${NC}"
cd frontend
nohup npm run dev > ../logs/frontend.log 2>&1 &
FRONTEND_PID=$!
cd ..
echo $FRONTEND_PID >> $PIDS_FILE
echo -e "${GREEN}Frontend started (PID: $FRONTEND_PID)${NC}"

# 5. Summary
echo -e "${BLUE}=== System Ready ===${NC}"
echo -e "Frontend: ${GREEN}http://localhost:5173${NC}"
echo -e "API:      ${GREEN}http://localhost:8080${NC}"
echo -e "Grafana:  ${GREEN}http://localhost:3000${NC}"
echo -e "Logs available in: logs/ directory"
echo -e "Run 'make down' to stop everything."
