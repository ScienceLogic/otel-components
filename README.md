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
    - service_group: # windows event log
        exp:
          source: lit:default
        rename: ze_deployment_name
      host:
        exp:
          source: body:computer
        rename: host
      logbasename:
        exp:
          op: lc
          exps:
          - op: alphanum
            exps:
              - op: rmprefix
                exps:
                  - source: body:provider.name
                  - source: lit:Microsoft-Windows-
        rename: logbasename
      message:
        exp:
          op: or
          exps:
            - source: body:message
            - source: body:event_data
            - source: body:keywords
      format: event

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
