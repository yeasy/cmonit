cmonit
===

Monitor for container stats, etc.

cmonit can automatically read host info from db, and check the containers (with `label=monitor=true`) status, and then write back to db.




## Installation

```sh
$ docker run -it yeasy/cmonit start
```

## Configuration
cmonit will automatically search the `cmonit.yaml` file under `.`, `$HOME`, `/etc/cmonit/` or `$GOPATH/github.com/yeasy/cmonit`.

Please see [cmonit.yaml](cmonit.yaml) for example.

A typical config file will look like
```yaml
db:
  url: "mongo:27017"    //mongo db url
  name: "dev"           //name of the db to use
  col_host: "host"      //from which collection to get host info
  col_monitor: "monitor"//store data to which collection
sync:
  interval: 60          //sync host info interval, in seconds
monitor:
  interval: 30          //monitor container info interval, in seconds
  expire: 7             //monitor data expiration, in days
```

## Usage


```sh
$ cmonit start --logging-level=debug
```
