backvendor
==========

This command inspects a Go source tree with vendored packages and attempts to work out the versions of the packages which are vendored.

It does this by comparing file hashes of the vendored packages with those from the upstream repositories.

If no semantic version tag matches but a commit is found that matches, a pseudo-version is generated.

Example output:

```
$ backvendor ~/go/src/github.com/docker/distribution
github.com/bugsnag/panicwrap: {1.1.0 aceac81c6e2f55f23844821679a0553b545e91df 1.1.0}
github.com/denverdino/aliyungo: { b97a5df887ed8cece2f2a166701a2eeff7fbf29c v0.0.0-20161212112416-b97a5df887ed}
rsc.io/letsencrypt: ?
...
```

In this example,

* bugsnag/panicwrap had a matching semantic version tag
* denverdino/aliyungo had a matching commit only
* rsc.io/letsencrypt had no matching commit (because the vendored source was from a fork)
