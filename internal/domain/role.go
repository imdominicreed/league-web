package domain

// Role represents a League of Legends position
type Role string

const (
	RoleTop     Role = "top"
	RoleJungle  Role = "jungle"
	RoleMid     Role = "mid"
	RoleADC     Role = "adc"
	RoleSupport Role = "support"
)

// AllRoles contains all valid roles in order
var AllRoles = []Role{RoleTop, RoleJungle, RoleMid, RoleADC, RoleSupport}

// IsValid checks if a role is valid
func (r Role) IsValid() bool {
	switch r {
	case RoleTop, RoleJungle, RoleMid, RoleADC, RoleSupport:
		return true
	}
	return false
}

// String returns the string representation of the role
func (r Role) String() string {
	return string(r)
}

// DisplayName returns a user-friendly display name for the role
func (r Role) DisplayName() string {
	switch r {
	case RoleTop:
		return "Top"
	case RoleJungle:
		return "Jungle"
	case RoleMid:
		return "Mid"
	case RoleADC:
		return "ADC"
	case RoleSupport:
		return "Support"
	default:
		return string(r)
	}
}

// LeagueRank represents a League of Legends rank
type LeagueRank string

const (
	RankUnranked    LeagueRank = "Unranked"
	RankIron4       LeagueRank = "Iron IV"
	RankIron3       LeagueRank = "Iron III"
	RankIron2       LeagueRank = "Iron II"
	RankIron1       LeagueRank = "Iron I"
	RankBronze4     LeagueRank = "Bronze IV"
	RankBronze3     LeagueRank = "Bronze III"
	RankBronze2     LeagueRank = "Bronze II"
	RankBronze1     LeagueRank = "Bronze I"
	RankSilver4     LeagueRank = "Silver IV"
	RankSilver3     LeagueRank = "Silver III"
	RankSilver2     LeagueRank = "Silver II"
	RankSilver1     LeagueRank = "Silver I"
	RankGold4       LeagueRank = "Gold IV"
	RankGold3       LeagueRank = "Gold III"
	RankGold2       LeagueRank = "Gold II"
	RankGold1       LeagueRank = "Gold I"
	RankPlatinum4   LeagueRank = "Platinum IV"
	RankPlatinum3   LeagueRank = "Platinum III"
	RankPlatinum2   LeagueRank = "Platinum II"
	RankPlatinum1   LeagueRank = "Platinum I"
	RankEmerald4    LeagueRank = "Emerald IV"
	RankEmerald3    LeagueRank = "Emerald III"
	RankEmerald2    LeagueRank = "Emerald II"
	RankEmerald1    LeagueRank = "Emerald I"
	RankDiamond4    LeagueRank = "Diamond IV"
	RankDiamond3    LeagueRank = "Diamond III"
	RankDiamond2    LeagueRank = "Diamond II"
	RankDiamond1    LeagueRank = "Diamond I"
	RankMaster      LeagueRank = "Master"
	RankGrandmaster LeagueRank = "Grandmaster"
	RankChallenger  LeagueRank = "Challenger"
)

// AllRanks contains all valid ranks in ascending order
var AllRanks = []LeagueRank{
	RankUnranked,
	RankIron4, RankIron3, RankIron2, RankIron1,
	RankBronze4, RankBronze3, RankBronze2, RankBronze1,
	RankSilver4, RankSilver3, RankSilver2, RankSilver1,
	RankGold4, RankGold3, RankGold2, RankGold1,
	RankPlatinum4, RankPlatinum3, RankPlatinum2, RankPlatinum1,
	RankEmerald4, RankEmerald3, RankEmerald2, RankEmerald1,
	RankDiamond4, RankDiamond3, RankDiamond2, RankDiamond1,
	RankMaster, RankGrandmaster, RankChallenger,
}

// rankToMMR maps each rank to a base MMR value
var rankToMMR = map[LeagueRank]int{
	RankUnranked:    1200,
	RankIron4:       400,
	RankIron3:       500,
	RankIron2:       600,
	RankIron1:       700,
	RankBronze4:     800,
	RankBronze3:     900,
	RankBronze2:     1000,
	RankBronze1:     1100,
	RankSilver4:     1200,
	RankSilver3:     1300,
	RankSilver2:     1400,
	RankSilver1:     1500,
	RankGold4:       1600,
	RankGold3:       1700,
	RankGold2:       1800,
	RankGold1:       1900,
	RankPlatinum4:   2000,
	RankPlatinum3:   2100,
	RankPlatinum2:   2200,
	RankPlatinum1:   2300,
	RankEmerald4:    2400,
	RankEmerald3:    2500,
	RankEmerald2:    2600,
	RankEmerald1:    2700,
	RankDiamond4:    2800,
	RankDiamond3:    2900,
	RankDiamond2:    3000,
	RankDiamond1:    3100,
	RankMaster:      3200,
	RankGrandmaster: 3400,
	RankChallenger:  3600,
}

// ToMMR converts a league rank to its base MMR value
func (r LeagueRank) ToMMR() int {
	if mmr, ok := rankToMMR[r]; ok {
		return mmr
	}
	return 1200 // Default to Silver 4 equivalent
}

// IsValid checks if a rank is valid
func (r LeagueRank) IsValid() bool {
	_, ok := rankToMMR[r]
	return ok
}

// String returns the string representation of the rank
func (r LeagueRank) String() string {
	return string(r)
}

// MMRToRank converts an MMR value to the closest league rank
func MMRToRank(mmr int) LeagueRank {
	if mmr < 400 {
		return RankUnranked
	}

	var closestRank LeagueRank
	closestDiff := 10000

	for rank, rankMMR := range rankToMMR {
		if rank == RankUnranked {
			continue
		}
		diff := abs(mmr - rankMMR)
		if diff < closestDiff {
			closestDiff = diff
			closestRank = rank
		}
	}

	return closestRank
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
