# Prometheus Pushgateway Bouncer

The purpose of this tool is to restart some process or container
should the last push to a prometheus pushgateway be too far in the past.

## Currently supported handlers:
- LXD
- Docker

## Currently supported actions:
- Run a command in the container
	- Example: systemctl restart <someprocess>
- Restart the entire container

## Configuration

The configuration file is documented in the config and metric structs.

Command-line parameters take precedence over config file parameters.

**Docker Container Restart Monitor**
``yaml
monitors:
  - name: API Exporter
    max_age: 1h30m
    type: docker
    container_name: prom-stack_api-exporter_1
    label_name: job
    label_value: api_exporter
    restart_type: container``

**LXD Container Command Example**
``yaml
monitors:
  - name: WeeWX
    max_age: 5m
    type: lxd
    container_name: weewx
    label_name: job
    label_value: weewx
    restart_type: command
    restart_command:
      - /bin/systemd
      - restart
      - weewx``

**Global Settings**
``yaml
global:
  check_interval: 1m
  log_level: info
  socket_lxd: /var/snap/lxd/common/lxd/unix.socket
  socket_docker: /var/run/docker.sock
  push_gw: http://pushgateway:9091``

## Running

Dockerfile and docker-compose file provided

- Be sure to mount socket files into the container
