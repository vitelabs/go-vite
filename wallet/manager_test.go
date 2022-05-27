package wallet_test

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vitelabs/go-vite/v2/common/config"
	"github.com/vitelabs/go-vite/v2/common/fileutils"
	"github.com/vitelabs/go-vite/v2/common/helper"
	"github.com/vitelabs/go-vite/v2/wallet"
)

func testManagerRecover(t *testing.T, dir string) {
	mneList := []string{
		"alter meat balance father season shop text figure pitch another fade figure faith chat smooth pottery dilemma pause differ equal shuffle series valve render",
		"that split virus bulk piece recall kick cave balance trigger burst license chat fame frog void theme soft unit subject crime tragic hip sand",
		"next aerobic ticket dragon real impulse unaware nut useful laundry forget prize ranch myth portion mail spare coast lonely lunar deer topic pill suspect",
		"twice catch reunion smooth impose predict device valid tobacco romance bind demand boy nest height toy pair salt journey bachelor choice siege setup hire",
		"orient ring dolphin metal arctic giraffe amazing great ticket genuine debate release night fit canvas fancy unknown powder burger window science health master marine",
		"twice nation bulb near fire wrap ensure gym panic color enhance zebra sail caught profit frequent process angle dad goddess jar plunge acid forward",
		"jungle south agent visa document inside sausage degree delay harbor idle sport moon cup pelican innocent bid winter gate blade faith check desert produce",
		"sound flock predict gorilla rhythm image regular ready speed hill globe thunder differ garage sustain vapor midnight arrive quiz tiger drive antique waste depend",
		"patch comic wife chair absurd tree skate win stage innocent anxiety solve spy bunker arrive actress blind ivory health sheriff hurdle enhance toss ensure",
		"sport coral praise boring shed object risk sick nominee render sunset boil aerobic gate genius spell attend tape ghost mercy myself cloud energy culture",
		"whip traffic alley rate frame digital carry survey amused picture cannon polar message lunch foil learn blossom adult together laptop smooth copy hub loop",
		"tobacco author base shift exit advice daughter unable famous twice tuna candy require carpet rocket price sea forget dog burden foster certain zero drop",
		"tide recycle razor cement keep liquid rebuild extend witness avocado era wool parade gravity that vessel blur angle bomb mechanic also prosper oak trick",
		"stereo arrive decline hockey ladder glory hip step toddler acoustic knee update oppose balcony stable various horn patrol click behave arch twice detail spare",
		"lecture weapon grief absorb road erupt call manage vessel rich lonely type wave adult glimpse before similar addict neither found sight finger friend visit",
		"dog depth grunt vault news mirror remind century illness rail main craft keep angle same trick dress brass vibrant voyage toss ceiling pumpkin fix",
		"bargain among length moral physical awkward face abstract wolf inhale nose what assault escape battle curious antenna proud express dismiss enrich lesson draw witness",
		"cash flower awesome describe style chunk expose dance figure same arrive foster blame leader bread dwarf timber random try pattern shove pattern tone antenna",
		"moment what learn beauty hover once fancy develop husband have someone patrol decide mouse total ritual gain minimum snake silk lake tragic bonus sister",
		"together perfect goddess fire broken strategy clog toe cat proud pupil enforce gaze nasty assist coin invest chat subject door theme toilet fitness lawsuit",
		"antenna donor silly valid priority runway fabric click weird need enroll ozone lottery shed blue narrow athlete coach fix drastic aware cruel depart that",
		"drive lobster pride frequent orbit citizen table thank build super seek shaft immense high hidden another sauce clever ensure miss spider sunset rotate key",
		"vibrant monitor example unhappy celery solve inject wire thank spatial suffer kick ship excite flower border erode clog chuckle seven despair chat desert daring",
		"oblige maid inch hamster joy talent poverty announce old return grass smile ginger hill delay evidence buyer curtain mutual any struggle squirrel skill whip",
		"remove protect wet couch moral slight slot virtual north where print chimney rack fresh barely angle hurdle scrub diet elder raise easily eager crisp",
	}

	manager := wallet.New(&config.Wallet{
		DataDir: dir,
	})
	err := manager.Start()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(mneList); i++ {
		_, err := manager.RecoverEntropyStoreFromMnemonic(mneList[i], "123456")
		if err != nil {
			panic(err)
		}
	}
	manager.Stop()
}

func TestManager_NewMnemonicAndSeedStore3(t *testing.T) {
	manager := wallet.New(&config.Wallet{
		DataDir: fileutils.CreateTempDir(),
	})
	for i := 1; i <= 5; i++ {
		_, _, err := manager.NewMnemonicAndEntropyStore("123456")
		if err != nil {
			t.Fatal(err)
		}
	}
}

func testManagerDerive(t *testing.T, dir string) {
	manager := wallet.New(&config.Wallet{
		DataDir: fileutils.CreateTempDir(),
	})
	err := manager.Start()
	if err != nil {
		t.Fatal(err)
	}
	files := manager.ListAllEntropyFiles()
	for _, v := range files {
		storeManager, err := manager.GetEntropyStoreManager(v)
		if err != nil {
			panic(err)
		}
		err = storeManager.Unlock("123456")
		if err != nil {
			t.Fatal(err)
		}
		_, key, err := storeManager.DeriveForIndexPath(0)
		if err != nil {
			panic(err)
		}
		keys, err := key.PrivateKey()
		if err != nil {
			panic(err)
		}
		addr, err := key.Address()
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s,0:%s,%s\n", addr, addr, keys.Hex())

		for i := 0; i < 25; i++ {
			_, key, err := storeManager.DeriveForIndexPath(uint32(i))
			helper.ErrFailf(t, err, "deriver for index %d err", i)
			address, _ := key.Address()
			fmt.Println(strconv.Itoa(i) + ":" + address.String())
		}
	}
	manager.Stop()
}

func TestManage(t *testing.T) {
	tmpDir := fileutils.CreateTempDir()
	testManagerRecover(t, tmpDir)
	testManagerDerive(t, tmpDir)
}

func TestRead(t *testing.T) {
	hexPath := "020000000d000000000000000000000000"

	byt, err := hex.DecodeString(hexPath)

	assert.NoError(t, err)

	t.Log(len(byt))
	length := uint8(byt[0])
	t.Log(length)
	var nums []uint32
	nums = append(nums, binary.BigEndian.Uint32(byt[1:5]))
	nums = append(nums, binary.BigEndian.Uint32(byt[6:10]))
	nums = append(nums, binary.BigEndian.Uint32(byt[11:15]))
	nums = append(nums, binary.BigEndian.Uint32(byt[16:20]))
	t.Log(nums)

	path := "m"

	for i := 0; i < int(length); i++ {
		path += fmt.Sprintf("/%d", nums[i])
	}

	t.Log(path)
}
