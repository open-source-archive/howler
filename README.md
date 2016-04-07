# Howler

[![Go Report Card](https://goreportcard.com/badge/zalando-techmonkeys/howler)](https://goreportcard.com/report/zalando-techmonkeys/howler)
[![Build Status](https://travis-ci.org/zalando-techmonkeys/howler.svg?branch=master)](https://travis-ci.org/zalando-techmonkeys/howler)
[![Coverage Status](https://coveralls.io/repos/zalando-techmonkeys/howler/badge.svg?branch=master&service=github)](https://coveralls.io/github/zalando-techmonkeys/howler?branch=master)
[![License](http://img.shields.io/badge/license-MIT-yellow.svg?style=flat)](https://raw.githubusercontent.com/zalando-techmonkeys/howler/master/LICENSE)

Howler is a service that listens to events posted in the [Marathon](https://github.com/mesosphere/marathon) Event Bus, processes them in arbitrary backends, and distributes them in an event-driven, flexible way. It enables you to integrate Marathon into your infrastructure via a single interface — freeing you up from having to change all of your configurations across your entire system. Using build flags, it makes enabling different backends possible.

### Project Context and Features
Different cluster managers offer different features. Unfortunately, some of them don't make it possible to get things to production on a "right-now"/instantaneous basis. Furthermore, if you need detailed information related to alerting and monitoring your endpoints, adding and removing load-balancer members, and/or secret distribution, you might want the ability to implement an event-driven approach that allows you to dynamically adjust your components. 

Howler enables you to adopt this event-driven approach. Instead of rebuilding the "world," you just add and delete single resources.

#### A Note on Using Bamboo
In the [Mesos](http://mesos.apache.org/) and [Marathon](https://github.com/mesosphere/marathon) stack, at
least two similar projects — [Bamboo](https://github.com/QubitProducts/bamboo) and [Marathon-LB](https://github.com/mesosphere/marathon-lb) — also generate a complete new HAProxy configuration, check the
configuration, and reload the HAProxy. While testing our setup with [Bamboo](https://github.com/QubitProducts/bamboo), we realized that we wanted to have a much more dynamic tool that a) distributed events to backends, and b) that could react to them in a much more dynamic and stable way. So we created Howler. 

### Requirements

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

### Backends
Backends are components that can be plugged to implement a particular action based on the events. A backend must implement the [backend interface](./backend/backend.go) to be pluggable.

Different users of Howler might have different needs in terms of backends. This could mean using a mix of different backends or implementing more. 
To allow composability, we choose a "compilation over configuration" approach: to configure your set of backends you have to define a backend configuration similar to [this one](backendconfig/zalando.go) and select it at compile time. 
All you have to do is the following: 
- write a [backend config](backendconfig/zalando.go) with the appropiate [build tag](https://golang.org/pkg/go/build/)
- use build tags to select your configuration: ```godep go install -tags YOUR_TAG github.com/zalando-techmonkeys/howler/...```


#### Loadbalancer - F5 LTM and GTM
F5 is a manufacturer that produces hardware loadbalancers like LTM Big
IP and GTM a smart DNS server.

LTM loadbalancer and GTM DNS server integration, also
shows
[baboon-proxy](https://github.com/zalando-techmonkeys/baboon-proxy),
the most feature complete F5 RESTful API, and
[chimp](https://github.com/zalando-techmonkeys/chimp), a PAAS
style deployment tool:

![LTM/GTM integration](https://raw.githubusercontent.com/zalando-techmonkeys/howler/master/docs/Loadbalancer_ltm_gtm_integration.png)

#### Monitoring - Zmon
[Zmon](https://github.com/zalando/zmon) is an Open Source monitoring
tool.  Howler can manage Zmon entities that need to be updated if an
instance is destroyed, scheduled somewhere else or newly created.

![Zmon integration](https://raw.githubusercontent.com/zalando-techmonkeys/howler/master/docs/monitoring.png)

#### Secret Distribution - vault
[vault](https://github.com/hashicorp/vault) is a tool for managing
secrets.

Howler can help you to provide a new deployed instance with it's
secrets maintained by [vault](https://github.com/hashicorp/vault).

This backend is currently under development.

The idea is a bit more completed than the other backends. It uses
vault's cubbyhole approach called
[coprocess](https://www.hashicorp.com/blog/vault-cubbyhole-principles.html).
This means howler will provide you with secrets, only if the requester
(in most cases your init script) can provide the shared cubbyhole token.

The picture shows the steps of secret distribution and the
responsibilities of howler and other components.
![Secret distribution integration](https://raw.githubusercontent.com/zalando-techmonkeys/howler/master/docs/secrets-distribution-vault.png)

##### Requirements Vault
- You need to have a running and unsealed vault
- You need to have "secret" and "cubbyhole" vault backends.

##### Howler-vault backend
Howler-vault has a rootToken to create policies for applications,
create cubbyhole-tokens, secret-tokens, read/write into cubbyhole.

A goroutine per deployment-instance will:

1. Create policies for new applications and write them to vault
1. Create cubbyhole-token
1. Create secret-token with policy
1. Authenticate with cubbyhole token (shared) to vault
1. Write secret-token into cubbyhole/sharedsecret. Cubbyhole stores
   secrets per token, that means same path for everyone is ok.
1. Create an https endpoint for the upcoming docker-host.
1. Wait for the newly deployed docker-host and respond with it's
   cubbyhole token (Requester may be an init script within docker)
1. terminate goroutine

##### Init Script
The init script got now a cubbyhole token which it will use to
authenticate to vault. These are the steps that you have to do within
the init script:

1. Authenticate with cubbyhole-token to vault
1. Read the secret token from cubbyhole/sharedsecret
1. Authenticate with secret-token to vault
1. Read application secrets from secret/&lt;marathon-appID&gt;

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
- write unit tests: this [talk](https://speakerdeck.com/mitchellh/advanced-testing-with-go) can help to do this.
- implement example init script to show a working secret distribution
  with vault backend
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
