# AWS Timestream Exporter

| Status                   |                       |
| ------------------------ | --------------------- |
| Stability                | metrics [development] |
| Supported pipeline types | metrics               |

Export metrics to AWS Timestream metric store

The following settings are required:

- `database` (no default): The timestream database where data will be inserted.
- `table` (no default): The timestream table where data will be inserted.
- `region` (no default): The region of the timestream deployment. 


Example:

```yaml
exporters:
  awstimestream:
    database: ae-sample-db
    table: sample-table
    region: us-east-1
```
