package fastlycertificatesync

import (
	"github.com/seatgeek/k8s-reconciler-generic/pkg/genrec"
)

func (l *Logic) ReconcileComplete(c *Context, rs genrec.ReconciliationStatus, err error) {

	if c.Subject == nil {
		return
	}

	if rs == genrec.PartitionMismatch { // ignore subjects in other partitions
		return
	}

	switch rs { //nolint:exhaustive
	case genrec.SubjectNotFound, genrec.PartitionMismatch:
		// TODO: delete all relevant gauges for this subject

	case genrec.Okay:
		// TODO: zero out all gauges

		// TODO: set any relevant gauges if observed
	}

	// TODO: report reconciliation errors but ignore transient errors
}
