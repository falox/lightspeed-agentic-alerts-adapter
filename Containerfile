FROM registry.access.redhat.com/ubi9-micro:9.8-1791895366@sha256:5be877a2a280ae36eb9ab80ca4efb28bbf77c57abbba38137cdc37a03b8465e6 AS runtime-base

FROM registry.redhat.io/ubi9/go-toolset:9.8-1790174511@sha256:0a4666f7a4eb0644c97a73cba198eb268691b270d97831822689e7a2088f87be AS builder

COPY go.mod go.sum* ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=1 GOOS=linux GOEXPERIMENT=strictfipsruntime go build -tags strictfipsruntime -ldflags="-s -w" -o ./alerts-adapter ./cmd/alerts-adapter

FROM builder AS runtime-deps

USER 0
COPY --from=runtime-base / /runtime-root/
# Install OpenSSL and its FIPS provider into the micro filesystem, including RPM metadata.
# Konflux mounts its prefetched repositories at /etc/yum.repos.d.
RUN rpmkeys --root=/runtime-root --import /etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release \
    && dnf install -y --installroot=/runtime-root --releasever=9 \
      --disablerepo='*' --enablerepo=ubi-9-baseos-rpms \
      --setopt=reposdir=/etc/yum.repos.d --setopt=install_weak_deps=false --nodocs \
      openssl-libs openssl-fips-provider crypto-policies \
    && dnf clean all --installroot=/runtime-root

FROM runtime-base

COPY --from=runtime-deps /runtime-root/ /

LABEL com.redhat.component="lightspeed-agentic-alerts-adapter" \
      name="openshift-lightspeed/lightspeed-agentic-alerts-adapter-rhel9" \
      version="0.1.0" \
      release="1" \
      summary="Lightspeed Agentic Alerts Adapter" \
      description="Polls OpenShift AlertManager for firing alerts and creates AgenticRun CRs to trigger automated remediation via the Lightspeed Agentic operator." \
      io.k8s.description="Polls OpenShift AlertManager for firing alerts and creates AgenticRun CRs to trigger automated remediation via the Lightspeed Agentic operator." \
      url="https://github.com/openshift/lightspeed-agentic-alerts-adapter" \
      vendor="Red Hat, Inc." \
      distribution-scope="public" \
      cpe="cpe:/a:redhat:openshift_lightspeed:2::el9"

COPY LICENSE /licenses/LICENSE
COPY --from=builder /opt/app-root/src/alerts-adapter /usr/local/bin/alerts-adapter

USER 1001

ENTRYPOINT ["alerts-adapter"]
