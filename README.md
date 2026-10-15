# Job Queue

A simple job queue that shows how a job comes in through an HTTP
request, gets placed into a Redis queue, and is then executed by workers.

## Architecture

```
                         +---------------------------+
                         |          CLIENT           |
                         +-------------+-------------+
                                       |
                                       |  POST /enqueue
                                       |  { type, payload, retries }
                                       v
                         +---------------------------+
                         |         PRODUCER          |
                         |    validate the task      |
                         +-------------+-------------+
                                       |
                                       |  RPUSH  (add a job)
                                       v
                         +---------------------------+
                         |           REDIS           |
                         |      list: task_queue     |
                         |         FIFO queue        |
                         +-------------+-------------+
                                       |
                                       |  BLPOP  (blocking pop, one job)
                                       v
                         +---------------------------+
                         |          WORKER           |
                         |       3 goroutines        |
                         |      Process_Task()       |
                         +-------------+-------------+
                                       |
                               +-------+-------+
                               |               |
                               v               v
                      +-------------+   +---------------+
                      |  jobs_done  |   |  jobs_failed  |
                      +------+------+   +-------+-------+
                             |                  |
                             v                  v
                        LogSuccess         LogFailure
                                                |
                                                v
                                          retries left?
                                           /        \
                                        yes          no
                                         |            |
                                         v            v
                                  RPUSH back      drop job
                                  to the queue   (dead-letter)
                                         \
                                          `--> back to REDIS

   Logs are appended to  log.txt   (internal/logger/logger.go)

   GET /metrics  ->  { total_jobs_in_queue, jobs_done, jobs_failed }
```

## How it works (step by step)

1. A **client** sends `POST /enqueue` with a task as JSON: `type`, `payload`, `retries`.
2. The **producer** (`cmd/producer/main.go`) validates the task and `RPUSH`es it onto the Redis list `task_queue`.
3. The **worker** (`cmd/worker/main.go`) runs 3 goroutines. Each one waits with `BLPOP`, which blocks until a job is available and hands each job to exactly one worker.
4. Each job is handled by `Process_Task` (`internal/worker/worker.go`), which runs the right action for the task `type`.
5. **On success**, `jobs_done` is increased and `LogSuccess` appends a line to `log.txt`.
6. **On failure**, `jobs_failed` is increased and `LogFailure` appends a line to `log.txt`:
   - if retries are left, the task is put back on the queue with `RPUSH`;
   - if no retries are left, the task is dropped (dead-letter).
7. Anyone can `GET /metrics` on the worker to read the current counters.

## HTTP endpoints

1. **POST /enqueue** — in `cmd/producer/main.go`
   Validates the incoming task and `RPUSH`es it onto the Redis `task_queue`.

2. **GET /metrics** — in `cmd/worker/main.go`
   Returns the worker's counters (`jobs_done` / `jobs_failed` / `total_jobs_in_queue`).

## Running locally

Two processes must run at the same time: the **producer** (HTTP API) and the
**worker** (queue consumer).

### Prerequisites

- **Go 1.27+** — check with `go version`
- **Redis 7+** — either installed locally or run with Docker

### 1. Start Redis

```sh
# Option A: local install
redis-server --daemonize yes

# Option B: Docker
docker run -d --name job-queue-redis -p 6379:6379 redis:7-alpine
```

Confirm it is up:

```sh
redis-cli ping   # -> PONG
```

### 2. Create a `.env` file

Create `.env` in the **repository root**:

```
REDIS_URL=redis://localhost:6379/0
PORT_PRODUCER=8080
PORT_WORKER=8081
```

`.env` is gitignored, so your connection string is never committed.

### 3. Start the worker (terminal 1)

```sh
go run ./cmd/worker
```

### 4. Start the producer (terminal 2)

```sh
go run ./cmd/producer
# -> Starting the server on port  :8080
```

> Both commands must run from the **repository root**. The programs load
> `.env` from the current working directory, so running from inside
> `cmd/producer/` or `cmd/worker/` will not find it.

### 5. Send a job

```sh
curl -X POST localhost:8080/enqueue \
  -H 'Content-Type: application/json' \
  -d '{"type":"send_email","payload":{"to":"a@b.com","subject":"hello"},"retries":3}'
```

Response:

```
Task of type 'send_email' has been successfully added to the queue
```

### 6. Check metrics and logs

```sh
curl localhost:8081/metrics
# -> {"total_jobs_in_queue":0,"jobs_done":1,"jobs_failed":0}

tail log.txt
```

### Supported task types

Only the types handled in `internal/worker/worker.go` do real work:

| Type | Behaviour |
| --- | --- |
| `send_email` | sleeps 2 s, then succeeds |
| `resize_image` | succeeds immediately |
| `generate_pdf` | succeeds immediately |
| anything else | fails and is retried until `retries` is exhausted |

### Trying the retry path

Send an unsupported type and watch it fail, requeue, then dead-letter:

```sh
curl -X POST localhost:8080/enqueue \
  -H 'Content-Type: application/json' \
  -d '{"type":"foobar","payload":{"x":1},"retries":2}'

sleep 3 && curl localhost:8081/metrics
# -> {"total_jobs_in_queue":0,"jobs_done":1,"jobs_failed":2}
```

With `retries: 2` the task fails twice: once, is put back on the queue, fails
again, then is dropped (dead-letter).

### Notes

- `jobs_done` / `jobs_failed` are **in-memory** counters. They reset every time
  the worker restarts, and are not shared if you run more than one worker.
- `log.txt` is written to the **current working directory**, so running from the
  repository root puts it at `./log.txt`.
- To clear the Redis queue between runs: `redis-cli DEL task_queue`


