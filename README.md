# ScienceLogic Format Processor

| Status                   |                       |
| ------------------------ | --------------------- |
| Stability                | logs [beta]           |
| Supported pipeline types | logs                  |
| Distributions            | [contrib]             |

The ScienceLogic format processor accepts logs and places them into
batches annotated with the attributes required for processing by
other ScienceLogic components.  Each batch is forwarded as a
resource log entry with resource attributes that identify the log
stream, i.e. from an instance of an application running on a single
host.  The configuration describes where to find the appropriate
attributes and how to present them in the metadata attributes.
To work with different receivers, you can define multiple `profiles`
that match against incoming logs in the order configured.  A match
requires the following attributes:

- `service_group`: Domain of anomaly correlation
- `host`: Host or computer name
- `logbasename`: Application in lowercase, e.g. postgres
- `severity`: [OPTIONAL] Location to find severity value

Additional optional attributes can be configured as `labels`.
These attributes can be derived from the following sources in the
incoming stream:

- `lit`: A literal injected from the configuration
- `rattr`: Resource attribute, also called resource label
- `attr`: Log record attribute
- `body`: The message body as a string or map to elements within

The syntax for associating metadata looks like:

```
<destination>:
  exp:
    source: <source>[:<key path>]
  rename: <replacement name>
```

If `rename` is omitted, the key path is used as the attribute
name if available followed by the resulting value.  It is
recommended to specify `rename` for service group, host,
logbasename, and all literals.

In addition to sources an expression can be formed from the
following operators with associated expressions A and B:

- `rmprefix`: Remove prefix B from A if matched
- `rmsuffix`: Remove suffix B from A if matched
- `rmtail`: Remove everything from A after and including the last match of B
- `alphanum`: Filter out all characters that are not letters or numbers from A
- `unescape`: Filter out ESC character
- `lc`: Transform A to lowercase
- `regexp`: Concatinate all captures from A using golang regexp B
- `and`: Concatinate all results from expressions
- `or`: Return the first expression result that is not empty

The syntax for operators looks like:

```
<destination>:
  exp:
    op: <operator>
    exps:
    - <expression A>
    - <expression B>
    - <expression C ...>
  rename: <replacement name>
```

The expressions under `exps` are either `source` or a single `op`
with associated `exps` of its own.

Profiles have an additional configuration for the message `format`
with the following values:

- `event`: Prefix the message with timestamp and severity
- `message`: Forward the message body as is
- `container`: Special handling for logs from docker containers

The attributes assigned by this processor for consumption by
ScienceLogic commponents include the following resource attributes:

- `sl_service_group`: Domain of anomaly correlation
- `sl_host`: Host or computer name
- `sl_logbasename`: Application in lowercase, e.g. postgres
- `sl_format`: Format option from the matching profile
- `sl_metadata`: An encoding of all log stream metadata

And the following log record attribute:

- `sl_msg`: The log message body formatted for consumption by ScienceLogic

The following configuration options can be modified:

- `send_batch_size` (default = 8192): Number of spans, metric data points, or log
records after which a batch will be sent regardless of the timeout.
- `timeout` (default = 200ms): Time duration after which a batch will be sent
regardless of size.
- `send_batch_max_size` (default = 0): The upper limit of the batch size.
  `0` means no upper limit of the batch size.
  This property ensures that larger batches are split into smaller units.
  It must be greater than or equal to `send_batch_size`.

Examples:

```yaml
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
      labels:
      - exp:
          source: body:channel
        rename: win_channel
      - exp:
          source: body:keywords
        rename: win_keywords
      message:
        exp:
          op: or
          exps:
            - source: body:message
            - source: body:event_data
            - source: body:keywords
      format: event
    - service_group: # docker logs
        exp:
          source: lit:default
        rename: ze_deployment_name
      host:
        exp:
          source: rattr:host.name
        rename: host
      logbasename:
        exp:
          source: attr:container_id
        rename: logbasename
      labels:
      - exp:
          source: rattr:os.type
      - exp:
          source: attr:log.file.path
        rename: zid_path
      message:
        exp:
          source: body
      format: container
```

[beta]: https://github.com/open-telemetry/opentelemetry-collector#beta
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
[core]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol
