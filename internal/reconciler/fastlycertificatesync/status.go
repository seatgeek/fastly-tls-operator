package fastlycertificatesync

import (
	"github.com/seatgeek/k8s-reconciler-generic/apiobjects"
	"github.com/seatgeek/k8s-reconciler-generic/pkg/genrec"
)

func (l *Logic) FillStatus(c *Context, obs genrec.Resources, ss apiobjects.SubjectStatus) error {
	res := &(c.Subject.Status)
	res.SubjectStatus = ss

	// TODO: Add status conditions here when implementing actual reconciliation logic

	return nil
}
