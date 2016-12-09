+++
linktitle = "database-sql"
date = "2016-12-09T00:00:00"
author = [ "Daniel Theophanes" ]
title = "New features in go1.8 database/sql"
series = ["Advent 2016"]
+++


[database/sql](https://beta.golang.org/pkg/database/sql/) has many new
features to better enable writing and controlling queries. In short it adds
support for:

 * Cancelable queries
 * Returning the SQL database types
 * Returning multiple result sets
 * Ping hitting the database server
 * Named parameters
 * Transaction isolation levels

**Cancelable Queries**

There is now support for [Context](https://golang.org/pkg/context#Context) for
most database methods. Why would you want to use them? Context allows queries
to be canceled while they are running. Reasons queries may block:

 * If a connection pool is starved it may wait indefinitely for a free
   connection.
 * If the database uses locks (which even most MVCC system do) it may block
   on a resource during a period of contention.
 * If your database is large or if the system lacks indexes or if the query
   planner does a poor job on a query the query may run much longer then
   desired.
 * A long query or transaction may compound other queries by also locking
   resources that it is using at the time.

Previously the only solution for many of these situations was manual intervention.
While steps can be taken to mitigate these, the failure mode is brittle.
Similar to initial bridges made from cast iron, they exhibit catastrophic
failure modes. Context adds the elasticity needed for a graceful deformation
and recovery. With the Context methods:

 * If a query blocks on getting a connection from a pool it can now be
   canceled and the process recovered.
 * If a query blocks on a database system lock can cancel the query
   in the database system itself as well and moving on in the goroutine.
 * By providing a way to cancel queries in the database system itself
   queries that are long running and blocking other queries
   will be stopped and the other queries they are blocking will be allowed
   to proceed.
 * Clients can ensure connections and transactions are returned to the
   connection pool when the context is canceled.

**SQL Database Type**

Drivers now have the option to return the specific database types returned
from a query. This enables another class of operations to be written using
the `database/sql` package such as [ETL](https://en.wikipedia.org/wiki/Extract,_transform,_load)
processes and serializations which need to understand what the underlying
type is (both a DATE and TIMESTAMP will be scanned into a `time.Time`).

**Multiple Result Sets**

Drivers may now support returning multiple result sets. This means for
a [*sql.Rows](http://beta.golang.org/pkg/database/sql/#Rows) rows may be
advanced through, when the last row in the row has been read,
[NextResultSet](http://beta.golang.org/pkg/database/sql/#Rows.NextResultSet)
may be called to advance to the next row sets.

This is useful when loading data has multiple natural arities or totally
different related results, especially when the initial query is expensive.
This can be done with mutliple Query calls in a transaction, but that is
harder to put in a framework or make loadable from a SQL file resource.

**Ping may hit Database**

The previous behavior of [Ping](http://beta.golang.org/pkg/database/sql/#DB.Ping)
was to ensure a live connection in the connection pool and return.
This worked when opening up a database initially because in that case
a new connection was established and the result returned.

With the new [driver Pinger](http://beta.golang.org/pkg/database/sql/driver/#Pinger)
interface, if a driver implements it [DB.Ping](http://beta.golang.org/pkg/database/sql/#DB.Ping)
and [DB.PingContext](http://beta.golang.org/pkg/database/sql/#DB.PingContext) will now allow
the driver to hit the database to ensure it is still alive and well.

**Named Parameters**

Named Parameters are also now supported. You many now write queries with
named parameters and then bind to them with [NamedArg](http://beta.golang.org/pkg/database/sql/#NamedArg)s.
These would typically be constructed with the helper method
[Named](http://beta.golang.org/pkg/database/sql/#Named) like
`sql.Named("ID", 5)`. It is critical to note that the `Name` field used
in `NamedArg` *MUST NOT* pass in a leading symbol. It is up to drivers
to add any required symbol prefix. Passing in "@ID" will always result in an error.

**Transaction Isolation**

Transaction isolation levels may be set now if the driver and database system
supports it. This is done by setting the isolation level in the context
with [IsolationContext](http://beta.golang.org/pkg/database/sql/#IsolationContext)
and then passing that context to
[DB.BeginContext](http://beta.golang.org/pkg/database/sql/#DB.BeginContext).

---

Don't blindly pass in any context you happen to have around. Know the lifespan
of the context first. If you are running a read query to return to the client,
it is appropriate to pass in the [http.Request.Context](https://golang.org/pkg/net/http/#Request.Context)
so that if the browser cancels the connection or closes the TCP connection,
the associated query will also be canceled.

However if you want to ensure that a piece of data is saved to the databases
even if the user doesn't stick around on the page or closes the tab, then you
should probably use an isolated context with a different lifespan.

If you are actively doing or interested in doing non-trivial query processes or ETL work in Go
please reach out to [kardianos](mailto:kardianos@gmail.com). I would like to better
understand your needs.

