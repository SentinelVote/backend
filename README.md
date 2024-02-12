# SentinelVote Backend

## Setup

```sh
git clone https://github.com/SentinelVote/backend.git
cd backend
go run . # add --help or -h for CLI flags.
```

## Build
```sh
CGO_ENABLED=0 go build -o ./api
./api # add --help or -h for CLI flags.
```

## Contributor Notes

### Read-Only Files

Files which are read-only are finalized and should not be modified.
