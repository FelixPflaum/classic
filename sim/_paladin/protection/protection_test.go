package protection

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

func init() {
	RegisterProtectionPaladin()
}

func TestProtection(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator(core.CharacterSuiteConfig{
		Class:      proto.Class_ClassPaladin,
		Race:       proto.Race_RaceHuman,
		OtherRaces: []proto.Race{proto.Race_RaceHuman},

		GearSet:     core.GetGearSet("../../../ui/protection_paladin/gear_sets", "p1"),
		Talents:     StandardTalents,
		Consumes:    FullConsumes,
		SpecOptions: core.SpecOptionsCombo{Label: "Protection Paladin SOV", SpecOptions: DefaultOptions},
		OtherSpecOptions: []core.SpecOptionsCombo{
			{
				Label: "Protection Paladin SOC",
				SpecOptions: &proto.Player_ProtectionPaladin{
					ProtectionPaladin: &proto.ProtectionPaladin{
						Options: &proto.ProtectionPaladin_Options{
							Judgement: proto.PaladinJudgement_JudgementOfWisdom,
							Seal:      proto.PaladinSeal_Command,
							Aura:      proto.PaladinAura_RetributionAura,
						},
					},
				},
			},
			{
				Label: "Protection Paladin SOR",
				SpecOptions: &proto.Player_ProtectionPaladin{
					ProtectionPaladin: &proto.ProtectionPaladin{
						Options: &proto.ProtectionPaladin_Options{
							Judgement: proto.PaladinJudgement_JudgementOfWisdom,
							Seal:      proto.PaladinSeal_Righteousness,
							Aura:      proto.PaladinAura_RetributionAura,
						},
					},
				},
			},
		},
		Rotation: core.GetAplRotation("../../../ui/protection_paladin/apls", "default"),

		IsTank:          true,
		InFrontOfTarget: true,

		ItemFilter: core.ItemFilter{
			WeaponTypes: []proto.WeaponType{
				proto.WeaponType_WeaponTypeSword,
				proto.WeaponType_WeaponTypePolearm,
				proto.WeaponType_WeaponTypeMace,
				proto.WeaponType_WeaponTypeShield,
			},
			ArmorType: proto.ArmorType_ArmorTypePlate,
			RangedWeaponTypes: []proto.RangedWeaponType{
				proto.RangedWeaponType_RangedWeaponTypeLibram,
			},
		},
	}))
}

var StandardTalents = "-05005135200132311333312321-511302012003"

var defaultProtOptions = &proto.ProtectionPaladin_Options{
	Judgement: proto.PaladinJudgement_JudgementOfWisdom,
	Seal:      proto.PaladinSeal_Vengeance,
	Aura:      proto.PaladinAura_RetributionAura,
}

var DefaultOptions = &proto.Player_ProtectionPaladin{
	ProtectionPaladin: &proto.ProtectionPaladin{
		Options: defaultProtOptions,
	},
}

var FullConsumes = &proto.Consumes{
	Flask:           proto.Flask_FlaskOfStoneblood,
	Food:            proto.Food_FoodDragonfinFilet,
	DefaultPotion:   proto.Potions_IndestructiblePotion,
	PrepopPotion:    proto.Potions_IndestructiblePotion,
	DefaultConjured: proto.Conjured_ConjuredDarkRune,
}
