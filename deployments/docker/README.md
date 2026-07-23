# deployments/docker

The canonical image build is the `Dockerfile` at the repository root — it
produces one multi-stage image containing all three binaries (`api`,
`worker`, `migrate`), selected via the container's `command`. `docker
compose up` at the repo root uses it directly.

This folder is reserved for deployment-specific image variants that
shouldn't live at the root (e.g. a hardened distroless production image, or
a Kubernetes-specific entrypoint wrapper). None exist yet in this delivery.
