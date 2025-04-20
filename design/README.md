## Design Doc:
First major factor in design is adressed to APIs structure: GET and PUT both are idempotent methods, so we don't require any state locking in our application code, to continue with code decisigions, library `log/slog` where picked for structured logging and ease of troubleshooting, for HTTP server `net/http` standard library was picked as it's covers all reqirements.

Postgres database was picked because it's always a good practice to start with relational database, and only later migrate to NoSQL.

In GCP CloudRun and CloudSQL where picked as they are manged service and they are higly scalable. Cloud SQL deployed with private IP which could help with secutiry. CloudRun is pay as you go service. And both of them low maintenance.

Why no Cloud Functions? Main reson is that CloudRun containers could be easily ported to on prem in Kubernetes, CloudRun has features such as readiness and startup probes, advanced storage(volume) and networking integrations. 
