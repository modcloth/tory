tory
====

Ansible inventory server

## Goals

Tory is meant to be used as a reasonably fast dynamic inventory source via the
[`hosts/tory`](./hosts/tory) inventory script, keeping track of the latest host
vars and tags.  It's also a handy, queryable snapshot of one's server estate
and provides server-side filtering options to help minimize the returned
inventory JSON.

Syncing into tory is a matter of hitting its API with a `PUT` per host.  There
is currently a syncer written for Joyent available at
[`bin/tory-sync-from-joyent`](./bin/tory-sync-from-joyent).

In addition to periodic syncs from external sources such as Joyent, tory ships
a [registration executable](./bin/tory-register) that's meant to be run locally
at intervals on each host for self-registration.  This, combined with the
default value for `since` which filters out hosts modified more than 30 days in
the past, means that a dynamic server estate can slowly change over time
without requiring explicit host cleanup.

## Non-Goals

Tory does not keep a versioned history of the things that it stores, but
instead is meant to be a queryable snapshot of what's current.  Holding on to
versioned bits for rolling back and restoring configuration versions is better
left to another tool.

## Installation

Either build it from source:

``` bash
go get github.com/modcloth/tory
```

Or download a tarball from github:

https://github.com/modcloth/tory/releases

## Usage

All commands and options are available via the built in help system:

``` bash
tory -h
```

Nearly all of the options may be provided as either environment variables or
command line options.

Assuming there's already a postgresql server running somewhere containing a
database named "tory", make sure the `DATABASE_URL` environment varable is set:

``` bash
# for example:
export DATABASE_URL="postgres://$(whoami)@localhost/tory?sslmode=disable"
```

First run migrations:

``` bash
tory migrate
```

Then run the server:

``` bash
tory serve
```


## API

### public API

All of the public API methods are prefixed with a default of `/ansible/hosts`,
which may be altered via the `TORY_PREFIX` environment variable or the
`-p`/`--prefix` option of `tory serve`:

* `GET /ansible/hosts` - returns the full inventory in ansible-compatible JSON.
Accepts the following query string variables to filter the results:
    * `name` - only return hosts with names that prefix match this value
	* `env` - only return hosts with a matching `env` tag
	* `team` - only return hosts with a matching `team` tag
* `GET /ansible/hosts/{hostname}` - returns a single host in a `host` JSON
object in the format described below.
* `PUT /ansible/hosts/{hostname}` - creates or updates a host by name with a
`host` JSON object in the format described below (*requires auth*)
* `DELETE /ansible/hosts/{hostname}` - deletes a host by name (*requires auth*)
* `GET /ansible/hosts/{hostname}/tags/{key}` - returns the value for a given
host tag as a `value` JSON object in the format described below.
* `PUT /ansible/hosts/{hostname}/tags/{key}` - creates or updates a tag for the
given host as a `value` JSON object in the format described below (*requires
auth*)
* `DELETE /ansible/hosts/{hostname}/tags/{key}` - deletes a host tag by name
(*requires auth*)
* `GET /ansible/hosts/{hostname}/vars/{key}` - returns the value for a given
host var as a `value` JSON object in the format described below.
* `PUT /ansible/hosts/{hostname}/vars/{key}` - creates or updates a var for the
given host as a `value` JSON object in the format described below (*requires
auth*)
* `DELETE /ansible/hosts/{hostname}/vars/{key}` - deletes a host var by name
(*requires auth*)

### other API stuff

* `GET /ping` - returns PONG
* `GET /debug/vars` - returns vars JSON as exposed by expvar

### authorization

All of the API methods that require authorization will expect it in the form of
an `Authorization: token` header, e.g.:

```
Authorization: token abc123
```

### `host` JSON

Tory uses the following JSON format to represent a host:

``` javascript
{
    "host": {
        "name": "test9823-1407878425102306799.example.com",
        "ip": "10.10.1.47",
        "package": "fancy-town-80",
        "image": "ubuntu-14.04",
        "type": "virtualmachine",
        "tags": {
            // string key-value pairs
        },
        "vars": {
            // string key-value pairs
        }
    }
}
```

### `value` JSON

Tory uses the following JSON format to represent a simple value, typically for
tags or vars:

``` javascript
{
    "value": "whatever"
}
```
