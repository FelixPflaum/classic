package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// TODO: 10 yd range
func (rogue *Rogue) registerBlunderbussSpell() {
	if !rogue.HasRune(proto.RogueRune_RuneBlunderbuss) {
		return
	}
	
	activate2PcBonuses := rogue.HasSetBonus(ItemSetNightSlayerBattlearmor, 2)  && rogue.HasAura("Blade Dance") && rogue.HasRune(proto.RogueRune_RuneJustAFleshWound)

	results := make([]*core.SpellResult, min(4, rogue.Env.GetNumTargets()))

	rogue.Blunderbuss = rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 436564},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeRanged,
		ProcMask:    core.ProcMaskRangedSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagCarnage,

		EnergyCost: core.EnergyCostOptions{
			Cost:   20 - core.TernaryFloat64(activate2PcBonuses, 20, 0),
			Refund: 0,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: time.Second * 15,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: core.TernaryFloat64(activate2PcBonuses, 4, 2), //2x base modifier up to 4x with 2pc

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			baseApDamage := spell.MeleeAttackPower() * 0.48

			for idx := range results {
				results[idx] = spell.CalcDamage(sim, target, rogue.rollBlunderbussDamage(sim)+baseApDamage, spell.OutcomeRangedHitAndCrit)
				target = sim.Environment.NextTargetUnit(target)
			}

			for _, result := range results {
				spell.DealDamage(sim, result)
			}
		},
	})
}

func (rogue *Rogue) rollBlunderbussDamage(sim *core.Simulation) float64 {
	baseDamage := rogue.baseRuneAbilityDamage()
	return sim.Roll(baseDamage*1.92, baseDamage*2.88)
}
