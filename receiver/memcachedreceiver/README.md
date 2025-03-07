# Memcached Receiver

| Status                   |           |
| ------------------------ |-----------|
| Stability                | [beta]    |
| Supported pipeline types | metrics   |
| Distributions            | [contrib] |

This receiver can fetch stats from a Memcached instance using the [stats
command](https://github.com/memcached/memcached/wiki/Commands#statistics). A
detailed description of all the stats available is at
https://github.com/memcached/memcached/blob/master/doc/protocol.txt#L1159.

## Details

## Configuration

> :information_source: This receiver is in beta and configuration fields are subject to change.

The following settings are required:

- `endpoint` (default: `localhost:11211`): The hostname/IP address and port or, unix socket file path of the memcached instance

The following settings are optional:

- `collection_interval` (default = `10s`): This receiver runs on an interval.
Each time it runs, it queries memcached, creates metrics, and sends them to the
next consumer. The `collection_interval` configuration option tells this
receiver the duration between runs. This value must be a string readable by
Golang's `ParseDuration` function (example: `1h30m`). Valid time units are
`ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`.

Example:

```yaml
receivers:
  memcached:
    endpoint: "localhost:11211"
    collection_interval: 10s
```

The full list of settings exposed for this receiver are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).

## Metrics

Details about the metrics produced by this receiver can be found in [metadata.yaml](./metadata.yaml) with further documentation in [documentation.md](./documentation.md)

### Feature gate configurations

#### Transition from metrics with "direction" attribute

There is a proposal to change some memcached metrics from being reported with a `direction` attribute to being
reported with the direction included in the metric name.

- `memcached.network` will become:
  - `memcached.network.sent`
  - `memcached.network.received`

The following feature gates control the transition process:

- **receiver.memcachedreceiver.emitMetricsWithoutDirectionAttribute**: controls if the new metrics without
  `direction` attribute are emitted by the receiver.
- **receiver.memcachedreceiver.emitMetricsWithDirectionAttribute**: controls if the deprecated metrics with
  `direction`
  attribute are emitted by the receiver.

##### Transition schedule:

The final decision on the transition is not finalized yet. The transition is on hold until
https://github.com/open-telemetry/opentelemetry-specification/issues/2726 is resolved.

[beta]:https://github.com/open-telemetry/opentelemetry-collector#beta
[contrib]:https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
