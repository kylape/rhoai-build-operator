# RHOAI build operator

Status: Design accepted; implementation starting. See README.md for
current status.

Updated: 2026-11-23.

## User experience

Apply one resource to install the requested RHOAI build. Choose whether the
operator should keep it updated. The controller handles discovery, catalog
publication, initial installation, and subsequent updates.

```yaml
apiVersion: dev.rhoai.openshift.io/v1alpha1
kind: RHOAIBuild
metadata:
  name: default
spec:
  version: "3.6"
  buildType: nightly
  automaticUpdates: true
```

This means: install the latest available 3.6 nightly, then check every night
and update the installation when a new build is available.

Set `automaticUpdates: false` to install the selected build once and keep it.
The controller continues repairing that installation to its recorded desired
state. Changing the requested version or build type requests an update even
when automatic updates are disabled.

Users do not create CatalogSources, Subscriptions, OperatorGroups, InstallPlans,
baselines, or a second refresh CR. Those are controller implementation details.
Enabling automatic updates authorizes the controller to perform compatible
updates without asking for approval for each nightly.

The resource is named `RHOAIBuild`: ci, nightly, ea, and ga are all builds
in Tracer and Konflux vocabulary, and `spec.buildType` selects among them, so
the name stays accurate when a request installs a GA release. The
`dev.rhoai.openshift.io` group carries the dev-tool framing, which holds even
for GA requests — production installs belong to standard GitOps and console
flows, not this operator. Earlier drafts used `RHOAICatalog`, which described
the mechanism rather than the subject and became misleading once GA
delegation meant no custom catalog is involved at all.

## Use cases

* **E2E and CI.** Stand up ephemeral clusters on the newest build of a
  stream, gate test start on readiness, destroy the cluster. Uses stream
  selection (`version` plus `buildType`) with `automaticUpdates` off — each
  run installs whatever is newest at cluster creation.
* **Bug reproduction.** Stand up a cluster running exactly the build a bug
  was reported against — a GA patch, a nightly from a particular date, or a
  CI build. Resolve the reported build to its digest through Tracer or the
  build announcements, then pin it with `fbcDigest`. Because status reports
  the installed digest pair for every install, bug reports can carry exact
  build identity instead of version strings.
* **Customer support scenarios.** Mirror a customer's RHOAI version to
  reproduce, troubleshoot, or verify a fix. Customers mostly run GA
  releases, which is why `buildType: ga` delegates to the standard catalog;
  EA and nightly customers are handled by digest pins. Verifying a fix bumps
  the same cluster to a newer build of the stream.
* **Persistent development environments.** Keep a long-lived cluster current
  on a stream with `automaticUpdates: true` — the case that needs the
  same-identity refresh machinery.

## API and defaults

Propose one cluster-scoped resource, initially limited to `metadata.name:
default`, representing the cluster's managed RHOAI installation. Cluster scope
lets users apply the CR before the RHOAI installation namespace exists and
matches the cluster-wide resources the installation affects.

| Field | Meaning |
| --- | --- |
| `version` | Required release stream, such as `3.6`; exact patch versions allowed for GA |
| `buildType` | `ci`, `nightly`, `ea`, or `ga`; defaults to `ci` |
| `automaticUpdates` | Follow newer builds in the requested stream; defaults to `false` |
| `pullSecretRef` | Optional reference to registry credentials in this operator's namespace |
| `fbcDigest` | Optional immutable FBC digest selecting one exact build; see Pinned builds |
| `bundleDigest` | Optional operator bundle digest, validated against the pinned FBC |

A cluster with sufficient existing registry credentials needs only the example
above. Otherwise, administrators supply a pull Secret and reference it. The
controller wires required registry access into the resources it creates.
No manual RHOAI installation is a prerequisite.

The first target is OpenShift with its existing OLM v0 installation. This
operator itself must be installed, but it does not bootstrap OpenShift or OLM.
Installing RHOAI here means installing its operator and required installation
resources. Selecting optional RHOAI components and user workloads remains
outside this API's initial scope.

## Pinned builds

`spec.fbcDigest` requests one exact build by the immutable digest of its
file-based catalog. A pinned request bypasses stream resolution: the
controller reads the package, channel, and bundle reference from that FBC and
installs it. `version` remains valid alongside a pin and is checked against
the FBC's own labels; a mismatch blocks with an actionable message rather
than installing something unexpected.

`spec.bundleDigest` is an optional cross-check. The controller always takes
the bundle reference from the pinned FBC, so a matching `bundleDigest` adds
no resolution step — it verifies that the FBC selects the bundle the
requester expected, catching stale or mistyped pins before any cluster
change.

* `automaticUpdates: true` together with `fbcDigest` is rejected: following
  a stream and pinning one build are contradictory requests.
* A bare digest (`sha256:...`) is the normal form; the controller supplies
  the registry and repository, keeping locations out of user specs. A fully
  qualified image reference is accepted as an escape hatch for unusual
  registry topologies.
* Changing or clearing `fbcDigest` is an explicit update request, with the
  same transition validation as a version change. Clearing the pin returns
  selection to the requested stream.

Pinning is the reproducibility mechanism across the dev use cases: a test
run, a bug reproduction, or a support scenario requests a digest, and status
confirms the installed FBC and bundle digests match the request. The
environment is then identified by a digest pair that any later cluster can
re-request — and because status reports the pair for stream installs too,
bug reports can carry the exact identity of the build that exhibited the
problem.

## Deployment model

GitOps applies this operator's own manifests — a stable target — and then one
CR per cluster; an Argo CD ApplicationSet can fan both out to labeled
clusters. The GitOps repository never encodes pre-release catalog details;
those live in this operator's code. When pre-release publishing changes, the
fix is an operator upgrade, not a change to GitOps repos or pipelines.

The operator's complexity is the pre-release surface: ci, nightly, and EA
discovery (Tracer tag conventions, `quay.io/rhoai` FBC fragments and bundles,
private registry authentication). GA builds are available in the standard
`redhat-operators` catalog with normal upgrade edges; a `buildType: ga`
request delegates to that catalog rather than duplicating it. Delegation
keeps one API and one readiness contract across every build type, which is
what bug reproduction and support scenarios depend on — most customers run
GA releases.

Non-OCP clusters are outside this operator's scope. The odh-gitops
`rhai-on-xks-chart` installs the inference stack directly without OLM and is
the reference path for those clusters.

## What the controller does

1. Resolve the requested version and build type to an immutable FBC digest.
2. Read its package and channel, and resolve the selected bundle to a digest.
3. Create the installation namespace, registry credential wiring, CatalogSource,
   OperatorGroup, and Subscription as needed.
4. Drive initial OLM installation, including approval of the InstallPlan that
   matches the selected artifacts and expected installation resources.
5. Wait for a serving catalog and a healthy installed operator, then record the
   installed digests and identity.
6. If automatic updates are enabled, check daily for a newer build and choose
   the appropriate update mechanism internally.

Use Tracer's version/tag conventions, with registry clients rather than shell
output parsing. Read the bundle reference from the resolved FBC; independently
resolving two mutable tags could select builds published at different times.
Pin all selected artifacts before starting an installation or update.

The controller chooses the actual catalog channel from validated package data.
An ambiguous or unsupported selection produces an actionable status message,
rather than guessing. It also validates the target platform and registry access.

## Two update mechanisms, one API

A user requests the desired release and update behavior. They do not choose
whether the controller uses OLM upgrades or an in-place nightly refresh.

**Normal OLM upgrade:** when the selected build has a distinct CSV identity and
an applicable upgrade path, update the catalog and drive the matching OLM
InstallPlan. After successful installation, record the new installed identity.

**Same-identity nightly refresh:** when a new nightly bundle reuses the installed
CSV identity, update the existing CSV's permitted runtime fields, bundle CRDs,
and supported accompanying resources. OLM continues managing the operator
Deployment, installation RBAC, webhooks, and certificates derived from the CSV.

The second mechanism follows the approach proposed in
[Stratus PR #1312](https://github.com/neuralmagic/stratus/pull/1312). The
nightly upgrade-path investigation (workspace scratchpad)
records why it is needed: different daily bundles can share
`rhods-operator.3.6.0`, so updating the catalog alone supplies no new OLM upgrade
edge between those builds.

For in-place refreshes, the controller captures the original installation
identity automatically after a successful initial OLM installation. It stores
that identity and pinned artifact references as internal durable state, and
checks them before each update. A successful controller-driven OLM upgrade
establishes the next identity automatically. There is no user baseline file or
capture command.

Normal OLM updates and in-place refreshes are serialized. The managed
Subscription uses Manual InstallPlan approval so this controller can authorize
only plans matching its selected target; users do not approve each plan.
An unexpected plan, unrelated resources, or a concurrent external installation
change blocks the update rather than being approved indiscriminately.

Automatic updates do not imply that every possible transition can be performed.
If a candidate requires an unsupported migration or lacks an applicable upgrade
path, keep the current installation and explain the blocked transition in status.
Do not uninstall and reinstall as a fallback.

## Update policy

With automatic updates disabled, resolve and install once for the requested
version/build selection, then retain those exact artifacts across restarts.
Credential rotation and metadata changes do not select a newer build.
An explicit change to version or build type requests a new selection.

With automatic updates enabled, check the requested stream daily. Install a
new candidate automatically after artifact and transition validation. Image
changes and supported resource changes follow this same policy; the design
does not introduce a separate human approval step for each build.

Resolve a target once and finish reconciling that target before considering
another. If a tag moves during a rollout, it cannot change the in-flight update.
Disabling automatic updates prevents the next selection; an already started
update continues toward its pinned target so it is not abandoned halfway.

Changing the requested version does not authorize arbitrary downgrades. Validate
the requested transition and report it as blocked when no supported mechanism
exists. Pinned builds are the explicit reproducible-selection mechanism; they
do not expose OLM mechanics in the API.

## Ownership and internal state

| Resource | Management |
| --- | --- |
| CatalogSource, OperatorGroup, Subscription, selected InstallPlan approval | This controller |
| Operator Deployment, installation RBAC, webhook lifecycle | OLM |
| CSV runtime fields during same-identity refresh | This controller, using a defined field allowlist |
| Bundle CRDs and supported accompanying resources during refresh | This controller, using limited field ownership |
| Operand configuration and workloads | Existing operators and users |

Use narrow field ownership; do not force conflicts with OLM or another manager.
Preserve CSV identity and replacement metadata during an in-place refresh.
Exclude generated certificate material and live conversion configuration from
ordinary refresh writes. Changes requiring a conversion migration are blocked
until the controller supports that transition.

Persist the installed identity, selected artifacts, and in-flight update state
in controller-managed storage independent of disposable status. Restart recovery
must resume the recorded target and must not reinterpret a moving tag as the
previous installation. Do not add another user-facing CRD for this state.

Initially, require a fresh installation or a previous installation managed by
this controller. An unrelated existing RHOAI installation produces a clear
conflict; explicit adoption can be added later. This does not require users to
preinstall RHOAI or prepare a baseline.

## Status and failure behavior

Expose the requested stream or pinned digest, installed version, FBC and bundle
digests, last successful update, next scheduled check, and conditions
explaining progress.
Typical states are `Resolving`, `Installing`, `Updating`, `Ready`, and `Blocked`.
Readiness means the selected catalog and RHOAI operator are healthy; it does not
certify user workload behavior.

Registry failures leave the existing installation in place and retry with
backoff. Failed validation changes no installation resources. After partial
application, resume the pinned target; Kubernetes writes are not transactional.
A rollout failure stops selection of further builds and reports the failure.
Recovery of a failed rollout is explicit, rather than an automatic downgrade
that might be incompatible with changed APIs or data.

Retain the installation when the CR is deleted by default. Stop this
controller's reconciliation, leave the Subscription manually approved, and
preserve workloads and CRDs. Deletion is not an implicit RHOAI uninstall.
A deliberate uninstall policy can be designed separately.

## Implementation and verification

Implement one Go operator with one public CRD and internal strategies for initial
installation, normal OLM upgrade, and same-identity refresh. Registry credentials
are Kubernetes Secrets; GitHub, GitLab, Bitwarden, and Konflux logins are not
runtime dependencies.

Build and test in this order:

1. Apply the CR to an OpenShift cluster without RHOAI and reach a serving FBC
   and healthy installed operator without manual installation steps.
2. Demonstrate that automatic updates disabled retains the exact installed build
   across new registry tags, credential rotation, and controller restarts.
3. Demonstrate that a pinned `fbcDigest` installs exactly that build, that a
   `bundleDigest` mismatch blocks before any cluster change, and that
   `automaticUpdates: true` is rejected alongside a pin.
4. Demonstrate automatic updates enabled selects and installs a new nightly
   sharing the previous CSV identity, including recovery across a restart.
5. Demonstrate a normal OLM upgrade establishes the next internal identity
   without user recapture or another CR.
6. Exercise registry errors, unsupported transitions, ownership conflicts,
   partial failures, and disabling updates during an active rollout.

The live pilot must verify OLM coexistence and certificate rotation, alongside
representative RHOAI workload checks. In-place nightly refresh remains an
experimental mechanism until that evidence exists. The intended delivered
experience still includes installation and updates through the single CR;
implementation stages do not become additional setup steps for users.
