package quota

import (
	"errors"
	"math/big"
)

var (
	ErrOutOfQuota      = errors.New("out of quota")
	errGasUintOverflow = errors.New("gas uint64 overflow")

	quotaByPledge = big.NewInt(1e18)
)

const (
	quotaForPoW            uint64 = 21000
	quotaLimit             uint64 = 1000000 // Maximum quota of an account referring to one snapshot block.
	quotaForCreateContract uint64 = 800000  // Quota limit for create contract.
	quotaForSection        uint64 = 21000
	TxGas                  uint64 = 21000     // Per transaction not creating a contract.
	txContractCreationGas  uint64 = 53000     // Per transaction that creates a contract.
	txDataZeroGas          uint64 = 4         // Per byte of data attached to a transaction that equals zero.
	txDataNonZeroGas       uint64 = 68        // Per byte of data attached to a transaction that is not equal to zero.
	maxQuotaHeightGap      uint64 = 3600 * 24 // Maximum Snapshot block height gap to gain quota by pledge.

	precForFloat uint = 18
)

var (
	paramA      = new(big.Float).SetPrec(precForFloat).SetFloat64(4.861825883582755e-26)
	paramB      = new(big.Float).SetPrec(precForFloat).SetFloat64(2.277159377950245e-21)
	sectionList = []*big.Float{ // Section list of x value in e**x
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.0),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.042006175634155006),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.08404944434245186),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.1261670961035256),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.16839681732546105),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.2107768956769977),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.25334643304410037),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.2961455696376917),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.3392157225669637),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.382599842575369),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.4263426931297194),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.4704911566788094),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.5150945736855665),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.5602051210238872),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.605878237567604),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.6521731063496397),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.6991532046201573),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.7468869355972497),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.7954483588344243),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.8449180401302736),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.8953840470548413),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.9469431228444231),
		new(big.Float).SetPrec(precForFloat).SetFloat64(0.9997020801479394),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.053779467629503),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.1093075777848576),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.1664348850068706),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.2253290311060194),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.286180514353531),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.3492072924575544),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.4146605870070175),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.4828322881625378),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.554064521717701),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.6287621852605034),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.707409634545938),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.7905932883378723),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.8790328663947373),
		new(big.Float).SetPrec(precForFloat).SetFloat64(1.97362554890186),
		new(big.Float).SetPrec(precForFloat).SetFloat64(2.0755100566945326),
		new(big.Float).SetPrec(precForFloat).SetFloat64(2.186162517630361),
		new(big.Float).SetPrec(precForFloat).SetFloat64(2.3075451472522963),
		new(big.Float).SetPrec(precForFloat).SetFloat64(2.4423470353692043),
		new(big.Float).SetPrec(precForFloat).SetFloat64(2.594395323511559),
		new(big.Float).SetPrec(precForFloat).SetFloat64(2.7694056956796604),
		new(big.Float).SetPrec(precForFloat).SetFloat64(2.976475888792767),
		new(big.Float).SetPrec(precForFloat).SetFloat64(3.2314282909393213),
		new(big.Float).SetPrec(precForFloat).SetFloat64(3.5656840708200748),
		new(big.Float).SetPrec(precForFloat).SetFloat64(4.057395776090949),
		new(big.Float).SetPrec(precForFloat).SetFloat64(5.029431885090279),
	}
)
