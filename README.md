
## Raft3D - Distributed 3D Printer Management System

This project implements a distributed 3D printer management system using the Raft consensus algorithm.
Nodes communicate via HTTP APIs and maintain a consistent replicated state across the cluster.

##  Installation

Install project dependencies:

```bash
go mod download
go mod tidy
```

##  Running Nodes

Start a node:

```bash
go run . --node_id=<NODE_ID> --raft_addr=127.0.0.1:900X --data_dir=<DATA_DIR> --http_port=<HTTP_PORT>
```

Note: Use the `--bootstrap` flag to create a new cluster with the first node.

### Example Commands

Start a new leader node:

```bash
go run . --bootstrap --node_id=node1 --raft_addr=127.0.0.1:9001 --data_dir=node1_data --http_port=8081
```

Start additional follower nodes:

```bash
go run . --node_id=node2 --raft_addr=127.0.0.1:9002 --data_dir=node2_data --http_port=8082
```

```bash
go run . --node_id=node3 --raft_addr=127.0.0.1:9003 --data_dir=node3_data --http_port=8083
```

## Joining Nodes to the Cluster

Join a node to the existing cluster by making a POST request to the leader node:

```bash
# Join node2 to the cluster
curl -X POST "http://localhost:8081/join?node_id=node2&addr=127.0.0.1:9002"

# Join node3 to the cluster
curl -X POST "http://localhost:8081/join?node_id=node3&addr=127.0.0.1:9003"
```

## Checking Node Status

Check if a node is a Leader or Follower:

```bash
# Check node1 status
curl "http://localhost:8081/raft/state"

# Check node2 status
curl "http://localhost:8082/raft/state"

# Check node3 status
curl "http://localhost:8083/raft/state"
```

A leader node will return "Leader".

Follower nodes will return "Follower".

##  REST API Endpoints

**Important:** Always send API requests to the Leader node.

### Printers

**Add a printer:**

```bash
curl -X POST http://localhost:8081/api/v1/printers -H "Content-Type: application/json" -d '{"id":"printer1","company":"HP","model":"L12"}'
```

**Get all printers:**

```bash
curl -X GET http://localhost:8081/api/v1/printers
```

### Filaments

**Add a filament:**

```bash
curl -X POST http://localhost:8081/api/v1/filaments -H "Content-Type: application/json" -d '{"id":"f1","type":"PVC","color":"Black","total_weight_in_grams":100,"remaining_weight_in_grams":100}'
```

**Get all filaments:**

```bash
curl -X GET http://localhost:8081/api/v1/filaments
```

### Print Jobs

**Create a new print job:**

```bash
curl -X POST http://localhost:8081/api/v1/print_jobs -H "Content-Type: application/json" -d '{"job_id":"job1","printer_id":"printer1","filament_id":"f1","print_weight_in_grams":50,"status":"Queued","details":"mountain"}'
```

**Get all print jobs:**

```bash
curl -X GET http://localhost:8081/api/v1/print_jobs
```

### Updating Print Job Status

Allowed state transitions:

- Queued → Running → Done
- Queued → Cancelled
- Queued → Running → Cancelled

**Example - Update job1 status to Running:**

```bash
curl -X PUT "http://localhost:8081/api/v1/print_jobs/job1/status?status=Running"
```

##  Restarting a Node

To restart a previously stopped node:

```bash
go run . --node_id=node1 --raft_addr=127.0.0.1:9001 --data_dir=node1_data --http_port=8081
```

If the cluster is already running, the restarted node will rejoin as a Follower automatically.
