package v1alpha1

// Condition types reported in RHOAIBuildStatus.Conditions. Typical
// reconciliation states; Readiness means the selected catalog and the
// installed RHOAI operator are healthy, and does not certify workload
// behavior.
const (
	// ConditionResolving: resolving the requested stream, or validating
	// a pinned digest, to immutable FBC and bundle digests.
	ConditionResolving = "Resolving"

	// ConditionInstalling: creating the installation namespace, registry
	// credential wiring, CatalogSource, OperatorGroup, and Subscription,
	// and driving InstallPlan approval.
	ConditionInstalling = "Installing"

	// ConditionUpdating: an update to a newer build is in progress.
	ConditionUpdating = "Updating"

	// ConditionReady: the selected catalog is serving and the installed
	// RHOAI operator is healthy.
	ConditionReady = "Ready"

	// ConditionBlocked: the request cannot proceed; the condition
	// message explains the reason and the action needed.
	ConditionBlocked = "Blocked"
)

// Phases mirrored in RHOAIBuildStatus.Phase.
const (
	PhaseResolving  = "Resolving"
	PhaseInstalling = "Installing"
	PhaseUpdating   = "Updating"
	PhaseReady      = "Ready"
	PhaseBlocked    = "Blocked"
)
