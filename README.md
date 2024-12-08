
## Introduction
TCP Message Processing System
- Listen for and handle TCP connections.
- Implement the communication protocol detailed below.
- Track and maintain state information.
  <img src="static/Task Distribution sequence" alt="Introduction" width="230" align="right" />
```sequence digram
title Task Distribution sequence
Server -> Server: start server
Server -> Server: start task distribute 

loop connection maintain
Cli -> Server: authorize req
Server -> Server: maintain conn,name
end

loop send job per 30s
Server -> Server: gen&record server_nonce
Server -> Server: increase job_id
Server -> Cli: send job
Cli -> Cli: gen client nonce
Cli -> Cli: calculate
Cli --> Server: submit, with rate control
Server -> Server: validate
Server -> Server: rate limit
Server -> Server: rate record
end
```

## How to Build

### Client
Mac os: `make build-client`

Linux: `make build-client-linux`

### Server
Mac os: `make build-server`

Linux: `make build-server-linux`

## How to run

Mac os: 
1. `docker-compose up`
2. `make run-server`
3. `make run-client`

Linux: 
1. `docker-compose up`
2. `make run-server`
3. `make run-client`