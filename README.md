# Howler

[![Coverage Status](https://coveralls.io/repos/zalando-techmonkeys/howler/badge.svg?branch=master&service=github)](https://coveralls.io/github/zalando-techmonkeys/howler?branch=master)
[![Go Report Card](http://goreportcard.com/badge/zalando-techmonkeys/howler)](http://goreportcard.com/report/zalando-techmonkeys/howler)

Howler registers to events of cluster-managers
([Marathon](https://github.com/mesosphere/marathon),
[Kubernetes](https://github.com/kubernetes/kubernetes)) and process
them in an event based, flexible way using different backends.
Backends can be enabled using build flags.

## Project Context and Features

In the world of cluster-managers there are several kinds of features,
that you miss to get it in production right now. If you look into the
details of alerting and monitoring your endpoints, adding and removing
loadbalancer members, secret distribution, then you may want to have
an event driven approach that can dynamically adjust your components.
Howler provides this event driven approach in which you just add and
delete single resources instead of rebuilding the "world".

### Marathon

Howler is a service which is intended to be an endpoint to receive
events from the Marathon Event Bus and process them in arbitrary
backends.

In case of [Mesos](http://mesos.apache.org/) and
[Marathon](https://github.com/mesosphere/marathon) stack there are at
least two competitors
[Bamboo](https://github.com/QubitProducts/bamboo)
[Marathon-LB](https://github.com/mesosphere/marathon-lb), that
basically generate a complete new HAProxy configuration, check the
configuration and reload HAProxy. While testing the setup with
[Bamboo](https://github.com/QubitProducts/bamboo) we realized, that we
want to have a much more dynamic tool that distribute events to
backends, which can react on them in a much more dynamic and stable
way.

#### Deployment concept

F5 LTM loadbalancer and F5 GTM DNS server integration, which also
shows [chimp](https://github.com/zalando-techmonkeys/chimp) a PAAS
style deployment tool:
![LTM/GTM integration](https://raw.githubusercontent.com/zalando-techmonkeys/howler/master/docs/Loadbalancer_ltm_gtm_integration.svg)

### Kubernetes

Currently we have no Kubernetes support, but we
[planned](https://github.com/zalando-techmonkeys/howler/issues/9) to
do it.

## Requirements

You need to have a running [Mesos](http://mesos.apache.org/)-[Marathon](https://github.com/mesosphere/marathon)
setup and [Go](https://golang.org/) installed on your desk.

## Install

Assuming you've installed Go on your desk and a GOPATH environment variable, run this:

```bash
# install godep if you don't have it
go get github.com/tools/godep
# get howler
go get github.com/zalando-techmonkeys/howler
cd $GOPATH/src/github.com/zalando-techmonkeys/howler
# install required dependencies
godep restore
# install to $GOBIN
godep go install -tags zalando github.com/zalando-techmonkeys/howler/...
# for tagging the build (where the `-tags` parameter is used to enable certain backend sets from [backendconfig](./backendconfig/) ):
godep go install -tags zalando -ldflags "-X main.Buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.Githash=`git rev-parse HEAD`" github.com/zalando-techmonkeys/howler/...
```

This should compile the server binary `howler` and put it into $GOBIN, which you can put in `/usr/bin/` and start with this [init-script](howler.init.d).

## Usage

Configure Marathon to send events to howler.
The URL of the endpoint should target to howler, which you have to configure howler and marathon accordingly:

    [marathon-host]% cat /etc/marathon/conf/event_subscriber
    http_callback
    [marathon-host]% cat /etc/marathon/conf/http_endpoints
    http://my-howler-host:12345/events

### Sample Config

Create a file `~/.config/howler/config.yaml` or `/etc/howler/config.yaml` with something like this:

```yaml
---
fluentdEnabled: true
debugEnabled: true
oauth2Enabled: false
authURL:  https://token.auth.zalando.com/access_token
tokenURL: https://auth.zalando.com/z/oauth2/tokeninfo
tlsCertfilePath: /path/to/your/certfile
tlsKeyfilePath: /path/to/your/keyfile
logFlushInterval: 5 #in seconds
port: 12345
backends:
    myCustomBackend:
        Url: https://foo.net/rest/api/v1/endpoint/
        User: jdoe
        Password: Secr3tP4ss
```

## Developement

To be actually useful, there have to be [backends](./backend) which
process the events coming from marathon in some way. All of these
backends have to fulfill the [`Backend` interface](backend/backend.go).
Following Marathon event types are currently dispatched and processed
by the respective methods:

- [api_post_event](http://mesosphere.github.io/marathon/docs/event-bus.html#api-request), handled by `HandleCreate()`
- [status_update_event](http://mesosphere.github.io/marathon/docs/event-bus.html#status-update), handled by `HandleUpdate()`
- [app_terminated_event](https://github.com/mesosphere/marathon/issues/1530), handled by `HandleDestroy()`

Have a look on the [dummy backend](backend/dummy.go) for an example.

## Contributing/TODO

We welcome contributions from the community; just submit a pull
request. To help you get started, here are some items that we'd love
help with:

- add Kubernetes (another Cluster-Manager) as event source
- the code base

Please use github issues as starting point for contributions, new
ideas or bugreports.

## Contact

* E-Mail: team-techmonkeys@zalando.de
* IRC on freenode: #zalando-techmonkeys

## Contributors

Thanks to:

- &lt;your name&gt;

## License

See [LICENSE](LICENSE) file.
