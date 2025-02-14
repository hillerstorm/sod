package paladin

import (
	"time"

	"github.com/wowsims/sod/sim/core"
	"github.com/wowsims/sod/sim/core/proto"
)

// Crusader Strike is an ap-normalised instant attack that has a weapon damage % modifier with a 0.75 coefficient.
// It also returns 5% of the paladin's maximum mana when cast, regardless of the ability being negated.
// As of 27/02/24 it deals holy school damage, but otherwise behaves like a melee attack.

func (paladin *Paladin) registerCrusaderStrike() {
	if !paladin.hasRune(proto.PaladinRune_RuneHandsCrusaderStrike) {
		return
	}

	actionID := core.ActionID{SpellID: int32(proto.PaladinRune_RuneHandsCrusaderStrike)}
	manaMetrics := paladin.NewManaMetrics(actionID)

	crusaderStrikeSpell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:    actionID,
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlag_RV | core.SpellFlagIgnoreResists | core.SpellFlagBatchStartAttackMacro,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true, // cs is on phys gcd, which cannot be hasted
			CD: core.Cooldown{
				Timer:    paladin.NewTimer(),
				Duration: time.Second * 6,
			},
		},

		DamageMultiplier: 0.75 * paladin.getWeaponSpecializationModifier(),
		ThreatMultiplier: 1,
		ClassSpellMask:   ClassSpellMask_PaladinCrusaderStrike,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			flags := spell.Flags
			baseDamage := spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower())
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			core.StartDelayedAction(sim, core.DelayedActionOptions{
				DoAt:     sim.CurrentTime + core.SpellBatchWindow,
				Priority: core.ActionPriorityLow,
				OnAction: func(sim *core.Simulation) {
					currentFlags := spell.Flags
					spell.Flags = flags
					spell.DealDamage(sim, result)
					spell.Flags = currentFlags
				},
			})

			paladin.AddMana(sim, 0.05*paladin.MaxMana(), manaMetrics)

			for _, aura := range target.GetAurasWithTag(core.JudgementAuraTag) {
				if aura.IsActive() && aura.Duration < core.NeverExpires {
					aura.UpdateExpires(sim, sim.CurrentTime+time.Second*30)
				}
			}
		},
	})

	paladin.crusaderStrike = crusaderStrikeSpell
}
