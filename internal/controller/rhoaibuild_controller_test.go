package controller

import (
	"strings"
	"testing"

	rhoaiv1alpha1 "github.com/kylape/rhoai-build-operator/api/v1alpha1"
)

func digest(char byte) string {
	return "sha256:" + strings.Repeat(string(char), 64)
}

func TestValidateSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    rhoaiv1alpha1.RHOAIBuildSpec
		wantErr bool
	}{
		{
			name: "nightly stream",
			spec: rhoaiv1alpha1.RHOAIBuildSpec{Version: "3.6", BuildType: rhoaiv1alpha1.BuildTypeNightly},
		},
		{
			name: "pinned build",
			spec: rhoaiv1alpha1.RHOAIBuildSpec{Version: "3.6", FBCDigest: digest('a'), BundleDigest: digest('b')},
		},
		{
			name: "ga with exact version",
			spec: rhoaiv1alpha1.RHOAIBuildSpec{Version: "3.6.2", BuildType: rhoaiv1alpha1.BuildTypeGA},
		},
		{
			name:    "automatic updates with pin",
			spec:    rhoaiv1alpha1.RHOAIBuildSpec{Version: "3.6", AutomaticUpdates: true, FBCDigest: digest('a')},
			wantErr: true,
		},
		{
			name:    "ga with pin",
			spec:    rhoaiv1alpha1.RHOAIBuildSpec{Version: "3.6.2", BuildType: rhoaiv1alpha1.BuildTypeGA, FBCDigest: digest('a')},
			wantErr: true,
		},
		{
			name:    "same digests",
			spec:    rhoaiv1alpha1.RHOAIBuildSpec{Version: "3.6", FBCDigest: digest('a'), BundleDigest: digest('a')},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			build := &rhoaiv1alpha1.RHOAIBuild{Spec: tt.spec}
			err := validateSpec(build)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
