package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	chain_block "github.com/vitelabs/go-vite/v2/ledger/chain/block"
	chain_file_manager "github.com/vitelabs/go-vite/v2/ledger/chain/file_manager"
)

type blocks struct {
	blockDb *chain_block.BlockDB
	meta    *blocksMeta
	dir     string
}

func newBlocks(dir string, filesize int64) (*blocks, error) {
	bs := &blocks{}
	bs.dir = dir
	blockDb, err := chain_block.NewBlockDBFixedSize(bs.dir, filesize)
	if err != nil {
		return nil, err
	}
	bs.blockDb = blockDb

	bs.meta = &blocksMeta{dir: dir}

	if err := bs.meta.load(); err != nil {
		if err = bs.meta.generateMeta(bs.blockDb); err != nil {
			log.Warn("rebuild the block meta info error", "err", err)
			return nil, err
		}
	}
	return bs, nil
}

func (bs blocks) location(height uint64) (*chain_file_manager.Location, error) {
	start, _ := bs.meta.hit(height)
	cur := start.Location
	for {
		chunk, location, err := bs.blockDb.ReadChunk(cur)
		if err != nil {
			return nil, fmt.Errorf("read chunk error, %s", err.Error())
		}
		if chunk.SnapshotBlock.Height == height {
			return &cur, nil
		} else if chunk.SnapshotBlock.Height > height {
			return nil, errors.New("strange error")
		}
		cur = *location
	}
}

type heightMeta struct {
	chain_file_manager.Location
	Height uint64
}

type blocksMeta struct {
	heights []heightMeta
	dir     string
}

func (bMeta *blocksMeta) load() error {
	// Open our jsonFile
	jsonFile, err := os.Open(bMeta.dir + "/meta")
	// if we os.Open returns an error then handle it
	if err != nil {
		return err
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	// read our opened jsonFile as a byte array.
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	// we initialize our Users array
	var heights []heightMeta

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	err = json.Unmarshal(byteValue, &heights)
	if err != nil {
		return err
	}
	bMeta.heights = heights
	return nil
}

func (bMeta *blocksMeta) generateMeta(blockDb *chain_block.BlockDB) error {
	i := uint64(0)
	step := uint64(1000)

	var heights []heightMeta

	current := &chain_file_manager.Location{
		FileId: 1,
		Offset: 0,
	}
	log.Info("rebuild the block meta info...")
	for {
		i++
		if i%step == 0 {
			sBlock, accBlocks, location, err := blockDb.ReadUnit(current)
			if err != nil {
				return err
			}
			if sBlock == nil && accBlocks == nil {
				break
			}
			if sBlock == nil {
				current = location
				continue
			}
			heights = append(heights, heightMeta{
				Location: *location,
				Height:   sBlock.Height + 1,
			})
			log.Info(fmt.Sprintf("rebuild height %d", sBlock.Height))
			current = location
		} else {
			location, err := blockDb.GetNextLocation(current)

			if err != nil {
				return err
			}
			if location == nil {
				break
			}
			current = location
		}
	}
	bMeta.heights = heights

	byt, _ := json.Marshal(heights)
	err := ioutil.WriteFile(bMeta.dir+"/meta", byt, 0644)
	return err
}

func (bMeta blocksMeta) hit(height uint64) (start, end heightMeta) {
	start = heightMeta{
		Location: chain_file_manager.Location{
			FileId: 1,
			Offset: 0,
		},
		Height: 1,
	}
	for _, tmp := range bMeta.heights {
		if tmp.Height < height {
			start = tmp
		}
		if tmp.Height >= height && end.Height == 0 {
			end = tmp
		}
	}
	return start, end
}
