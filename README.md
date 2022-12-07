# myko

myko is a simple attribution engine. It can be used for several use cases such as:

* Analyzing the cost of certain features on shared infrastructure.
* Analyzing the origin of expensive operations.
* Analyzing the load on downstream components in the lifetime of a request.
* Estimating costs and billing.
* Capacity management and forecasting capacity needs.

## Usage

myko currently only supports Cassandra and Cassandra-compatible datastores.

``` bash
$ cat config/config.yaml
data:
    cassandra:
        peers:
            - node-0.cassandra.host
            - node-1.cassandra.host
            - node-2.cassandra.host
        username: cassandra
        password: *********

$ docker run -it -p 6959:6959 -v $(pwd)/config:/config \
    public.ecr.aws/q1p8v8z2/myko:latest -config /config/config.yaml
```

## Concepts

myko has three fundamental concepts:

* **Event**: Any measurable action identifiable with a name & unit. It could be anything
defined by the user, such as a query to MySQL, a checkout transaction,
an expensive encoding job.
* **Origin**: Origin of the event. It can be a request handler, a background job,
or an identifier like a customer ID.
* **Trace ID**: Used optionally to debug events happening in the lifetime of a debug request.

For example, while loading navigation bar, the following events can be collected:

```
{ trace_id: "xxx", origin: "site_navbar", event_name: "render", unit: "ms", value: 23.1 }
{ trace_id: "xxx", origin: "site_navbar", event_name: "sql_query_latency", unit: "ms", value: 40.5 }
{ trace_id: "xxx", origin: "site_navbar", event_name: "sql_query_latency", unit: "ms", value: 12.5 }
{ trace_id: "xxx", origin: "site_navbar", event_name: "sql_query_latency", unit: "ms", value: 11.5 }
{ trace_id: "xxx", origin: "site_navbar", event_name: "sql_query_count", unit: "", value: 3 }
```

myko ingests the events and can report:

* The total cost of rendering and SQL querying in the lifetime of trace ID, xxx.
* The highest contributor origin of database load.
* Events started at origins to help you analyze dependencies between services or components.

See the [examples](tree/main/examples) directory for example programs.


## FAQ

**Why did you create myko?**

We had a large number of shared resources such as databases, message queues,
and networking components.
Organic growth of our systems and their dependencies to shared resources
made it really hard to investigate where the load or capacity needs were coming from.
Due to the complex nature of our systems, we needed high levels of
cardinality to have the right granularity when it comes to slicing and dicing
telemetry. We couldn't position our existing metric collection service to do the job,
and wanted to build something optimized for attribution analysis. 

**How is myko different than metric collection systems?**

myko is not a timeseries database and doesn't support all aggregations metric
collection systems provide. myko ingests events and aggregates them as sums.
Aggregation happens server side in a time window which can be configurable
by users. In the aggregation window, myko aggregates all incoming events into a sum
by trace ID, origin, event name & unit. Depending of the cardinality of these
attributes, thousands of events can be aggregated to a few data points.

**How is myko different than distributed tracing systems?**

myko is not a distributed tracing system. It doesn't store distributed trace spans.
We expect myko to be used together with distributed tracing systems. We accept
a trace ID in cases where users want to correlate their traces with myko data.
We recommend ingesting trace ID only for debugging purposes or if the traces
are aggressively downsampled. Trace IDs can increase the cardinality significantly
and can cause performance regressions when querying.

**Does myko have any cardinality limits?**

We decided not to set any limits and leave it up to the user. Cardinality
can impact the write and query latency. We will work on optimizations and
new compaction methods where possible.

**Do you have any plans for other datastores?**

We are initially only supporting Cassandra or Cassandra-compatible datastores
but would like to introduce support for other datastores if we can
make maintenance commitments.