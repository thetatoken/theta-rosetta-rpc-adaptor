# theta-rosetta-rpc-adaptor
<p align="center">
  <a href="https://www.rosetta-api.org">
    <img width="90%" alt="Rosetta" src="https://www.rosetta-api.org/img/rosetta_header.png">
  </a>
</p>
<h3 align="center">
   Theta Rosetta
</h3>

## Overview
`Theta-rosetta` provides a reference implementation of the [Rosetta specification](https://github.com/coinbase/rosetta-specifications) for Theta.

## Features
* Comprehensive tracking of all ETH balance changes
* Stateless, offline, curve-based transaction construction (with address checksum validation)
* Atomic balance lookups using go-ethereum's GraphQL Endpoint
* Idempotent access to all transaction traces and receipts

## Pre-requisite
To run `Theta-rosetta`, you must install Docker. Please refer to [Docker official documentation](https://docs.docker.com/get-docker/) on installation instruction.

## Usage
As specified in the [Rosetta API Principles](https://www.rosetta-api.org/docs/automated_deployment.html),
all Rosetta implementations must be deployable via Docker and support running via either an
[`online` or `offline` mode](https://www.rosetta-api.org/docs/node_deployment.html#multiple-modes).

**YOU MUST INSTALL DOCKER FOR THE FOLLOWING INSTRUCTIONS TO WORK. YOU CAN DOWNLOAD
DOCKER [HERE](https://www.docker.com/get-started).**

## Install
Running the following commands will create a Docker image called `theta-rosetta-rpc-adaptor:latest`.

_Get the Dockerfile from this repository and put it into your desired local directory_

```text
docker build --no-cache -t theta-rosetta-rpc-adaptor:latest .
```

## Run
Running the following command will start a Docker container and expose the Rosetta APIs.
```shell script
docker run -p 8080:8080 -p 16888:16888 -p 15872:15872 -p 21000:21000 -p 30001:30001 -e THETA_NETWORK=testnet -it theta-rosetta-rpc-adaptor:latest
```

### Restarting `Theta-rosetta`

```
docker stop <container name>
docker start <container name>
```

## Restful APIs

### Rosetta restful APIs

#### All Data APIs specified in https://www.rosetta-api.org/docs/data_api_introduction.html

#### All supported Construction APIs specified in https://www.rosetta-api.org/docs/ConstructionApi.html


### Unsupported APIs

Indexer APIs specifed in https://www.rosetta-api.org/docs/indexers.html


## How to test
Install the latest rosetta-cli from https://github.com/coinbase/rosetta-cli.

### Testing Data API

```
rosetta-cli check:data --configuration-file=cli-test-config.json
```

### Testing Construction API

```
rosetta-cli check:construction --configuration-file=cli-test-config.json
```

### End Conditions

The end conditions for `check:construction` is set to:
```
  broadcast complete for job "transfer (3)" with transaction hash <tx hash>
```


## License

This project is available open source under the terms of the [GNU Lesser General Public License 3.0](LICENSE.md).

© 2021 Theta
