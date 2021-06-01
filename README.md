# syncrd

Synchronize instances of *Custom Resource Definition* between the source and destination cluster.

## Binary & Run Binary

```bash
# Build the binary. Here the default values are also given.
# The definition of `ClusterwideNetworkPolicy` can be found be in `github.com/metal-stack/firewall-controller/api/v1`.
# Modify these four variables to suit your repository.
make \
REPO_URL=github.com/metal-stack/firewall-controller \
REPO_VERSION=latest \
SUB_PATH=api/v1 \
CRD_KIND=ClusterwideNetworkPolicy

# Run the binary. Make sure the valid paths to the source and destination clusters are given.
bin/syncrd --source /tmp/source --dest /tmp/dest
```

## Build & Run Docker Image

```bash
# Build the image. As above, the default values are also given.
make docker-build \
REPO_URL=github.com/metal-stack/firewall-controller \
REPO_VERSION=latest \
SUB_PATH=api/v1 \
CRD_KIND=ClusterwideNetworkPolicy

# Run the image. As above, make sure `/tmp/source` and `/tmp/dest` are valid paths.
docker run -v /tmp/source:/source:ro -v /tmp/dest:/dest:ro --network host ghcr.io/metal-stack/syncrd:latest
```
