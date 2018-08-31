backvendor
==========

This command inspects a Go source tree with vendored packages and attempts to work out the versions of the packages which are vendored.

It does this by comparing file hashes of the vendored packages with those from the upstream repositories.

If no semantic version tag matches but a commit is found that matches, a pseudo-version is generated.

Example output:

```
$ backvendor go/src/github.com/docker/distribution
github.com/opencontainers/image-spec@87998cd070d9e7a2c79f8b153a26bea0425582e5 =v1.0.0 ~v1.0.0
golang.org/x/oauth2@2897dcade18a126645f1368de827f1e613a60049 ~v0.0.0-20160323192119-2897dcade18a
rsc.io/letsencrypt ?
...
```

In this example,

* github.com/opencontainers/image-spec had a matching semantic version tag
* golang.org/x/oath2 had a matching commit only
* rsc.io/letsencrypt had no matching commit (because the vendored source was from a fork)
