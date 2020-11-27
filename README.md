# SCHEDULER

This is a distributed scheduler based on RabbitMQ that allows you to execute 
tasks on specific time.

## Supported task types
- Remote http (POST/GET with specified body)
- Local cmd
- Dummy (for testing or just for notification callback)

## Prerequisites
- [RabbitMQ](https://www.rabbitmq.com/#getstarted) ^3.8.x
- [RabbitMQ Delayed Message Plugin](https://github.com/rabbitmq/rabbitmq-delayed-message-exchange) installed
- [docker](https://www.docker.com/get-started)
- [docker-compose](https://docs.docker.com/compose/install/) ^1.26.0 or higher
- go 1.15 or higher

Please note limitations of MQ delayed messages that are applied same for this project, eg. task
can be scheduled up to ~47days from its publication date

## Installation

``
docker-compose up -d --build
``

By default, service will be available on port ``8245`` via grpc, monitoring via 
grafana on port ``3000``.
Raw prometheus metrics exposed on ``0.0.0.0:8457/metrics``.
Both Grafana and Prometheus included for demonstration purposes. 

If you want to run it locally after cloning repository create ``config.yaml`` 
from ``config.example.yaml``  in project root directory and run ``go run main.go``

Generated API and ``*.proto`` files are available [here](https://github.com/DoomSentinel/scheduler-api)

Project includes ``client`` package for external integrations 

You can install example application by running

``GO111MODULE=on go get github.com/DoomSentinel/scheduler/cmd/scheduler-example``

####Usage
```
  scheduler-example --host=0.0.0.0 --port=8245  
```

## Configuration

Configuration passed by environment variables, defaults defined in ``config.example.yaml`` 
in project root

|Variable|Default|Description|
| --- | --- | --- |
| API_GRPC_PORT | 8245 | Listen for for GRPC API |
| AMQP_HOST | 127.0.0.01 | RabbitMQ broker host |
| AMQP_PORT | 5672 | RabbitMQ broker port |
| AMQP_USER | guest | RabbitMQ auth user |
| AMQP_PASSWORD | guest | RabbitMQ auth password |
| MONITORING_PORT | 8457 | Port of http service where Prometheus metrics are exposed |
| DEBUG_DEBUG | false | Enables debug mode |

## Project maturity

Currently, not battle tested - therefore not ready for production usage. 
Needs more ~~vespene gas~~ tests.
