package request

// UpdateBusinessSettingsRequest allows admins to configure statutory and workflow settings.
type UpdateBusinessSettingsRequest struct {
	// Statutory deduction toggles
	PensionEnabled *bool `json:"pension_enabled,omitempty"`
	NHFEnabled     *bool `json:"nhf_enabled,omitempty"`
	NSITFEnabled   *bool `json:"nsitf_enabled,omitempty"`
	PAYEEnabled    *bool `json:"paye_enabled,omitempty"`

	// Payroll workflow configuration
	PayrollRequiresApproval *bool `json:"payroll_requires_approval,omitempty"`
	PayrollAutoProcess      *bool `json:"payroll_auto_process,omitempty"`
}
