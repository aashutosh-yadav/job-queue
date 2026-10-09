The project has two HTTP endpoints, one per service:

1.POST /enqueue — in cmd/producer/main.go
Validates the incoming task and RPUSHes it onto the Redis task_queue.

2.GET /metrics — in cmd/worker/main.go
Returns the worker's counters (jobs done / jobs failed).his is a simple job queue project where it shows how a jobs is taken from  the post request and how it is put in the redis queue and how it it gets executed.

The detail architecture of the project is in the architecture.md file .
