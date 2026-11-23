package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	rhoaiv1alpha1 "github.com/kylape/rhoai-build-operator/api/v1alpha1"
)

// RHOAIBuildReconciler reconciles a RHOAIBuild object.
type RHOAIBuildReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// The resource represents the cluster's single managed installation;
// keep its permissions limited to the OLM objects it owns and observes.

// +kubebuilder:rbac:groups=dev.rhoai.openshift.io,resources=rhoaibuilds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dev.rhoai.openshift.io,resources=rhoaibuilds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dev.rhoai.openshift.io,resources=rhoaibuilds/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// +kubebuilder:rbac:groups=operators.coreos.com,resources=catalogsources,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=operators.coreos.com,resources=operatorgroups,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=operators.coreos.com,resources=installplans,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operators.coreos.com,resources=clusterserviceversions,verbs=get;list;watch

// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch

// Reconcile drives the RHOAIBuild phase machine:
//
//  1. Resolving: resolve the requested stream (or validate the pinned
//     digest) against quay.io/rhoai using registry clients and Tracer's
//     tag conventions, producing pinned FBC and bundle digests.
//  2. Installing: ensure the installation namespace, registry credential
//     wiring, CatalogSource, OperatorGroup, and Subscription; approve
//     only the InstallPlan matching the selected artifacts.
//  3. Ready: wait for a serving catalog and a healthy installed
//     operator, then record the installed identity.
//  4. Updating (automaticUpdates): check daily for a newer build and
//     choose a normal OLM upgrade or a same-identity refresh.
//
// The full contract is in docs/design.md.
func (r *RHOAIBuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var build rhoaiv1alpha1.RHOAIBuild
	if err := r.Get(ctx, req.NamespacedName, &build); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// The resource represents the cluster's single managed installation.
	if build.Name != "default" {
		return ctrl.Result{}, r.block(ctx, &build, "UnsupportedName",
			fmt.Sprintf("only metadata.name %q is supported in this release, got %q", "default", build.Name))
	}

	// Validate the request before touching anything.
	if err := validateSpec(&build); err != nil {
		logger.Info("rejecting RHOAIBuild spec", "error", err)
		return ctrl.Result{}, r.block(ctx, &build, "InvalidSpec", err.Error())
	}

	// TODO(implementation): the phase machine described above. Until
	// resolution and OLM wiring exist, report Blocked so an applied CR
	// gets actionable feedback instead of silence.
	return ctrl.Result{}, r.block(ctx, &build, "NotImplemented",
		"build resolution and OLM installation are not implemented yet; see docs/design.md and README.md")
}

// validateSpec enforces the cross-field rules that CRD validation
// cannot express.
func validateSpec(b *rhoaiv1alpha1.RHOAIBuild) error {
	if b.Spec.AutomaticUpdates && b.Spec.FBCDigest != "" {
		return fmt.Errorf("automaticUpdates cannot follow a stream while fbcDigest pins one build")
	}
	if b.Spec.FBCDigest != "" && b.Spec.FBCDigest == b.Spec.BundleDigest {
		return fmt.Errorf("fbcDigest and bundleDigest must differ")
	}
	if b.Spec.BuildType == rhoaiv1alpha1.BuildTypeGA && b.Spec.FBCDigest != "" {
		return fmt.Errorf("fbcDigest applies to pre-release builds; request an exact GA version instead")
	}
	return nil
}

// block sets every progress condition False, Blocked True, records the
// phase, and persists status. Failed validation changes no installation
// resources.
func (r *RHOAIBuildReconciler) block(ctx context.Context, b *rhoaiv1alpha1.RHOAIBuild, reason, message string) error {
	changed := false
	for _, t := range []string{
		rhoaiv1alpha1.ConditionResolving,
		rhoaiv1alpha1.ConditionInstalling,
		rhoaiv1alpha1.ConditionUpdating,
		rhoaiv1alpha1.ConditionReady,
	} {
		changed = meta.SetStatusCondition(&b.Status.Conditions, metav1.Condition{
			Type:               t,
			Status:             metav1.ConditionFalse,
			Reason:             reason,
			Message:            message,
			ObservedGeneration: b.Generation,
		}) || changed
	}
	changed = meta.SetStatusCondition(&b.Status.Conditions, metav1.Condition{
		Type:               rhoaiv1alpha1.ConditionBlocked,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: b.Generation,
	}) || changed
	if b.Status.Phase != rhoaiv1alpha1.PhaseBlocked {
		b.Status.Phase = rhoaiv1alpha1.PhaseBlocked
		changed = true
	}
	if !changed {
		return nil
	}
	if err := r.Status().Update(ctx, b); err != nil {
		return err
	}
	r.Recorder.Eventf(b, nil, corev1.EventTypeWarning, reason, "Reconcile", message)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RHOAIBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rhoaiv1alpha1.RHOAIBuild{}).
		Named("rhoaibuild").
		Complete(r)
}
