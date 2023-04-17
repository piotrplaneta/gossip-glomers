module github.com/piotrplaneta/gossip-glomers

go 1.20

require internal/echo v1.0.0

require github.com/jepsen-io/maelstrom/demo/go v0.0.0-20230417134804-423a6b11c462

replace internal/echo => ./internal/echo
