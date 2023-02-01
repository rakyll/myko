# myko

myko is a simple attribution engine. It can be used for several use cases such as:

* Analyzing the cost of certain features on shared infrastructure.
* Analyzing the origin of expensive operations.
* Analyzing the load on downstream components in the lifetime of a request.
* Estimating costs and billing.
* Capacity management and forecasting capacity needs.

## Concepts

myko has three fundamental concepts:

* **Origin**: Origin of the event. It can be a request handler, a background job,
or an identifier like a customer ID.
* **Event**: Any measurable action identifiable with a name & unit. It could be anything
defined by the user, such as a query to MySQL, a checkout transaction,
an expensive encoding job.
* **Target**: A target downstream resource to be impacted by the event created at origin, e.g.
a database cluster, a storage backend, or any commonly shared service.


For example, while loading navigation bar, the following events can be collected:

```
{ target: "webserver", origin: "site_navbar", event_name: "render_ms", value: 23.1 }
{ target: "webserver-mysql", origin: "site_navbar", event_name: "sql_query_latency_ms", value: 40.5 }
{ target: "webserver-mysql", origin: "site_navbar", event_name: "sql_query_latency_ms", value: 12.5 }
{ target: "webserver-mysql", origin: "site_navbar", event_name: "sql_query_latency_ms", value: 11.5 }
{ target: "webserver-mysql", origin: "site_navbar", event_name: "sql_query_count", value: 3 }
```

myko ingests the above events and can report:

* How site navigation impact total rendering hours spent on the webserver.
* How site navigation impact the total load on webserver-mysql instance.
* What type of events started at site navigation and what resources they are targeting.

See the [examples](https://github.com/rakyll/myko/tree/main/examples/) directory for example programs.


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
by target, origin, event name & unit. Depending of the cardinality of these
attributes, thousands of events can be aggregated to a few data points.

**Does myko have any cardinality limits?**

We decided not to set any limits and leave it up to the user. Cardinality
can impact the write and query latency. We will work on optimizations and
new compaction methods where possible.

**Do you have any plans for other datastores?**

We are initially only supporting Kusto but would like to introduce 
support for other datastores if we can make maintenance commitments.