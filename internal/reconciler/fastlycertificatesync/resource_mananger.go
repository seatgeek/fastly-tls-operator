package fastlycertificatesync

import (
	rm "github.com/seatgeek/k8s-reconciler-generic/pkg/resourcemanager"
)

var ResourceManager = rm.ResourceManager[*Context]{}
