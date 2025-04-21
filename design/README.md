## Design Doc:

### Code:
In APIs structure: GET and PUT both are idempotent methods, so we don't require any state locking in our application code. Ad programming language was chosen Golang, it's clear, performant and efficient. Notable changes:
* library `log/slog` where picked for structured logging and ease of troubleshooting
* standard `net/http` library was picked for router, as don't introduce additional dependencies, and lowers securiy risks
* Instrumentation with prometheus `metrics` for better alerting and readiness and liveness probes in CloudRun

#### Key Features of the Implementation

1. **Prometheus Readiness Metrics**:
    - `app_ready` gauge metric (1 when ready, 0 when not ready)
    - `db_connections_active` to track active database connections
    - Performance metrics for request counts and response times
2. **Database Connectivity Monitoring**:
    - A background goroutine checks the database connection every 10 seconds
    - If connection is established, `app_ready` is set to 1; otherwise, it's set to 0
3. **Health and Readiness Endpoints**:
    - `/health`: Simple liveness check that returns 200 OK if the server is running
    - `/readiness`: Checks database connectivity and returns 200 OK only if ready
    - `/metrics`: Exposes all Prometheus metrics
4. **Request Handling**:
    - All API endpoints check database readiness before processing
    - If the database isn't ready, endpoints return 503 Service Unavailable

### Database
Postgres database was picked because it's always a good practice to start with relational database, as we keep all the features of ACID for free. GCP managed CloudSQL service supports Postgres latest versions.

### Google Cloud Platform
In GCP, CloudRun and CloudSQL where picked as they are manged service and they are higly scalable. Cloud SQL deployed with private IP which could help with secutiry. CloudRun is pay as you go service. And both of them low maintenance.

### Q&A
Why no Cloud Functions? Main reson is that CloudRun containers could be easily ported to on prem in Kubernetes, CloudRun has features such as readiness and startup probes, advanced storage(volume) and networking integrations. 

### Disadvantages
One the main disadvantages as of not it's a manual step of Database Migration between deployment of CloudSQL and CloudRun revision. As a solution to this additonal CloudRun service could be developed which could expose `/migrations/<migration>` endpoint and been executed by terraoform `local-exec`.

Second option could be deployment of terraform code from the VM in the private network, but this would require to create network in advance, and provision all the nessesary IAM roles for VM service account. 
