go mod download    

go mod tidy

go run . --bootstrap --node_id=leader --raft_addr=127.0.0.1:9001 --data_dir=leader_data --http_port=8081

go run . --node_id=node2 --raft_addr=127.0.0.1:9002 --data_dir=node2_data --http_port=8082

go run . --node_id=node3 --raft_addr=127.0.0.1:9003 --data_dir=node3_data --http_port=8083

curl -X POST "http://localhost:8081/join?node_id=node2&addr=127.0.0.1:9002"

curl -X POST "http://localhost:8081/join?node_id=node3&addr=127.0.0.1:9003"

curl "http://localhost:8081/raft/state"  # Should return "Leader"
curl "http://localhost:8082/raft/state"  # Should return "Follower"
curl "http://localhost:8083/raft/state"  # Should return "Follower"

# RESTARTING LEADER:
go run . --node_id=leader --raft_addr=127.0.0.1:9001 --data_dir=leader_data --http_port=8081

#API Calls:

Add printer:
curl -X POST http://localhost:8081/api/v1/printers -H "Content-Type: application/json" -d '{"id":"printer1","company":"HP", "model": "L12"}'

Add filament:
curl -X POST http://localhost:8081/api/v1/filaments -H "Content-Type: application/json" -d '{"id":"f1","type":"PVC", "color": "Black","total_weight_in_grams":100, "remaining_weight_in_grams":100}'

Make print job:
curl -X POST http://localhost:8081/api/v1/print_jobs -H "Content-Type: application/json" -d '{"job_id":"job1","printer_id":"printer1","filament_id":"f1","print_weight_in_grams":50,"status":"Queued","details":"mountain"}'

Update Print Job:
curl -X PUT "http://localhost:8081/api/v1/print_jobs/job1/status?status=Done"

