package core

import (
	"fmt"

	"github.com/wowsims/sod/sim/core/stats"
)

type SpellResult struct {
	// Target of the spell.
	Target *Unit

	// Results
	Outcome HitOutcome
	Damage  float64 // Damage done by this cast.
	Threat  float64 // The amount of threat generated by this cast.

	ResistanceMultiplier float64 // Partial Resists / Armor multiplier
	PreOutcomeDamage     float64 // Damage done by this cast before Outcome is applied

	inUse bool
}

func (spell *Spell) NewResult(target *Unit) *SpellResult {
	result := &spell.resultCache
	if result.inUse {
		result = &SpellResult{}
	}

	result.Target = target
	result.Damage = 0
	result.Threat = 0
	result.Outcome = OutcomeEmpty // for blocks
	result.inUse = true

	return result
}
func (spell *Spell) DisposeResult(result *SpellResult) {
	result.inUse = false
}

func (result *SpellResult) Landed() bool {
	return result.Outcome.Matches(OutcomeLanded)
}

func (result *SpellResult) DidCrit() bool {
	return result.Outcome.Matches(OutcomeCrit)
}

func (result *SpellResult) DamageString() string {
	outcomeStr := result.Outcome.String()
	if !result.Landed() {
		return outcomeStr
	}
	return fmt.Sprintf("%s for %0.3f damage", outcomeStr, result.Damage)
}

func (result *SpellResult) HealingString() string {
	return fmt.Sprintf("%s for %0.3f healing", result.Outcome.String(), result.Damage)
}

func (result *SpellResult) RawDamage() float64 {
	return result.PreOutcomeDamage / result.ResistanceMultiplier
}

func (spell *Spell) ThreatFromDamage(outcome HitOutcome, damage float64) float64 {
	if outcome.Matches(OutcomeLanded) {
		return (damage*spell.ThreatMultiplier + spell.FlatThreatBonus) * spell.Unit.PseudoStats.ThreatMultiplier
	} else {
		return 0
	}
}

func (spell *Spell) MeleeAttackPower() float64 {
	return spell.Unit.stats[stats.AttackPower] + spell.Unit.PseudoStats.MobTypeAttackPower
}

func (spell *Spell) RangedAttackPower(target *Unit) float64 {
	return spell.Unit.stats[stats.RangedAttackPower] +
		spell.Unit.PseudoStats.MobTypeAttackPower +
		target.PseudoStats.BonusRangedAttackPowerTaken
}

func (spell *Spell) PhysicalHitChance(attackTable *AttackTable) float64 {
	hitRating := spell.Unit.stats[stats.MeleeHit] +
		spell.BonusHitRating +
		attackTable.Defender.PseudoStats.BonusMeleeHitRatingTaken
	hitChance := hitRating / (MeleeHitRatingPerHitChance * 100)
	return max(hitChance-attackTable.HitSuppression, 0)
}

func (spell *Spell) PhysicalCritChance(attackTable *AttackTable) float64 {
	critRating := spell.Unit.stats[stats.MeleeCrit] +
		spell.BonusCritRating
	return critRating/(CritRatingPerCritChance*100) - attackTable.MeleeCritSuppression
}
func (spell *Spell) PhysicalCritCheck(sim *Simulation, attackTable *AttackTable) bool {
	return sim.RandomFloat("Physical Crit Roll") < spell.PhysicalCritChance(attackTable)
}

// The combined bonus damage (aka spell power) for this spell's school(s).
func (spell *Spell) GetBonusDamage() float64 {
	var schoolBonusDamage float64

	switch spell.SchoolIndex {
	case stats.SchoolIndexNone:
		return 0
	case stats.SchoolIndexPhysical:
		// PseudoStats.BonusDamage is physical "spell power", just return that here.
		// TODO: Do "MobTypeSpellPower" effects for physical exist? E.g. something like "Do x extra weapon damage against y type"?
		// If yes then it needs to be handled here and for the multi school case below.
		return spell.Unit.PseudoStats.BonusDamage
	case stats.SchoolIndexArcane:
		schoolBonusDamage = spell.Unit.GetStat(stats.ArcanePower)
	case stats.SchoolIndexFire:
		schoolBonusDamage = spell.Unit.GetStat(stats.FirePower)
	case stats.SchoolIndexFrost:
		schoolBonusDamage = spell.Unit.GetStat(stats.FrostPower)
	case stats.SchoolIndexHoly:
		schoolBonusDamage = spell.Unit.GetStat(stats.HolyPower)
	case stats.SchoolIndexNature:
		schoolBonusDamage = spell.Unit.GetStat(stats.NaturePower)
	case stats.SchoolIndexShadow:
		schoolBonusDamage = spell.Unit.GetStat(stats.ShadowPower)
	default:
		// Multi school: Get best power choice available.
		for _, baseSchoolIndex := range spell.SchoolBaseIndices {
			var power float64

			// TODO: If physical multi school spells with a coef > 0 emerge at any point this function or
			// the way bonus damage stats are set up needs to be changed.
			// SpellPower, MobTypeSpellPower and SpellDamage stats are a convinience to group bonus damage for
			// non-physical schools and would need to be considered against physical here and obviously not applied
			// if physical is chosen.
			if baseSchoolIndex == stats.SchoolIndexPhysical {
				power = spell.Unit.PseudoStats.BonusDamage
			} else {
				// School and stat indices are ordered the same way.
				power = spell.Unit.GetStat(stats.ArcanePower + stats.Stat(baseSchoolIndex) - 2)
			}

			if power > schoolBonusDamage {
				schoolBonusDamage = power
			}
		}
	}

	return spell.BonusDamage +
		schoolBonusDamage +
		spell.Unit.GetStat(stats.SpellPower) +
		spell.Unit.GetStat(stats.SpellDamage) +
		spell.Unit.PseudoStats.MobTypeSpellPower
}

func (spell *Spell) SpellHitChance(target *Unit) float64 {
	hitRating := spell.Unit.stats[stats.SpellHit] +
		spell.BonusHitRating +
		spell.Unit.GetSchoolBonusHitChance(spell) +
		target.PseudoStats.BonusSpellHitRatingTaken

	return hitRating / (SpellHitRatingPerHitChance * 100)
}
func (spell *Spell) SpellChanceToMiss(attackTable *AttackTable) float64 {
	missChance := 0.01

	if spell.Flags.Matches(SpellFlagBinary) {
		baseHitChance := (1 - attackTable.BaseSpellMissChance) * attackTable.GetBinaryHitChance(spell)
		missChance = 1 - baseHitChance - spell.SpellHitChance(attackTable.Defender)
	} else {
		missChance = attackTable.BaseSpellMissChance - spell.SpellHitChance(attackTable.Defender)
	}

	// Always a 1% chance to miss in classic
	return max(0.01, missChance)
}
func (spell *Spell) MagicHitCheck(sim *Simulation, attackTable *AttackTable) bool {
	return sim.Proc(1.0-spell.SpellChanceToMiss(attackTable), "Magical Hit Roll")
}

func (spell *Spell) spellCritRating(_ *Unit) float64 {
	return spell.Unit.stats[stats.SpellCrit] +
		spell.BonusCritRating
}
func (spell *Spell) SpellCritChance(target *Unit) float64 {
	// TODO: Classic verify crit suppression
	return spell.spellCritRating(target)/(SpellCritRatingPerCritChance*100) +
		target.GetSchoolCritTakenChance(spell)
	// - spell.Unit.AttackTables[target.UnitIndex][spell.CastType].SpellCritSuppression
}
func (spell *Spell) MagicCritCheck(sim *Simulation, target *Unit) bool {
	critChance := spell.SpellCritChance(target)
	return sim.RandomFloat("Magical Crit Roll") < critChance
}

func (spell *Spell) HealingPower(target *Unit) float64 {
	return spell.Unit.GetStat(stats.SpellPower) +
		spell.Unit.GetStat(stats.HealingPower) +
		target.PseudoStats.BonusHealingTaken
}
func (spell *Spell) healingCritRating() float64 {
	return spell.Unit.GetStat(stats.SpellCrit) + spell.BonusCritRating
}
func (spell *Spell) HealingCritChance() float64 {
	return spell.healingCritRating() / (CritRatingPerCritChance * 100)
}

func (spell *Spell) HealingCritCheck(sim *Simulation) bool {
	critChance := spell.HealingCritChance()
	return sim.RandomFloat("Healing Crit Roll") < critChance
}

func (spell *Spell) ApplyPostOutcomeDamageModifiers(sim *Simulation, result *SpellResult) {
	attackTable := spell.Unit.AttackTables[result.Target.UnitIndex][spell.CastType]
	for i := range result.Target.DynamicDamageTakenModifiers {
		result.Target.DynamicDamageTakenModifiers[i](sim, spell, result)
	}
	result.Damage += spell.TargetBonusDamageTakenAfterModifiers(attackTable)
	result.Damage = max(0, result.Damage)
}

// For spells that do no damage but still have a hit/miss check.
func (spell *Spell) CalcOutcome(sim *Simulation, target *Unit, outcomeApplier OutcomeApplier) *SpellResult {
	attackTable := spell.Unit.AttackTables[target.UnitIndex][spell.CastType]
	result := spell.NewResult(target)

	outcomeApplier(sim, result, attackTable)
	result.Threat = spell.ThreatFromDamage(result.Outcome, result.Damage)
	return result
}

func (spell *Spell) calcDamageInternal(sim *Simulation, target *Unit, baseDamage float64, attackerMultiplier float64, isPeriodic bool, outcomeApplier OutcomeApplier) *SpellResult {
	attackTable := spell.Unit.AttackTables[target.UnitIndex][spell.CastType]

	result := spell.NewResult(target)
	result.Damage = baseDamage

	if sim.Log == nil {
		result.Damage *= attackerMultiplier
		result.applyResistances(sim, spell, isPeriodic, attackTable)
		result.applyTargetModifiers(spell, attackTable, isPeriodic)

		// Save partial outcome which comes from applyResistances call
		partialOutcome := OutcomeEmpty
		if result.Outcome.Matches(OutcomePartial) {
			partialOutcome = result.Outcome & OutcomePartial
		}

		// outcome applier overwrites the Outcome
		outcomeApplier(sim, result, attackTable)

		// Restore partial outcome
		if partialOutcome != OutcomeEmpty {
			result.Outcome |= partialOutcome
		}

		spell.ApplyPostOutcomeDamageModifiers(sim, result)
	} else {
		result.Damage *= attackerMultiplier
		afterAttackMods := result.Damage
		result.applyResistances(sim, spell, isPeriodic, attackTable)
		afterResistances := result.Damage
		result.applyTargetModifiers(spell, attackTable, isPeriodic)
		afterTargetMods := result.Damage

		// Save partial outcome which comes from applyResistances call
		partialOutcome := OutcomeEmpty
		if result.Outcome.Matches(OutcomePartial) {
			partialOutcome = result.Outcome & OutcomePartial
		}

		// outcome applier overwrites the Outcome
		outcomeApplier(sim, result, attackTable)

		// Restore partial outcome
		if partialOutcome != OutcomeEmpty {
			result.Outcome |= partialOutcome
		}

		afterOutcome := result.Damage
		spell.ApplyPostOutcomeDamageModifiers(sim, result)
		afterPostOutcome := result.Damage

		spell.Unit.Log(
			sim,
			"%s %s [DEBUG] MAP: %0.01f, RAP: %0.01f, SP: %0.01f, BaseDamage:%0.01f, AfterAttackerMods:%0.01f, AfterResistances:%0.01f, AfterTargetMods:%0.01f, AfterOutcome:%0.01f, AfterPostOutcome:%0.01f",
			target.LogLabel(), spell.ActionID, spell.Unit.GetStat(stats.AttackPower), spell.Unit.GetStat(stats.RangedAttackPower), spell.Unit.GetStat(stats.SpellPower), baseDamage, afterAttackMods, afterResistances, afterTargetMods, afterOutcome, afterPostOutcome)
	}

	result.Threat = spell.ThreatFromDamage(result.Outcome, result.Damage)

	return result
}
func (spell *Spell) CalcDamage(sim *Simulation, target *Unit, baseDamage float64, outcomeApplier OutcomeApplier) *SpellResult {
	attackerMultiplier := spell.AttackerDamageMultiplier(spell.Unit.AttackTables[target.UnitIndex][spell.CastType])
	if spell.BonusCoefficient > 0 {
		baseDamage += spell.BonusCoefficient * spell.GetBonusDamage()
	}
	return spell.calcDamageInternal(sim, target, baseDamage, attackerMultiplier, false, outcomeApplier)
}
func (spell *Spell) CalcPeriodicDamage(sim *Simulation, target *Unit, baseDamage float64, outcomeApplier OutcomeApplier) *SpellResult {
	attackerMultiplier := spell.AttackerDamageMultiplier(spell.Unit.AttackTables[target.UnitIndex][spell.CastType])
	if dot := spell.DotOrAOEDot(target); dot.BonusCoefficient > 0 {
		baseDamage += dot.BonusCoefficient * spell.GetBonusDamage()
	}
	return spell.calcDamageInternal(sim, target, baseDamage, attackerMultiplier, true, outcomeApplier)
}
func (dot *Dot) CalcSnapshotDamage(sim *Simulation, target *Unit, outcomeApplier OutcomeApplier) *SpellResult {
	return dot.Spell.calcDamageInternal(sim, target, dot.SnapshotBaseDamage, dot.SnapshotAttackerMultiplier, true, outcomeApplier)
}

func (dot *Dot) Snapshot(target *Unit, baseDamage float64, isRollover bool) {
	// Rollovers in SoD don't seem to update anything
	if !isRollover {
		dot.SnapshotBaseDamage = baseDamage
		if dot.BonusCoefficient > 0 {
			dot.SnapshotBaseDamage += dot.BonusCoefficient * dot.Spell.GetBonusDamage()
		}
		attackTable := dot.Spell.Unit.AttackTables[target.UnitIndex][dot.Spell.CastType]
		dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(attackTable)
		if dot.Spell.SchoolIndex == stats.SchoolIndexPhysical {
			dot.SnapshotCritChance = dot.Spell.PhysicalCritChance(attackTable)
		} else {
			dot.SnapshotCritChance = dot.Spell.SpellCritChance(target)
		}
	}
}

func (spell *Spell) DealOutcome(sim *Simulation, result *SpellResult) {
	spell.DealDamage(sim, result)
}
func (spell *Spell) CalcAndDealOutcome(sim *Simulation, target *Unit, outcomeApplier OutcomeApplier) *SpellResult {
	result := spell.CalcOutcome(sim, target, outcomeApplier)
	spell.DealDamage(sim, result)
	return result
}

// Applies the fully computed spell result to the sim.
func (spell *Spell) dealDamageInternal(sim *Simulation, isPeriodic bool, result *SpellResult) {
	if sim.CurrentTime >= 0 {
		spell.SpellMetrics[result.Target.UnitIndex].TotalDamage += result.Damage
		spell.SpellMetrics[result.Target.UnitIndex].TotalThreat += result.Threat
	}

	// Mark total damage done in raid so far for health based fights.
	// Don't include damage done by EnemyUnits to Players
	if result.Target.Type == EnemyUnit {
		sim.Encounter.DamageTaken += result.Damage
	}

	if sim.Log != nil && !spell.Flags.Matches(SpellFlagNoLogs) {
		if isPeriodic {
			spell.Unit.Log(sim, "%s %s tick %s. (Threat: %0.3f)", result.Target.LogLabel(), spell.ActionID, result.DamageString(), result.Threat)
		} else {
			spell.Unit.Log(sim, "%s %s %s. (Threat: %0.3f)", result.Target.LogLabel(), spell.ActionID, result.DamageString(), result.Threat)
		}
	}

	if !spell.Flags.Matches(SpellFlagNoOnDamageDealt) {
		if isPeriodic {
			spell.Unit.OnPeriodicDamageDealt(sim, spell, result)
			result.Target.OnPeriodicDamageTaken(sim, spell, result)
		} else {
			spell.Unit.OnSpellHitDealt(sim, spell, result)
			result.Target.OnSpellHitTaken(sim, spell, result)
		}
	}

	spell.DisposeResult(result)
}
func (spell *Spell) DealDamage(sim *Simulation, result *SpellResult) {
	spell.dealDamageInternal(sim, false, result)
}
func (spell *Spell) DealPeriodicDamage(sim *Simulation, result *SpellResult) {
	spell.dealDamageInternal(sim, true, result)
}

func (spell *Spell) CalcAndDealDamage(sim *Simulation, target *Unit, baseDamage float64, outcomeApplier OutcomeApplier) *SpellResult {
	result := spell.CalcDamage(sim, target, baseDamage, outcomeApplier)
	spell.DealDamage(sim, result)
	return result
}
func (spell *Spell) CalcAndDealPeriodicDamage(sim *Simulation, target *Unit, baseDamage float64, outcomeApplier OutcomeApplier) *SpellResult {
	result := spell.CalcPeriodicDamage(sim, target, baseDamage, outcomeApplier)
	spell.DealPeriodicDamage(sim, result)
	return result
}
func (dot *Dot) CalcAndDealPeriodicSnapshotDamage(sim *Simulation, target *Unit, outcomeApplier OutcomeApplier) *SpellResult {
	result := dot.CalcSnapshotDamage(sim, target, outcomeApplier)
	dot.Spell.DealPeriodicDamage(sim, result)
	return result
}

func (spell *Spell) calcHealingInternal(sim *Simulation, target *Unit, baseHealing float64, casterMultiplier float64, outcomeApplier OutcomeApplier) *SpellResult {
	attackTable := spell.Unit.AttackTables[target.UnitIndex][spell.CastType]

	result := spell.NewResult(target)
	result.Damage = baseHealing

	if sim.Log == nil {
		result.Damage *= casterMultiplier
		result.Damage = spell.applyTargetHealingModifiers(result.Damage, attackTable)
		outcomeApplier(sim, result, attackTable)
	} else {
		result.Damage *= casterMultiplier
		afterCasterMods := result.Damage
		result.Damage = spell.applyTargetHealingModifiers(result.Damage, attackTable)
		afterTargetMods := result.Damage
		outcomeApplier(sim, result, attackTable)
		afterOutcome := result.Damage

		spell.Unit.Log(
			sim,
			"%s %s [DEBUG] HealingPower: %0.01f, BaseHealing:%0.01f, AfterCasterMods:%0.01f, AfterTargetMods:%0.01f, AfterOutcome:%0.01f",
			target.LogLabel(), spell.ActionID, spell.HealingPower(target), baseHealing, afterCasterMods, afterTargetMods, afterOutcome)
	}

	result.Threat = spell.ThreatFromDamage(result.Outcome, result.Damage)

	return result
}
func (spell *Spell) CalcHealing(sim *Simulation, target *Unit, baseHealing float64, outcomeApplier OutcomeApplier) *SpellResult {
	if spell.BonusCoefficient > 0 {
		baseHealing += spell.BonusCoefficient * spell.HealingPower(target)
	}
	return spell.calcHealingInternal(sim, target, baseHealing, spell.CasterHealingMultiplier(), outcomeApplier)
}
func (dot *Dot) CalcSnapshotHealing(sim *Simulation, target *Unit, outcomeApplier OutcomeApplier) *SpellResult {
	return dot.Spell.calcHealingInternal(sim, target, dot.SnapshotBaseDamage, dot.SnapshotAttackerMultiplier, outcomeApplier)
}

func (dot *Dot) SnapshotHeal(target *Unit, baseHealing float64, isRollover bool) {
	// Rollovers in SoD don't seem to update anything
	if !isRollover {
		dot.SnapshotBaseDamage = baseHealing
		if dot.BonusCoefficient > 0 {
			dot.SnapshotBaseDamage += dot.BonusCoefficient * dot.Spell.HealingPower(target)
		}
		dot.SnapshotCritChance = dot.Spell.SpellCritChance(target)
		attackTable := dot.Spell.Unit.AttackTables[target.UnitIndex][dot.Spell.CastType]
		dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(attackTable)
	}
}

// Applies the fully computed spell result to the sim.
func (spell *Spell) dealHealingInternal(sim *Simulation, isPeriodic bool, result *SpellResult) {
	spell.SpellMetrics[result.Target.UnitIndex].TotalHealing += result.Damage
	spell.SpellMetrics[result.Target.UnitIndex].TotalThreat += result.Threat
	if result.Target.HasHealthBar() {
		result.Target.GainHealth(sim, result.Damage, spell.HealthMetrics(result.Target))
	}

	if sim.Log != nil {
		if isPeriodic {
			spell.Unit.Log(sim, "%s %s tick %s. (Threat: %0.3f)", result.Target.LogLabel(), spell.ActionID, result.HealingString(), result.Threat)
		} else {
			spell.Unit.Log(sim, "%s %s %s. (Threat: %0.3f)", result.Target.LogLabel(), spell.ActionID, result.HealingString(), result.Threat)
		}
	}

	if isPeriodic {
		spell.Unit.OnPeriodicHealDealt(sim, spell, result)
		result.Target.OnPeriodicHealTaken(sim, spell, result)
	} else {
		spell.Unit.OnHealDealt(sim, spell, result)
		result.Target.OnHealTaken(sim, spell, result)
	}

	spell.DisposeResult(result)
}
func (spell *Spell) DealHealing(sim *Simulation, result *SpellResult) {
	spell.dealHealingInternal(sim, false, result)
}
func (spell *Spell) DealPeriodicHealing(sim *Simulation, result *SpellResult) {
	spell.dealHealingInternal(sim, true, result)
}

func (spell *Spell) CalcAndDealHealing(sim *Simulation, target *Unit, baseHealing float64, outcomeApplier OutcomeApplier) *SpellResult {
	result := spell.CalcHealing(sim, target, baseHealing, outcomeApplier)
	spell.DealHealing(sim, result)
	return result
}
func (spell *Spell) CalcAndDealPeriodicHealing(sim *Simulation, target *Unit, baseHealing float64, outcomeApplier OutcomeApplier) *SpellResult {
	// This is currently identical to CalcAndDealHealing, but keeping it separate in case they become different in the future.
	return spell.CalcAndDealHealing(sim, target, baseHealing, outcomeApplier)
}
func (dot *Dot) CalcAndDealPeriodicSnapshotHealing(sim *Simulation, target *Unit, outcomeApplier OutcomeApplier) *SpellResult {
	result := dot.CalcSnapshotHealing(sim, target, outcomeApplier)
	dot.Spell.DealPeriodicHealing(sim, result)
	return result
}

func (spell *Spell) WaitTravelTime(sim *Simulation, callback func(*Simulation)) {
	StartDelayedAction(sim, DelayedActionOptions{
		DoAt:     sim.CurrentTime + spell.TravelTime(),
		OnAction: callback,
	})
}

// Returns the combined attacker modifiers.
func (spell *Spell) AttackerDamageMultiplier(attackTable *AttackTable) float64 {
	return spell.attackerDamageMultiplierInternal(attackTable) *
		spell.DamageMultiplier *
		spell.DamageMultiplierAdditive
}
func (spell *Spell) attackerDamageMultiplierInternal(attackTable *AttackTable) float64 {
	if spell.Flags.Matches(SpellFlagIgnoreAttackerModifiers) {
		return 1
	}

	return spell.Unit.PseudoStats.DamageDealtMultiplier *
		spell.Unit.GetSchoolDamageDoneMultiplier(spell) *
		attackTable.DamageDealtMultiplier
}

func (result *SpellResult) applyTargetModifiers(spell *Spell, attackTable *AttackTable, isPeriodic bool) {
	result.Damage += spell.TargetBonusDamageTakenBeforeModifiers(attackTable)

	if isPeriodic {
		if dot := spell.DotOrAOEDot(attackTable.Defender); dot.BonusCoefficient > 0 {
			result.Damage += attackTable.Defender.GetSchoolBonusDamageTaken(spell) * dot.BonusCoefficient
		}
	} else {
		if spell.BonusCoefficient > 0 {
			result.Damage += attackTable.Defender.GetSchoolBonusDamageTaken(spell) * spell.BonusCoefficient
		}
	}

	result.Damage *= spell.TargetDamageMultiplier(attackTable, isPeriodic)
}

// Apply flat bonus damage taken before modifiers
// e.g. Blessing of Sanctuary
func (spell *Spell) TargetBonusDamageTakenBeforeModifiers(attackTable *AttackTable) float64 {
	if spell.Flags.Matches(SpellFlagIgnoreTargetModifiers) {
		return 0
	}

	return attackTable.Defender.PseudoStats.BonusDamageTakenBeforeModifiers[spell.DefenseType]
}

// Apply flat bonus damage taken after modifiers
// e.g. Stoneskin Totem, Windwall Totem
func (spell *Spell) TargetBonusDamageTakenAfterModifiers(attackTable *AttackTable) float64 {
	if spell.Flags.Matches(SpellFlagIgnoreTargetModifiers) {
		return 0
	}

	return attackTable.Defender.PseudoStats.BonusDamageTakenAfterModifiers[spell.DefenseType]
}
func (spell *Spell) TargetDamageMultiplier(attackTable *AttackTable, isPeriodic bool) float64 {
	if spell.Flags.Matches(SpellFlagIgnoreTargetModifiers) {
		return 1
	}

	multiplier := attackTable.Defender.PseudoStats.DamageTakenMultiplier *
		attackTable.Defender.GetSchoolDamageTakenMultiplier(spell) *
		attackTable.DamageTakenMultiplier

	if spell.Flags.Matches(SpellFlagPoison) {
		multiplier *= attackTable.Defender.PseudoStats.PoisonDamageTakenMultiplier
	}

	if isPeriodic && spell.SpellSchool.Matches(SpellSchoolPhysical) {
		multiplier *= attackTable.Defender.PseudoStats.BleedDamageTakenMultiplier
	}

	if attackTable.DamageDoneByCasterMultiplier != nil {
		multiplier *= attackTable.DamageDoneByCasterMultiplier(spell, attackTable)
	}

	return multiplier
}

func (spell *Spell) CasterHealingMultiplier() float64 {
	if spell.Flags.Matches(SpellFlagIgnoreAttackerModifiers) {
		return 1
	}

	return spell.DamageMultiplier * spell.DamageMultiplierAdditive
}
func (spell *Spell) applyTargetHealingModifiers(damage float64, attackTable *AttackTable) float64 {
	if spell.Flags.Matches(SpellFlagIgnoreTargetModifiers) {
		return damage
	}

	return damage *
		attackTable.Defender.PseudoStats.HealingTakenMultiplier *
		attackTable.HealingDealtMultiplier
}
