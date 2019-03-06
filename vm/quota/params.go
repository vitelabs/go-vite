package quota

import (
	"errors"
	"math/big"
	"time"
)

var (
	errGasUintOverflow = errors.New("gas uint64 overflow")
)

const (
	quotaForCreateContract uint64 = 1000000 // Quota limit for create contract.
	quotaForSection        uint64 = 21000

	maxQuotaHeightGap uint64 = 3600 * 24 // Maximum Snapshot block height gap to gain quota by pledge.

	precForFloat uint = 18

	day            = 24 * time.Hour
	powTimesPerDay = 10
)

type QuotaParams struct {
	paramA *big.Float // Quota param of pledge amount
	paramB *big.Float // Quota param of PoW difficulty
}

func NewQuotaParams(strA, strB string) QuotaParams {
	paramA, _ := new(big.Float).SetPrec(precForFloat).SetString(strA)
	paramB, _ := new(big.Float).SetPrec(precForFloat).SetString(strB)
	return QuotaParams{paramA, paramB}
}

var (
	QuotaParamTest    = NewQuotaParams("4.200604096e-21", "6.40975486e-07")
	QuotaParamMainNet = NewQuotaParams("4.200627522e-24", "6.259419649e-10")

	sectionStrList = []string{
		"0.0",
		"0.042006175634155006",
		"0.08404944434245186",
		"0.1261670961035256",
		"0.16839681732546105",
		"0.2107768956769977",
		"0.25334643304410037",
		"0.2961455696376917",
		"0.3392157225669637",
		"0.382599842575369",
		"0.4263426931297194",
		"0.4704911566788094",
		"0.5150945736855665",
		"0.5602051210238872",
		"0.605878237567604",
		"0.6521731063496397",
		"0.6991532046201573",
		"0.7468869355972497",
		"0.7954483588344243",
		"0.8449180401302736",
		"0.8953840470548413",
		"0.9469431228444231",
		"0.9997020801479394",
		"1.053779467629503",
		"1.1093075777848576",
		"1.1664348850068706",
		"1.2253290311060194",
		"1.286180514353531",
		"1.3492072924575544",
		"1.4146605870070175",
		"1.4828322881625378",
		"1.554064521717701",
		"1.6287621852605034",
		"1.707409634545938",
		"1.7905932883378723",
		"1.8790328663947373",
		"1.97362554890186",
		"2.0755100566945326",
		"2.186162517630361",
		"2.3075451472522963",
		"2.4423470353692043",
		"2.594395323511559",
		"2.7694056956796604",
		"2.976475888792767",
		"3.2314282909393213",
		"3.5656840708200748",
		"4.057395776090949",
		"5.029431885090279",
	}

	difficultyListTest = []*big.Int{
		big.NewInt(0),
		big.NewInt(65535),
		big.NewInt(131127),
		big.NewInt(196836),
		big.NewInt(262720),
		big.NewInt(328838),
		big.NewInt(395250),
		big.NewInt(462024),
		big.NewInt(529220),
		big.NewInt(596904),
		big.NewInt(665147),
		big.NewInt(734024),
		big.NewInt(803612),
		big.NewInt(873984),
		big.NewInt(945240),
		big.NewInt(1017468),
		big.NewInt(1090768),
		big.NewInt(1165237),
		big.NewInt(1241000),
		big.NewInt(1318176),
		big.NewInt(1396912),
		big.NewInt(1477344),
		big.NewInt(1559656),
		big.NewInt(1644024),
		big.NewInt(1730656),
		big.NewInt(1819784),
		big.NewInt(1911656),
		big.NewInt(2006592),
		big.NewInt(2104928),
		big.NewInt(2207040),
		big.NewInt(2313408),
		big.NewInt(2424528),
		big.NewInt(2541072),
		big.NewInt(2663776),
		big.NewInt(2793552),
		big.NewInt(2931520),
		big.NewInt(3079104),
		big.NewInt(3238064),
		big.NewInt(3410672),
		big.NewInt(3600048),
		big.NewInt(3810368),
		big.NewInt(4047568),
		big.NewInt(4320608),
		big.NewInt(4643648),
		big.NewInt(5041440),
		big.NewInt(5562912),
		big.NewInt(6330048),
		big.NewInt(7846496),
	}
	difficultyListMainNet = []*big.Int{
		big.NewInt(0),
		big.NewInt(67108864),
		big.NewInt(134276096),
		big.NewInt(201564160),
		big.NewInt(269029376),
		big.NewInt(336736256),
		big.NewInt(404742144),
		big.NewInt(473120768),
		big.NewInt(541929472),
		big.NewInt(611241984),
		big.NewInt(681119744),
		big.NewInt(751652864),
		big.NewInt(822910976),
		big.NewInt(894976000),
		big.NewInt(967946240),
		big.NewInt(1041903616),
		big.NewInt(1116962816),
		big.NewInt(1193222144),
		big.NewInt(1270800384),
		big.NewInt(1349836800),
		big.NewInt(1430462464),
		big.NewInt(1512824832),
		big.NewInt(1597120512),
		big.NewInt(1683513344),
		big.NewInt(1772216320),
		big.NewInt(1863491584),
		big.NewInt(1957568512),
		big.NewInt(2054791168),
		big.NewInt(2155479040),
		big.NewInt(2260041728),
		big.NewInt(2368962560),
		big.NewInt(2482757632),
		big.NewInt(2602090496),
		big.NewInt(2727755776),
		big.NewInt(2860646400),
		big.NewInt(3001933824),
		big.NewInt(3153051648),
		big.NewInt(3315826688),
		big.NewInt(3492593664),
		big.NewInt(3686514688),
		big.NewInt(3901882368),
		big.NewInt(4144775168),
		big.NewInt(4424400896),
		big.NewInt(4755193856),
		big.NewInt(5162500096),
		big.NewInt(5696520192),
		big.NewInt(6482067456),
		big.NewInt(8034975744),
	}
)
