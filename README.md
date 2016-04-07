# Howler

[![Go Report Card](https://goreportcard.com/badge/zalando-techmonkeys/howler)](https://goreportcard.com/report/zalando-techmonkeys/howler)
[![Build Status](https://travis-ci.org/zalando-techmonkeys/howler.svg?branch=master)](https://travis-ci.org/zalando-techmonkeys/howler)
[![Coverage Status](https://coveralls.io/repos/zalando-techmonkeys/howler/badge.svg?branch=master&service=github)](https://coveralls.io/github/zalando-techmonkeys/howler?branch=master)
[![License](http://img.shields.io/badge/license-MIT-yellow.svg?style=flat)](https://raw.githubusercontent.com/zalando-techmonkeys/howler/master/LICENSE)

Howler is a service that listens to events posted in the [Marathon](https://github.com/mesosphere/marathon) Event Bus, processes them in arbitrary backends, and distributes them in an event-driven, flexible way. It enables you to integrate Marathon into your infrastructure via a single interface — freeing you up from having to change all of your configurations across your entire system. Using build flags, it makes enabling different backends possible.

###Project Context and Features
Different cluster managers offer different features. Unfortunately, some of them don't support getting things to production on a "right-now"/instantaneous basis. 

Furthermore, you might need detailed information related to alerting and monitoring your endpoints, adding and removing load-balancer members, and/or secret distribution. In that case, implementing an event-driven approach that allows you to dynamically adjust your components is generally a good idea. 

Howler enables you to adopt this event-driven approach. Instead of rebuilding the "world," you just add and delete single resources.

#### A Note on Using Bamboo
In the [Mesos](http://mesos.apache.org/) and [Marathon](https://github.com/mesosphere/marathon) stack, at
least two similar projects — [Bamboo](https://github.com/QubitProducts/bamboo) and [Marathon-LB](https://github.com/mesosphere/marathon-lb) — generate a complete new HAProxy configuration, check the
configuration, and reload the HAProxy. While testing our setup with [Bamboo](https://github.com/QubitProducts/bamboo), we realized that we wanted to have a much more dynamic tool that a) distributed events to backends, and b) that could react to them in a much more dynamic and stable way. So we created Howler. 

###Basic Requirements

To start using Howler, you'll need:
- a running [Mesos](http://mesos.apache.org/)-[Marathon](https://github.com/mesosphere/marathon)
setup 
- [Go](https://golang.org/)

### Installation

Once you've installed Go and a GOPATH environment variable, run this:

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

### Usage

####Configuring Marathon to Send Events to Howler
The URL of the endpoint should target Howler. Configure Howler and Marathon accordingly:

    [marathon-host]% cat /etc/marathon/conf/event_subscriber
    http_callback
    [marathon-host]% cat /etc/marathon/conf/http_endpoints
    http://my-howler-host:12345/events

###Backends
[Backends](./backend) are components that you can plug in to process events coming from Marathon, and to implement particular actions based on these events. To be pluggable, a backend *must* implement the [backend interface](./backend/backend.go). Howler's usefulness depends on backends.  

Howler users will vary in their backend-related needs. One approach is to mix different backends; another is to implement a greater number of backends. 

To allow composability, one can choose a "compilation over configuration" approach:
- define/write a backend configuration similar to [this one](backendconfig/zalando.go) 
- Include the appropriate [build tag](https://golang.org/pkg/go/build/) to select your configuration: ```godep go install -tags YOUR_TAG github.com/zalando-techmonkeys/howler/...```
- Select the configuration at compile time 

The following Marathon event types are currently dispatched and processed by the respective methods:

- [api_post_event](http://mesosphere.github.io/marathon/docs/event-bus.html#api-request), handled by `HandleCreate()`
- [status_update_event](http://mesosphere.github.io/marathon/docs/event-bus.html#status-update), handled by `HandleUpdate()`
- [app_terminated_event](https://github.com/mesosphere/marathon/issues/1530), handled by `HandleDestroy()`

Have a look at the [dummy backend](backend/dummy.go) for an example.

####Load Balancing
[F5](https://f5.com/) produces hardware load balancers like [LTM Big-IP](https://f5.com/products/modules/local-traffic-manager) and [GTM](https://f5.com/products/modules/global-traffic-manager), a smart DNS server.

This diagram shows how to combine LTM Big-IP and GTM DNS server integration with [baboon-proxy](https://github.com/zalando-techmonkeys/baboon-proxy) (currently the most feature-complete F5 RESTful API available) and
[chimp](https://github.com/zalando-techmonkeys/chimp), a PAAS-style deployment tool:

![LTM/GTM integration](https://raw.githubusercontent.com/zalando-techmonkeys/howler/master/docs/Loadbalancer_ltm_gtm_integration.png)

####Secret Distribution with Vault
[Vault](https://github.com/hashicorp/vault) is a tool for managing secrets. With Howler, you can create a new deployed instance with its secrets maintained by [vault](https://github.com/hashicorp/vault). 

Howler's backend for Vault is still under development, but farther along than [the others underway](https://github.com/zalando-techmonkeys/howler/tree/master/backend). It uses [coprocess](https://www.hashicorp.com/blog/vault-cubbyhole-principles.html), Vault's cubbyhole approach.
This means that Howler will provide you with secrets, but only if the requester (in most cases, your init script) can provide the shared cubbyhole token.

This diagram illustrates the steps of secret distribution, and the roles of Howler and other components:
![Secret distribution integration](https://raw.githubusercontent.com/zalando-techmonkeys/howler/master/docs/secrets-distribution-vault.png)

#####Requirements
To use Vault with Howler, you need:
- a running and unsealed vault
- "secret" and "cubbyhole" Vault backends

#####Howler-Vault Backend
Howler-vault includes a rootToken to create policies for applications, cubbyhole tokens, and secret-tokens, and to read/write into cubbyhole.

A goroutine per deployment instance will:
- create policies for new applications and write them to Vault
- create cubbyhole tokens
- create secret-tokens with policies
- authenticate with cubbyhole tokens (shared) to Vault
- write secret-tokens into cubbyhole/sharedsecret. Cubbyhole stores secrets per token, so the same path for everyone is ok
- creates an HTTPS endpoint for the upcoming Docker host
- waits for the newly deployed Docker host and responds with its cubbyhole token. The requester may be an init script within Docker)
- terminates goroutine

#####Init Script
The init script receives a cubbyhole token that it can use to authenticate to Vault. Here are the required steps you must take within the init script:

1. Authenticate with the cubbyhole token to Vault
1. Read the secret token from cubbyhole/sharedsecret
1. Authenticate with secret-token to Vault
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

### Contributing/TODO
We welcome contributions from the community; just submit a pull request. To help you get started, here are some items that we'd love help with:

- Adding Kubernetes (another cluster manager) as an event source
- Writing unit tests. This [talk](https://speakerdeck.com/mitchellh/advanced-testing-with-go) can help you get started.
- Implementing an example init script to show a working secret distribution with Vault backend
- Cleaning up the code base.

Please use GitHub Issues as a starting point for contributions, new ideas or bugreports.

### Contact

* E-Mail: team-techmonkeys@zalando.de
* IRC on freenode: #zalando-techmonkeys

### Contributors

Thanks to:

- &lt;your name&gt;

### License

See [LICENSE](LICENSE) file.
