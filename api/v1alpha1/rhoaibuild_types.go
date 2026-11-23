package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildType selects which RHOAI build stream a request installs from.
// +kubebuilder:validation:Enum=ci;nightly;ea;ga
type BuildType string

const (
	// BuildTypeCI selects the latest CI build of the stream.
	BuildTypeCI BuildType = "ci"

	// BuildTypeNightly selects the latest nightly build of the stream.
	BuildTypeNightly BuildType = "nightly"

	// BuildTypeEA selects the latest early-access build of the stream.
	BuildTypeEA BuildType = "ea"

	// BuildTypeGA delegates to the standard redhat-operators catalog,
	// where a plain Subscription is the simplest correct install.
	BuildTypeGA BuildType = "ga"
)

// RHOAIBuildSpec defines the desired state of RHOAIBuild: which RHOAI
// build the cluster should run, and whether the operator follows the
// stream or keeps the selected build.
type RHOAIBuildSpec struct {
	// Version is the RHOAI release stream to install, such as "3.6".
	// Exact patch versions are allowed for GA builds, e.g. "3.6.2".
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Version string `json:"version"`

	// BuildType selects the build stream for the requested version:
	// ci, nightly, ea, or ga. Defaults to ci.
	// +kubebuilder:default=ci
	BuildType BuildType `json:"buildType,omitempty"`

	// AutomaticUpdates follows newer builds in the requested stream,
	// checking daily once the installation is Ready. Defaults to false.
	// Incompatible with fbcDigest.
	// +kubebuilder:default=false
	AutomaticUpdates bool `json:"automaticUpdates,omitempty"`

	// PullSecretRef references a Secret in this operator's namespace
	// holding registry credentials for reading pre-release artifacts
	// (quay.io/rhoai FBC fragments and bundles). Needed for ci, nightly,
	// and ea builds unless the cluster already has sufficient registry
	// credentials.
	PullSecretRef *corev1.SecretReference `json:"pullSecretRef,omitempty"`

	// FBCDigest requests one exact build by the immutable digest of its
	// file-based catalog, bypassing stream resolution. A bare sha256
	// digest is the normal form; the controller supplies the registry
	// and repository. Incompatible with automaticUpdates.
	// +kubebuilder:validation:Pattern=`^sha256:[a-f0-9]{64}$`
	FBCDigest string `json:"fbcDigest,omitempty"`

	// BundleDigest is an optional cross-check on FBCDigest: the pinned
	// FBC must select this operator bundle digest, otherwise the request
	// is Blocked before any cluster change.
	// +kubebuilder:validation:Pattern=`^sha256:[a-f0-9]{64}$`
	BundleDigest string `json:"bundleDigest,omitempty"`
}

// RHOAIBuildStatus is the observed state of a RHOAIBuild.
type RHOAIBuildStatus struct {
	// Conditions explain reconciliation progress: Resolving, Installing,
	// Updating, Ready, and Blocked. A stuck build explains itself here.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// Phase is the current reconciliation phase.
	// +optional
	Phase string `json:"phase,omitempty"`

	// InstalledVersion is the RHOAI version reported by the installed
	// operator.
	// +optional
	InstalledVersion string `json:"installedVersion,omitempty"`

	// FBCDigest is the digest of the installed file-based catalog.
	// +optional
	FBCDigest string `json:"fbcDigest,omitempty"`

	// BundleDigest is the digest of the installed operator bundle.
	// +optional
	BundleDigest string `json:"bundleDigest,omitempty"`

	// LastSuccessfulUpdate is when the installation last reached Ready
	// for the current identity.
	// +optional
	LastSuccessfulUpdate *metav1.Time `json:"lastSuccessfulUpdate,omitempty"`

	// NextCheck is when the controller will next look for a newer build
	// when automaticUpdates is enabled.
	// +optional
	NextCheck *metav1.Time `json:"nextCheck,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="BuildType",type=string,JSONPath=`.spec.buildType`
// +kubebuilder:printcolumn:name="Installed",type=string,JSONPath=`.status.installedVersion`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// RHOAIBuild is the cluster's managed RHOAI installation: apply one
// resource to install the requested build, and optionally keep it
// current on its stream. Catalog discovery, OLM wiring, and InstallPlan
// approval are controller implementation details.
type RHOAIBuild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RHOAIBuildSpec   `json:"spec,omitempty"`
	Status RHOAIBuildStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RHOAIBuildList contains a list of RHOAIBuild.
type RHOAIBuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RHOAIBuild `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RHOAIBuild{}, &RHOAIBuildList{})
}
