backvendor
==========

This command inspects a Go source tree with vendored packages and attempts to work out the versions of the packages which are vendored.

It does this by comparing file hashes of the vendored packages with those from the upstream repositories.

If no semantic version tag matches but a commit is found that matches, a pseudo-version is generated.

Example output:

```
$ backvendor go/src/github.com/docker/distribution
github.com/opencontainers/image-spec@87998cd070d9e7a2c79f8b153a26bea0425582e5 =v1.0.0 ~v1.0.0
github.com/ncw/swift@b964f2ca856aac39885e258ad25aec08d5f64ee6 ~1.0.25-0.20160617142549-b964f2ca856a
golang.org/x/oauth2@2897dcade18a126645f1368de827f1e613a60049 ~v0.0.0-20160323192119-2897dcade18a
rsc.io/letsencrypt ?
...
```

In this example,

* github.com/opencontainers/image-spec had a matching semantic version tag
* github.com/ncw/swift's matching commit had tag 1.0.24 reachable (so the next patch version would be 1.0.25)
* golang.org/x/oath2 had a matching commit but no reachable semantic version tag
* rsc.io/letsencrypt had no matching commit (because the vendored source was from a fork)
