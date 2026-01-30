# 14-Day Study Plan

Based on the interview analysis, focus on: **Kubernetes Deep Dive**, **System Design (Secret Management)**, and **Go Runtime Internals in K8s**.

## Week 1: Kubernetes & Infrastructure (The "Gaps")

### Day 1-2: Kubernetes Config & Secrets
- **Read**: K8s docs on ConfigMaps, Secrets (Immutable Secrets).
- **Practice**: Deploy a simple Go app. Mount credentials via Secret (env var) and Config (volume).
- **Deep Dive**: How does `ExternalSecrets` or `Vault` work? (Just concept).
- **Goal**: Be able to explain why storing secrets in git is bad and how to fix it (Helm Secrets / SOPS).

### Day 3-4: Kubernetes Networking & Ingress
- **Read**: Ingress vs Service vs Gateway API. Nginx Ingress Controller configuration (`client_max_body_size`, timeout).
- **Practice**: Configure Nginx Ingress regarding the "413 Payload Too Large" error locally with Kind or Minikube.

### Day 5-7: Observability (Prometheus/Grafana)
- **Read**: Prometheus metric types (Counter, Gauge, Histogram). RED Method.
- **Practice**: Add metrics to the Go app. Create a dashboard showing RPS and Error Rate.

## Week 2: Go Internals & System Design

### Day 8-9: Go Runtime in Containers
- **Read**: "Go in Containers" (Ardan Labs).
- **Topic**: `GOMAXPROCS`, CFS Quota, Throttling.
- **Practice**: Run a benchmark with generic GOMAXPROCS vs `automaxprocs` under CPU limits.

### Day 10-11: Async & Kafka
- **Read**: Kafka guarantees (at-least-once, exactly-once). Consumer Groups.
- **Practice**: Implement Graceful Shutdown for Kafka Consumer in the project.

### Day 12-14: Mock Interview & Review
- **Activity**: Re-watch the interview video.
- **Test**: Answer the "failed" questions aloud. Record yourself.
- **Final**: Complete the README and Code of this practice project.
