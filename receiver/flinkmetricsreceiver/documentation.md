[comment]: <> (Code generated by mdatagen. DO NOT EDIT.)

# flinkmetricsreceiver

## Metrics

These are the metrics available for this scraper.

| Name | Description | Unit | Type | Attributes |
| ---- | ----------- | ---- | ---- | ---------- |
| **flink.job.checkpoint.count** | The number of checkpoints completed or failed. | {checkpoints} | Sum(Int) | <ul> <li>checkpoint</li> </ul> |
| **flink.job.checkpoint.in_progress** | The number of checkpoints in progress. | {checkpoints} | Sum(Int) | <ul> </ul> |
| **flink.job.last_checkpoint.size** | The total size of the last checkpoint. | By | Sum(Int) | <ul> </ul> |
| **flink.job.last_checkpoint.time** | The end to end duration of the last checkpoint. | ms | Gauge(Int) | <ul> </ul> |
| **flink.job.restart.count** | The total number of restarts since this job was submitted, including full restarts and fine-grained restarts. | {restarts} | Sum(Int) | <ul> </ul> |
| **flink.jvm.class_loader.classes_loaded** | The total number of classes loaded since the start of the JVM. | {classes} | Sum(Int) | <ul> </ul> |
| **flink.jvm.cpu.load** | The CPU usage of the JVM for a jobmanager or taskmanager. | % | Gauge(Double) | <ul> </ul> |
| **flink.jvm.cpu.time** | The CPU time used by the JVM for a jobmanager or taskmanager. | ns | Sum(Int) | <ul> </ul> |
| **flink.jvm.gc.collections.count** | The total number of collections that have occurred. | {collections} | Sum(Int) | <ul> <li>garbage_collector_name</li> </ul> |
| **flink.jvm.gc.collections.time** | The total time spent performing garbage collection. | ms | Sum(Int) | <ul> <li>garbage_collector_name</li> </ul> |
| **flink.jvm.memory.direct.total_capacity** | The total capacity of all buffers in the direct buffer pool. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.direct.used** | The amount of memory used by the JVM for the direct buffer pool. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.heap.committed** | The amount of heap memory guaranteed to be available to the JVM. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.heap.max** | The maximum amount of heap memory that can be used for memory management. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.heap.used** | The amount of heap memory currently used. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.mapped.total_capacity** | The number of buffers in the mapped buffer pool. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.mapped.used** | The amount of memory used by the JVM for the mapped buffer pool. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.metaspace.committed** | The amount of memory guaranteed to be available to the JVM in the Metaspace memory pool. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.metaspace.max** | The maximum amount of memory that can be used in the Metaspace memory pool. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.metaspace.used** | The amount of memory currently used in the Metaspace memory pool. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.nonheap.committed** | The amount of non-heap memory guaranteed to be available to the JVM. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.nonheap.max** | The maximum amount of non-heap memory that can be used for memory management. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.memory.nonheap.used** | The amount of non-heap memory currently used. | By | Sum(Int) | <ul> </ul> |
| **flink.jvm.threads.count** | The total number of live threads. | {threads} | Sum(Int) | <ul> </ul> |
| **flink.memory.managed.total** | The total amount of managed memory. | By | Sum(Int) | <ul> </ul> |
| **flink.memory.managed.used** | The amount of managed memory currently used. | By | Sum(Int) | <ul> </ul> |
| **flink.operator.record.count** | The number of records an operator has. | {records} | Sum(Int) | <ul> <li>operator_name</li> <li>record</li> </ul> |
| **flink.operator.watermark.output** | The last watermark this operator has emitted. | ms | Sum(Int) | <ul> <li>operator_name</li> </ul> |
| **flink.task.record.count** | The number of records a task has. | {records} | Sum(Int) | <ul> <li>record</li> </ul> |

**Highlighted metrics** are emitted by default. Other metrics are optional and not emitted by default.
Any metric can be enabled or disabled with the following scraper configuration:

```yaml
metrics:
  <metric_name>:
    enabled: <true|false>
```

## Resource attributes

| Name | Description | Type |
| ---- | ----------- | ---- |
| flink.job.name | The job name. | String |
| flink.resource.type | The flink scope type in which a metric belongs to. | String |
| flink.subtask.index | The subtask index. | String |
| flink.task.name | The task name. | String |
| flink.taskmanager.id | The taskmanager ID. | String |
| host.name | The host name. | String |

## Metric attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| checkpoint | The number of checkpoints completed or that failed. | completed, failed |
| garbage_collector_name (name) | The names for the parallel scavenge and garbage first garbage collectors. | PS_MarkSweep, PS_Scavenge, G1_Young_Generation, G1_Old_Generation |
| operator_name (name) | The operator name. |  |
| record | The number of records received in, sent out or dropped due to arriving late. | in, out, dropped |
