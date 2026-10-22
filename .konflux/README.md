# Hermetic RPM dependencies

The runtime keeps the pinned UBI micro base and adds OpenSSL, its FIPS provider,
and crypto policies. Both Konflux pipelines prefetch Go modules and the RPMs
listed in `rpms.lock.yaml` before building with network access disabled.

Konflux mounts the local RPM repository files at `/etc/yum.repos.d`. The
Containerfile uses that directory explicitly because packages are installed
into `/runtime-root`. It imports the Red Hat signing key into that root before
installation. Local builds use the builder image's online UBI repository at
the same path.

`rpms.in.yaml` resolves packages against the `runtime-base` stage in the
Containerfile. It covers x86_64 and aarch64 and includes source RPMs for the
Konflux source image. Do not resolve against the Go builder: it already has
OpenSSL, so the lockfile could omit packages needed by the micro runtime.

## Update the lockfile

Install [rpm-lockfile-prototype](https://github.com/konflux-ci/rpm-lockfile-prototype)
with its DNF and Skopeo dependencies, then run from the repository root:

```sh
make konflux-rpm-lock
```

Regenerate the lockfile when changing the runtime base or package list. Include
both `rpms.in.yaml` and `rpms.lock.yaml` in the change. MintMaker can also update
this standard lockfile automatically.

## Check an offline build locally

Install [Hermeto](https://github.com/hermetoproject/hermeto) and its dependencies,
then prefetch the same inputs as Konflux:

```sh
hermeto fetch-deps --source . --output /tmp/alerts-adapter-prefetch \
  '[{"type":"gomod","path":"."},{"type":"rpm","path":".konflux"}]'
hermeto inject-files /tmp/alerts-adapter-prefetch \
  --for-output-dir /tmp/alerts-adapter-prefetch
```

Build on the host architecture with networking and layer reuse disabled:

```sh
podman build --no-cache --network=none \
  --volume /tmp/alerts-adapter-prefetch:/tmp/alerts-adapter-prefetch:ro,z \
  --volume "/tmp/alerts-adapter-prefetch/deps/rpm/$(uname -m)/repos.d:/etc/yum.repos.d:ro,z" \
  --env GOPROXY=file:///tmp/alerts-adapter-prefetch/deps/gomod/pkg/mod/cache/download \
  --env GOSUMDB=off \
  --env GOCACHE=/tmp/go-build \
  --env GOMODCACHE=/tmp/go-mod-cache \
  --env GOPATH=/tmp/go-path \
  -f Containerfile -t localhost/alerts-adapter:hermetic .
podman run --rm --network=none --read-only \
  -e GOLANG_FIPS=1 localhost/alerts-adapter:hermetic --help
```

The temporary Go caches stay writable inside the build container; the
prefetched files are mounted read-only. Hermeto checks the downloaded modules
before the build. `GOSUMDB=off` avoids an online checksum lookup during the
offline build. A full FIPS cluster E2E run is still needed after this startup
check.

See the [Konflux prefetch documentation](https://konflux-ci.dev/docs/building/prefetching-dependencies/)
for the pipeline setup.
