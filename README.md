# Howler

[![Go Report Card](http://goreportcard.com/badge/zalando-techmonkeys/howler)](http://goreportcard.com/report/zalando-techmonkeys/howler)

Howler is an service which is intended to be an endpoint to receive events from the Marathon Event Bus and process them in arbitrary backends.

## Install

```bash
#install godep if you don't have it
go get github.com/tools/godep
#install required dependencies
godep restore
#install to $GOBIN
godep go install -tags zalando github.com/zalando-techmonkeys/howler/...
#for tagging the build, both server and cli:
godep go install  -tags zalando -ldflags "-X main.Buildstamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X main.Githash=`git rev-parse HEAD`"   github.com/zalando-techmonkeys/howler/...
```

This should compile the server binary `howler`, which you can put in `/usr/bin/` and start with this [init-script](howler.init.d).


## Config

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

Configure the "port" on which howler should listen to receive events and configure marathon accordingly:

    cat /etc/marathon/conf/http_endpoints
    http://my-marathon-host:12345/events

## Developement

To be actually useful, there have to be [backends](./backend) which process the events coming from marathon in some way. All of these backends have to fulfill the [`Backend` interface](backend/backend.go). Following Marathon event types are currently dispatched and processed by the respective methods:
- [api_post_event](http://mesosphere.github.io/marathon/docs/event-bus.html#api-request), handled by `HandleCreate()`
- [status_update_event](http://mesosphere.github.io/marathon/docs/event-bus.html#status-update), handled by `HandleUpdate()`
- [app_terminated_event](https://github.com/mesosphere/marathon/issues/1530), handled by `HandleDestroy()`

Have a look on the [dummy backend](backend/dummy.go) for an example.

## Development

* Issues: Just create issues on github
* Enhancements/Bugfixes: Pull requests are welcome
* get in contact: team-techmonkeys@zalando.de

## License

See [LICENSE](LICENSE) file.
