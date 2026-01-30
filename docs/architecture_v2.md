# Architecture V2: gRPC & Outbox

## 1. Project Overview (Current State)
The current project is a **Modular Monolith** in Go, designed for high load (WB/PVZ context).
- **Core**: Go 1.21, Clean Architecture (simplified).
- **Storage**: PostgreSQL (pgxpool).
- **Messaging**: Kafka (producer/consumer).
- **Cache**: Redis.
- **Infrastructure**: Docker Compose, Kubernetes manifests (Deployment, HPA).
- **Observability**: Prometheus metrics.

### Key Decisions
- **Factory Pattern**: strict dependency injection via `internal/application/factories`.
- **Infrastructure Layer**: Isolated in `internal/infrastructure`. Domains do not import infrastructure directly.

## 2. New Requirements via V2

### A. gRPC Service
We will add a gRPC interface to the API service.
- **Protocol**: Protocol Buffers v3.
- **Role**: High-performance inter-service communication (e.g., typically used for internal traffic vs REST for external).

### B. Transactional Outbox Pattern
To guarantee **simultaneous** database update and event publishing.

#### The Workflow
1. **API Request**: User creates an order (`POST /orders`).
2. **Transaction Start**: Start Postgres transaction.
3. **Data Write**: Insert into `orders` table.
4. **Outbox Write**: Insert event into `outbox` table (payload: JSON).
5. **Commit**: Commit transaction. (If fails, neither happens).
6. **Async Worker**:
   - Loop pulls unprocessed events from `outbox`.
   - Publishes to Kafka `orders-events` topic.
   - Marks entry as `processed` (or deletes it).

## 3. Diagrams

### Outbox Flow
```mermaid
sequenceDiagram
    participant Client
    participant API
    participant DB as PostgreSQL (TX)
    participant Worker
    participant Kafka

    Client->>API: POST /orders
    API->>DB: BEGIN TX
    API->>DB: INSERT INTO orders
    API->>DB: INSERT INTO outbox (event_type="OrderCreated")
    API->>DB: COMMIT TX
    API-->>Client: 200 OK

    loop Async Poller
        Worker->>DB: SELECT * FROM outbox WHERE status='new' LIMIT 50
        Worker->>Kafka: Publish "OrderCreated"
        alt Publish Success
            Worker->>DB: UPDATE outbox SET status='processed'
        else Publish Fail
            Worker->>DB: Retry later (Backoff)
        end
    end
```

### Component View
```mermaid
graph TD
    subgraph "API Pod"
        H[HTTP Handler]
        G[gRPC Server]
        U[UseCase]
    end

    subgraph "Worker Pod"
        P[Outbox Poller]
    end

    subgraph "Infrastructure"
        DB[(PostgreSQL)]
        K{Kafka}
    end

    H --> U
    G --> U
    U -->|TxManager| DB
    P -->|Read New| DB
    P -->|Publish| K
```
