package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (paladin *Paladin) registerAvengersShieldSpell() {
	// apply to up to 3 targets
	numHits := min(3, paladin.Env.GetNumTargets())
	results := make([]*core.SpellResult, numHits)

	paladin.AvengersShield = paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 48827},
		SpellSchool: core.SpellSchoolHoly,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

		ManaCost: core.ManaCostOptions{
			BaseCost:   0.26,
			Multiplier: 1 - 0.02*float64(paladin.Talents.Benediction),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    paladin.NewTimer(),
				Duration: time.Second * 30,
			},
		},

		DamageMultiplier: 1,
		CritMultiplier:   paladin.MeleeCritMultiplier(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			constBaseDamage := .07*spell.SpellDamage() + .07*spell.MeleeAttackPower()

			curTarget := target
			for hitIndex := int32(0); hitIndex < numHits; hitIndex++ {
				baseDamage := constBaseDamage + sim.Roll(1100, 1344)

				results[hitIndex] = spell.CalcDamage(sim, curTarget, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
				curTarget = sim.Environment.NextTargetUnit(curTarget)
			}

			curTarget = target
			for hitIndex := int32(0); hitIndex < numHits; hitIndex++ {
				spell.DealDamage(sim, results[hitIndex])
				curTarget = sim.Environment.NextTargetUnit(curTarget)
			}
		},
	})
}
