package quota

import (
	"errors"
	"math/big"
)

var (
	errGasUintOverflow = errors.New("gas uint64 overflow")
)

const (
	quotaForCreateContract uint64 = 800000 // Quota limit for create contract.
	quotaForSection        uint64 = 21000

	maxQuotaHeightGap uint64 = 3600 * 24 // Maximum Snapshot block height gap to gain quota by pledge.

	precForFloat uint = 18
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
)
