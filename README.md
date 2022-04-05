# Prometheus Pushgateway Bouncer

The purpose of this tool is to restart some process or container
should the last push to a prometheus pushgateway be too far in the past.

To use, simply push anything into your pushgateway with at least one identifying
label, match on that with label_name and label_value, and configure the restart action.

**TODO** Currently pushgwAPI.go uses the time_stamp field of the push_time_seconds metric.
This should be extended to permit specifying a different metric name.

## Currently supported handlers:

- LXD
- Docker

## Currently supported actions:

As of right now your options are to run a command inside a container or
restart an entire container. More could be added by extending the handlers.Handler interface.

- Run a command in the container
	- Example: systemctl restart <someprocess>
- Restart the entire container

## Configuration

The configuration file is documented in the config and metric structs.

Command-line parameters take precedence over config file parameters.

The label_name and label_value fields are for matching the correct metric among all pushgateway metrics.
At the top level, metrics are grouped by unique labels, in the below example a label called job
is being used to match, and the value is given in label_value.

**Docker Container Restart Monitor**
```yaml
monitors:
  - name: API Exporter
    max_age: 1h30m
    type: docker
    container_name: prom-stack_api-exporter_1
    label_name: job
    label_value: api_exporter
    restart_type: container
```

**LXD Container Command Example**
```yaml
monitors:
  - name: WeeWX
    max_age: 5m
    type: lxd
    container_name: weewx
    label_name: job
    label_value: weewx
    restart_type: command
    restart_command:
      - /bin/systemctl
      - restart
      - weewx
```

**Global Settings**
```yaml
global:
  check_interval: 1m
  log_level: info
  socket_lxd: /var/snap/lxd/common/lxd/unix.socket
  socket_docker: /var/run/docker.sock
  push_gw: http://pushgateway:9091
  addr: :9090
```

## Running

Dockerfile and docker-compose file provided

- Be sure to mount socket files into the container
