# rhoai-build-operator

A dev-focused operator that installs a requested build of RHOAI on
OpenShift. Apply one `RHOAIBuild` resource naming the release stream and
build type; the controller resolves the build to immutable digests,
wires OLM (CatalogSource, OperatorGroup, Subscription), approves the
matching InstallPlan, and reports readiness through status conditions.

```yaml
apiVersion: dev.rhoai.openshift.io/v1alpha1
kind: RHOAIBuild
metadata:
  name: default
spec:
  version: "3.6"
  buildType: nightly      # ci | nightly | ea | ga
  automaticUpdates: false
```

The operator encodes the complexity of pre-release RHOAI builds — Tracer
tag conventions, `quay.io/rhoai` FBC fragments and bundles, private
registry authentication, channel selection, digest pinning. GA builds
delegate to the standard `redhat-operators` catalog so one API serves
every build type. The `dev.rhoai.openshift.io` group carries the
dev-tool framing; production installs belong to standard GitOps and
console flows.

## Status

Scaffold. The `RHOAIBuild` API, spec validation, and status/conditions
machinery exist; build resolution and OLM installation are next. An
applied CR currently reports `Blocked / NotImplemented` rather than
silently doing nothing.

## Use cases

* **E2E and CI** — ephemeral clusters on the newest build of a stream
* **Bug reproduction** — pin the exact reported build with `fbcDigest`
* **Customer support scenarios** — mirror a customer's version; mostly
  GA, which is why GA delegation exists
* **Persistent dev environments** — `automaticUpdates: true` follows a
  stream

## Development

```
make build       # compile the manager
make test        # unit tests
make generate    # regenerate deepcopy, CRD, and RBAC after API changes
```

Deploy from source on a dev cluster by applying `config/` after
`make generate` (CRD and RBAC are generated), or build the image with
the Dockerfile. Install is plain manifests — no operator catalog needed.

## Implementation order

From the design doc (`docs/design.md`):

1. Stream resolution — registry client for `quay.io/rhoai`, Tracer tag
   conventions, digest pinning
2. OLM wiring — namespace, CatalogSource, OperatorGroup, Subscription,
   Manual InstallPlan approval matched to selected artifacts
3. Readiness — serving catalog + healthy operator conditions; record
   installed identity
4. Pin support — `fbcDigest`/`bundleDigest` validation and install
5. GA delegation — `redhat-operators` Subscription path
6. Automatic updates — daily check; OLM upgrade or same-identity
   nightly refresh (experimental, last)

## Open items

* `quay.io/rhoai` read credential for pre-release resolution — the
  shared "RHOAI Quay ReadOnly bot" pull secret, wired through
  `spec.pullSecretRef`
* Image publishing (`quay.io/klape/rhoai-build-operator`)
* CI onboarding (ARC check + publish)
