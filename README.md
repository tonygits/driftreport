# Drift Report
## Overview

- The program should compare two configurations and detect drift across a list of specified attributes, including but
  not limited to instance_type.
- It should return a json of whether a drift is detected for any attribute in the list and specify which attributes

## Getting Up and Running

### Create local env file

```sh
cp .env.sample .env
```

### Setup AWS Environment

```sh
export AWS_ACCESS_KEY_ID={your_access_key}
export export AWS_SECRET_ACCESS_KEY={your_secret_key}
```

### To run unit tests and coverage

```sh
go test ./...
go test -coverprofile=coverage.out ./...
```

### To run the application

```sh
go run cmd/main.go
```
