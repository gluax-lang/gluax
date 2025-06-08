package sema

type State struct {
	Label     string               // "SERVER" or "CLIENT"
	Macros    map[string]string    // e.g. {"SERVER": ""}, {"CLIENT": ""}
	RootScope *Scope               // which root scope we attach to in this pass
	Files     map[string]*Analysis // where we store the resulting analyses
}

func NewState(label string) *State {
	return &State{
		Label:     label,
		Macros:    make(map[string]string),
		RootScope: NewScope(nil),
		Files:     make(map[string]*Analysis),
	}
}
