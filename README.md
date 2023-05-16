# ScienceLogic Zebrium Exporter

| Status                   |           |
| ------------------------ |-----------|
| Stability                | [beta]    |
| Supported pipeline types | logs      |
| Distributions            | [contrib] |

Exports data via HTTP to Zebrium

## Getting Started

The following settings are required:

- `endpoint` (no default): The target URL to send Zebrium log streams to (e.g.: `https://cloud.zebrium.com`).
- `ze_token` (no default): Authorization token for the Zebrium deployment

Optional settings:

- `verbosity` (default 'normal'): Use 'detailed' to log all incoming log records

The ScienceLogic Zebrium Exporter relies on the [ScienceLogic Log Format Processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/processor/sllogformatprocessor/README.md)
to populate the following attributes:

- `sl_metadata`: Resource attribute with encoded metadata that identifies logs from a single application instance
- `sl_msg`: Log record attribute with the body of the log message formated for ScienceLogic
- `sl_format`: Format option from the matching profile, e.g. event

The following example shows how to configure the format processor and exporter in a pipeline:

```yaml
receivers:
  windowseventlog:
    channel: application
    start_at: end

processors:
  sllogformat:
    send_batch_size: 10000
    timeout: 10s
    profiles:
    - service_group: lit.default:ze_deployment_name # windows event log
      host: body.computer:host
      logbasename: body.provider.name:logbasename:rmprefix=Microsoft-Windows-:alphanum:lc
      labels:
      - body.channel:win_channel
      - body.keywords:win_keywords
      message: body.message||body.event_data||body.keywords
      format: event
    - service_group: lit.default:ze_deployment_name # docker logs
      host: rattr.host.name:host
      logbasename: attr.container_id:logbasename
      labels:
      - rattr.os.type
      - attr.log.file.path:zid_path
      message: body
      format: container

exporters:
  slzebrium:
    verbosity: detailed
    endpoint: https://cloud.zebrium.com
    ze_token: 0000000000000000000000000000000000000000

service:
  pipelines:
    logs:
      receivers: [windowseventlog]
      processors: [sllogformat]
      exporters: [slzebrium]
```

## Advanced Configuration

Several helper files are leveraged to provide additional capabilities automatically:

- [HTTP settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md)
- [Queuing and retry settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md)

[beta]:https://github.com/open-telemetry/opentelemetry-collector#beta
[contrib]:https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
