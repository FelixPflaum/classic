package hunter

import (
	"time"

	"github.com/wowsims/wotlk/sim/core"
	"github.com/wowsims/wotlk/sim/core/stats"
)

func (hunter *Hunter) registerMultiShotSpell() {
	baseCost := 0.09 * hunter.BaseMana()

	baseEffect := core.SpellEffect{
		ProcMask: core.ProcMaskRangedSpecial,

		BonusCritRating:  4 * core.CritRatingPerCritChance * float64(hunter.Talents.ImprovedBarrage),
		DamageMultiplier: 1 + 0.04*float64(hunter.Talents.Barrage),
		ThreatMultiplier: 1,

		BaseDamage: hunter.talonOfAlarDamageMod(core.BaseDamageConfig{
			Calculator: func(sim *core.Simulation, hitEffect *core.SpellEffect, spell *core.Spell) float64 {
				return (hitEffect.RangedAttackPower(spell.Unit)+hitEffect.RangedAttackPowerOnTarget())*0.2 +
					hunter.AutoAttacks.Ranged.BaseDamage(sim) +
					hunter.AmmoDamageBonus +
					hitEffect.BonusWeaponDamage(spell.Unit) +
					408
			},
			TargetSpellCoefficient: 1,
		}),
		OutcomeApplier: hunter.OutcomeFuncRangedHitAndCrit(hunter.critMultiplier(true, hunter.CurrentTarget)),

		OnSpellHitDealt: func(sim *core.Simulation, spell *core.Spell, spellEffect *core.SpellEffect) {
			hunter.rotation(sim, false)
		},
	}

	numHits := core.MinInt32(3, hunter.Env.GetNumTargets())
	effects := make([]core.SpellEffect, 0, numHits)
	for i := int32(0); i < numHits; i++ {
		effects = append(effects, baseEffect)
		effects[i].Target = hunter.Env.GetTargetUnit(i)
	}

	hunter.MultiShot = hunter.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 49048},
		SpellSchool: core.SpellSchoolPhysical,
		Flags:       core.SpellFlagMeleeMetrics,

		ResourceType: stats.Mana,
		BaseCost:     baseCost,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: baseCost *
					(1 - 0.03*float64(hunter.Talents.Efficiency)) *
					core.TernaryFloat64(ItemSetDemonStalker.CharacterHasSetBonus(&hunter.Character, 4), 0.9, 1),

				GCD:      core.GCDDefault + hunter.latency,
				CastTime: 1, // Dummy value so core doesn't optimize the cast away
			},
			ModifyCast: func(_ *core.Simulation, _ *core.Spell, cast *core.Cast) {
				cast.CastTime = hunter.MultiShotCastTime()
			},
			IgnoreHaste: true, // Hunter GCD is locked at 1.5s
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Second * 10,
			},
		},

		ApplyEffects: core.ApplyEffectFuncDamageMultiple(effects),
	})
}

func (hunter *Hunter) MultiShotCastTime() time.Duration {
	return time.Duration(float64(time.Millisecond*500)/hunter.RangedSwingSpeed()) + hunter.latency
}
