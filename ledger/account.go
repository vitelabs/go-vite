package ledger

import (
	"math/big"
	"github.com/vitelabs/go-vite/vitepb"
	"github.com/golang/protobuf/proto"
)

type AccountSimpleToken struct {
	TokenId []byte
	LastAccountBlockHeight *big.Int
}

type AccountMeta struct {
	AccountId *big.Int
	TokenList []*AccountSimpleToken
}

type Account struct {
	AccountMeta
	blockHeight *big.Int
}


// modify by sanjin
// has to be query from accountBlockMeta?????
//
func (account *Account) GetBlockHeight () *big.Int {
	//return big.NewInt(456)
	return account.blockHeight
}

func (am *AccountMeta) GetTokenList () []*AccountSimpleToken {
	return am.TokenList
}

func (am *AccountMeta) DbSerialize () ([]byte, error) {
	accountMetaPB := &vitepb.AccountMeta{
	}
	serializedBytes, err := proto.Marshal(accountMetaPB)

	if err != nil {
		return nil, err
	}
	return serializedBytes, nil
}


func (am *AccountMeta) DbDeserialize (buf []byte) error {
	accountMetaPB := &vitepb.AccountMeta{}
	if err := proto.Unmarshal(buf, accountMetaPB ); err != nil {
		return err
	}
	return nil
}