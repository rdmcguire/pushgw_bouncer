# Prometheus Pushgateway Bouncer

The purpose of this tool is to restart some process or container
should the last push to a prometheus pushgateway take too long.

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
