package rogue

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

func (rogue *Rogue) registerStealthAura() {
	// TODO: Add Stealth spell for use with prepull in APL
	rogue.StealthAura = rogue.RegisterAura(core.Aura{
		Label:    "Stealth",
		ActionID: core.ActionID{SpellID: 1787},
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			// Stealth triggered auras
			if rogue.HasRune(proto.RogueRune_RuneMasterOfSubtlety) {
				rogue.MasterOfSubtletyAura.Activate(sim)
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			if rogue.HasRune(proto.RogueRune_RuneMasterOfSubtlety) {
				// Refresh aura to have a duration as it should out of stealth
				rogue.MasterOfSubtletyAura.Deactivate(sim)
				rogue.MasterOfSubtletyAura.Activate(sim)
			}
		},
		// Stealth breaks on damage taken (if not absorbed)
		// This may be desirable later, but not applicable currently
	})
}
