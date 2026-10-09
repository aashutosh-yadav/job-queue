# Job Queue

A simple job queue project that shows how a job comes in through an HTTP
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

   Logs are appended to  logs.txt   (internal/logger/logger.go)

   GET /metrics  ->  { total_jobs_in_queue, jobs_done, jobs_failed }
```

## How it works (step by step)

1. A **client** sends `POST /enqueue` with a task as JSON: `type`, `payload`, `retries`.
2. The **producer** (`cmd/producer/main.go`) validates the task and `RPUSH`es it onto the Redis list `task_queue`.
3. The **worker** (`cmd/worker/main.go`) runs 3 goroutines. Each one waits with `BLPOP`, which blocks until a job is available and hands each job to exactly one worker.
4. Each job is handled by `Process_Task` (`internal/worker/worker.go`), which runs the right action for the task `type`.
5. **On success**, `jobs_done` is increased and `LogSuccess` appends a line to `logs.txt`.
6. **On failure**, `jobs_failed` is increased and `LogFailure` appends a line to `logs.txt`:
   - if retries are left, the task is put back on the queue with `RPUSH`;
   - if no retries are left, the task is dropped (dead-letter).
7. Anyone can `GET /metrics` on the worker to read the current counters.

## HTTP endpoints

1. **POST /enqueue** — in `cmd/producer/main.go`
   Validates the incoming task and `RPUSH`es it onto the Redis `task_queue`.

2. **GET /metrics** — in `cmd/worker/main.go`
   Returns the worker's counters (`jobs_done` / `jobs_failed` / `total_jobs_in_queue`).

