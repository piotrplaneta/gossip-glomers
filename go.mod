module github.com/piotrplaneta/gossip-glomers

go 1.20

require internal/handlers v1.0.0

require github.com/jepsen-io/maelstrom/demo/go v0.0.0-20230417134804-423a6b11c462

require github.com/google/uuid v1.3.0 // indirect

replace internal/handlers => ./internal/handlers
