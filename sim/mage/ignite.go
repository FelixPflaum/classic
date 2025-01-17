package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// If two spells proc Ignite at almost exactly the same time, the latter
// overwrites the former.
const IgniteTicks = 2

func (mage *Mage) applyIgnite() {
	if mage.Talents.Ignite == 0 {
		return
	}

	mage.RegisterAura(core.Aura{
		Label:    "Ignite Talent",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !spell.ProcMask.Matches(core.ProcMaskSpellDamage) {
				return
			}
			if spell.SpellSchool.Matches(core.SpellSchoolFire) && result.DidCrit() {
				mage.procIgnite(sim, result)
			}
		},
		// TODO: Classic verify mechanics match for rune based Living Bomb
		OnPeriodicDamageDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !spell.ProcMask.Matches(core.ProcMaskSpellDamage) {
				return
			}
			if mage.LivingBomb != nil && result.DidCrit() {
				mage.procIgnite(sim, result)
			}
		},
	})

	mage.Ignite = mage.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_MageIgnite,
		ActionID:    core.ActionID{SpellID: 12654},
		SpellSchool: core.SpellSchoolFire,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellProc,
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell | SpellFlagMage,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Cast: core.CastConfig{
			IgnoreHaste: true,
		},

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Ignite",
			},
			NumberOfTicks: IgniteTicks,
			TickLength:    time.Second * 2,
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.Dot(target).ApplyOrReset(sim)
		},
	})
}

func (mage *Mage) procIgnite(sim *core.Simulation, result *core.SpellResult) {
	dot := mage.Ignite.Dot(result.Target)

	newDamage := result.Damage * 0.08 * float64(mage.Talents.Ignite)
	outstandingDamage := core.TernaryFloat64(dot.IsActive(), dot.SnapshotBaseDamage*float64(dot.NumberOfTicks-dot.TickCount), 0)

	dot.Snapshot(result.Target, (outstandingDamage+newDamage)/float64(IgniteTicks), false)

	mage.Ignite.Cast(sim, result.Target)
}
