# Minimal install Git-based decentralized version control system

A Git-based decentralized version control system running on Hyperledger Fabric network using IPFS for file storage

## Components

- [Python developer-facing client](client/)
- [Smart contract in Go to be deployed on the blockchain](contract/)

## Installation requirements

Ubuntu 22.04 or newer is required for this application. Windows users should install Docker directly into their WSL (Windows Subsystem for Linux) distribution and not use Docker Desktop

Run [setup.sh](scripts/setup.sh) to install all required software, namely:

- Docker
- Docker-compose
- JQ
- Go 1.22.1
- Hyperledger Fabric 2.5.6
- IPFS 0.7.0
- Python packages (listed in [requirements.txt](client/requirements.txt))
  - fabric-sdk-py
  - cryptography
  - gitpython
  - ipfshttpclient
  - pydantic
  - pytz

## How to execute

1. Copy directories [client](client/), [contract](contract/), and [scripts](scripts/) to the home folder of your Ubuntu installation
2. Run `cd scripts/`, then `chmod a+x *.sh` to allow all scripts to be executable
3. Run `./deploy_chaincode.sh` to start the Hyperledger Fabric test network and install the chaincode
4. Run `./start_ipfs_daemon.sh` to start the daemon
5. Switch to the [client](client/) directory to execute the [client](client/main.py)

Full reference of commands supported by the client can be found [here](client/README.md).

## Cleanup

To cleanup, stop the IPFS daemon by pressing Ctrl+C in the terminal in which the daemon is running, then run the [stop_and_clean.sh](scripts/stop_and_clean.sh) scripts from inside the 'scripts' directory. This will stop the Fabric network and clean up the IPFS instance.
