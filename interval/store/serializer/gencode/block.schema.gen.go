package gencode

import (
	"io"
	"time"
	"unsafe"
)

var (
	_ = unsafe.Sizeof(0)
	_ = io.ReadFull
	_ = time.Now()
)

type HashHeight struct {
	Hash   string
	Height uint64
}

func (d *HashHeight) Size() (s uint64) {

	{
		l := uint64(len(d.Hash))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	s += 8
	return
}
func (d *HashHeight) Marshal(buf []byte) ([]byte, error) {
	size := d.Size()
	{
		if uint64(cap(buf)) >= size {
			buf = buf[:size]
		} else {
			buf = make([]byte, size)
		}
	}
	i := uint64(0)

	{
		l := uint64(len(d.Hash))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+0] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+0] = byte(t)
			i++

		}
		copy(buf[i+0:], d.Hash)
		i += l
	}
	{

		buf[i+0+0] = byte(d.Height >> 0)

		buf[i+1+0] = byte(d.Height >> 8)

		buf[i+2+0] = byte(d.Height >> 16)

		buf[i+3+0] = byte(d.Height >> 24)

		buf[i+4+0] = byte(d.Height >> 32)

		buf[i+5+0] = byte(d.Height >> 40)

		buf[i+6+0] = byte(d.Height >> 48)

		buf[i+7+0] = byte(d.Height >> 56)

	}
	return buf[:i+8], nil
}

func (d *HashHeight) Unmarshal(buf []byte) (uint64, error) {
	i := uint64(0)

	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+0] & 0x7F)
			for buf[i+0]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+0]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.Hash = string(buf[i+0 : i+0+l])
		i += l
	}
	{

		d.Height = 0 | (uint64(buf[i+0+0]) << 0) | (uint64(buf[i+1+0]) << 8) | (uint64(buf[i+2+0]) << 16) | (uint64(buf[i+3+0]) << 24) | (uint64(buf[i+4+0]) << 32) | (uint64(buf[i+5+0]) << 40) | (uint64(buf[i+6+0]) << 48) | (uint64(buf[i+7+0]) << 56)

	}
	return i + 8, nil
}

type DBAccountBlock struct {
	Height         uint64
	Hash           string
	PreHash        string
	Signer         string
	Timestamp      int64
	Amount         int64
	ModifiedAmount int64
	SnapshotHeight uint64
	SnapshotHash   string
	BlockType      uint8
	From           string
	To             string
	Source         *HashHeight
}

func (d *DBAccountBlock) Size() (s uint64) {

	{
		l := uint64(len(d.Hash))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	{
		l := uint64(len(d.PreHash))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	{
		l := uint64(len(d.Signer))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	{
		l := uint64(len(d.SnapshotHash))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	{
		l := uint64(len(d.From))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	{
		l := uint64(len(d.To))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	{
		if d.Source != nil {

			{
				s += (*d.Source).Size()
			}
			s += 0
		}
	}
	s += 42
	return
}
func (d *DBAccountBlock) Marshal(buf []byte) ([]byte, error) {
	size := d.Size()
	{
		if uint64(cap(buf)) >= size {
			buf = buf[:size]
		} else {
			buf = make([]byte, size)
		}
	}
	i := uint64(0)

	{

		buf[0+0] = byte(d.Height >> 0)

		buf[1+0] = byte(d.Height >> 8)

		buf[2+0] = byte(d.Height >> 16)

		buf[3+0] = byte(d.Height >> 24)

		buf[4+0] = byte(d.Height >> 32)

		buf[5+0] = byte(d.Height >> 40)

		buf[6+0] = byte(d.Height >> 48)

		buf[7+0] = byte(d.Height >> 56)

	}
	{
		l := uint64(len(d.Hash))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+8] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+8] = byte(t)
			i++

		}
		copy(buf[i+8:], d.Hash)
		i += l
	}
	{
		l := uint64(len(d.PreHash))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+8] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+8] = byte(t)
			i++

		}
		copy(buf[i+8:], d.PreHash)
		i += l
	}
	{
		l := uint64(len(d.Signer))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+8] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+8] = byte(t)
			i++

		}
		copy(buf[i+8:], d.Signer)
		i += l
	}
	{

		buf[i+0+8] = byte(d.Timestamp >> 0)

		buf[i+1+8] = byte(d.Timestamp >> 8)

		buf[i+2+8] = byte(d.Timestamp >> 16)

		buf[i+3+8] = byte(d.Timestamp >> 24)

		buf[i+4+8] = byte(d.Timestamp >> 32)

		buf[i+5+8] = byte(d.Timestamp >> 40)

		buf[i+6+8] = byte(d.Timestamp >> 48)

		buf[i+7+8] = byte(d.Timestamp >> 56)

	}
	{

		buf[i+0+16] = byte(d.Amount >> 0)

		buf[i+1+16] = byte(d.Amount >> 8)

		buf[i+2+16] = byte(d.Amount >> 16)

		buf[i+3+16] = byte(d.Amount >> 24)

		buf[i+4+16] = byte(d.Amount >> 32)

		buf[i+5+16] = byte(d.Amount >> 40)

		buf[i+6+16] = byte(d.Amount >> 48)

		buf[i+7+16] = byte(d.Amount >> 56)

	}
	{

		buf[i+0+24] = byte(d.ModifiedAmount >> 0)

		buf[i+1+24] = byte(d.ModifiedAmount >> 8)

		buf[i+2+24] = byte(d.ModifiedAmount >> 16)

		buf[i+3+24] = byte(d.ModifiedAmount >> 24)

		buf[i+4+24] = byte(d.ModifiedAmount >> 32)

		buf[i+5+24] = byte(d.ModifiedAmount >> 40)

		buf[i+6+24] = byte(d.ModifiedAmount >> 48)

		buf[i+7+24] = byte(d.ModifiedAmount >> 56)

	}
	{

		buf[i+0+32] = byte(d.SnapshotHeight >> 0)

		buf[i+1+32] = byte(d.SnapshotHeight >> 8)

		buf[i+2+32] = byte(d.SnapshotHeight >> 16)

		buf[i+3+32] = byte(d.SnapshotHeight >> 24)

		buf[i+4+32] = byte(d.SnapshotHeight >> 32)

		buf[i+5+32] = byte(d.SnapshotHeight >> 40)

		buf[i+6+32] = byte(d.SnapshotHeight >> 48)

		buf[i+7+32] = byte(d.SnapshotHeight >> 56)

	}
	{
		l := uint64(len(d.SnapshotHash))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+40] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+40] = byte(t)
			i++

		}
		copy(buf[i+40:], d.SnapshotHash)
		i += l
	}
	{

		buf[i+0+40] = byte(d.BlockType >> 0)

	}
	{
		l := uint64(len(d.From))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+41] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+41] = byte(t)
			i++

		}
		copy(buf[i+41:], d.From)
		i += l
	}
	{
		l := uint64(len(d.To))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+41] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+41] = byte(t)
			i++

		}
		copy(buf[i+41:], d.To)
		i += l
	}
	{
		if d.Source == nil {
			buf[i+41] = 0
		} else {
			buf[i+41] = 1

			{
				nbuf, err := (*d.Source).Marshal(buf[i+42:])
				if err != nil {
					return nil, err
				}
				i += uint64(len(nbuf))
			}
			i += 0
		}
	}
	return buf[:i+42], nil
}

func (d *DBAccountBlock) Unmarshal(buf []byte) (uint64, error) {
	i := uint64(0)

	{

		d.Height = 0 | (uint64(buf[i+0+0]) << 0) | (uint64(buf[i+1+0]) << 8) | (uint64(buf[i+2+0]) << 16) | (uint64(buf[i+3+0]) << 24) | (uint64(buf[i+4+0]) << 32) | (uint64(buf[i+5+0]) << 40) | (uint64(buf[i+6+0]) << 48) | (uint64(buf[i+7+0]) << 56)

	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+8] & 0x7F)
			for buf[i+8]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+8]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.Hash = string(buf[i+8 : i+8+l])
		i += l
	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+8] & 0x7F)
			for buf[i+8]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+8]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.PreHash = string(buf[i+8 : i+8+l])
		i += l
	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+8] & 0x7F)
			for buf[i+8]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+8]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.Signer = string(buf[i+8 : i+8+l])
		i += l
	}
	{

		d.Timestamp = 0 | (int64(buf[i+0+8]) << 0) | (int64(buf[i+1+8]) << 8) | (int64(buf[i+2+8]) << 16) | (int64(buf[i+3+8]) << 24) | (int64(buf[i+4+8]) << 32) | (int64(buf[i+5+8]) << 40) | (int64(buf[i+6+8]) << 48) | (int64(buf[i+7+8]) << 56)

	}
	{

		d.Amount = 0 | (int64(buf[i+0+16]) << 0) | (int64(buf[i+1+16]) << 8) | (int64(buf[i+2+16]) << 16) | (int64(buf[i+3+16]) << 24) | (int64(buf[i+4+16]) << 32) | (int64(buf[i+5+16]) << 40) | (int64(buf[i+6+16]) << 48) | (int64(buf[i+7+16]) << 56)

	}
	{

		d.ModifiedAmount = 0 | (int64(buf[i+0+24]) << 0) | (int64(buf[i+1+24]) << 8) | (int64(buf[i+2+24]) << 16) | (int64(buf[i+3+24]) << 24) | (int64(buf[i+4+24]) << 32) | (int64(buf[i+5+24]) << 40) | (int64(buf[i+6+24]) << 48) | (int64(buf[i+7+24]) << 56)

	}
	{

		d.SnapshotHeight = 0 | (uint64(buf[i+0+32]) << 0) | (uint64(buf[i+1+32]) << 8) | (uint64(buf[i+2+32]) << 16) | (uint64(buf[i+3+32]) << 24) | (uint64(buf[i+4+32]) << 32) | (uint64(buf[i+5+32]) << 40) | (uint64(buf[i+6+32]) << 48) | (uint64(buf[i+7+32]) << 56)

	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+40] & 0x7F)
			for buf[i+40]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+40]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.SnapshotHash = string(buf[i+40 : i+40+l])
		i += l
	}
	{

		d.BlockType = 0 | (uint8(buf[i+0+40]) << 0)

	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+41] & 0x7F)
			for buf[i+41]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+41]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.From = string(buf[i+41 : i+41+l])
		i += l
	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+41] & 0x7F)
			for buf[i+41]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+41]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.To = string(buf[i+41 : i+41+l])
		i += l
	}
	{
		if buf[i+41] == 1 {
			if d.Source == nil {
				d.Source = new(HashHeight)
			}

			{
				ni, err := (*d.Source).Unmarshal(buf[i+42:])
				if err != nil {
					return 0, err
				}
				i += ni
			}
			i += 0
		} else {
			d.Source = nil
		}
	}
	return i + 42, nil
}

type AccountHashH struct {
	Hash   string
	Height uint64
	Addr   string
}

func (d *AccountHashH) Size() (s uint64) {

	{
		l := uint64(len(d.Hash))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	{
		l := uint64(len(d.Addr))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	s += 8
	return
}
func (d *AccountHashH) Marshal(buf []byte) ([]byte, error) {
	size := d.Size()
	{
		if uint64(cap(buf)) >= size {
			buf = buf[:size]
		} else {
			buf = make([]byte, size)
		}
	}
	i := uint64(0)

	{
		l := uint64(len(d.Hash))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+0] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+0] = byte(t)
			i++

		}
		copy(buf[i+0:], d.Hash)
		i += l
	}
	{

		buf[i+0+0] = byte(d.Height >> 0)

		buf[i+1+0] = byte(d.Height >> 8)

		buf[i+2+0] = byte(d.Height >> 16)

		buf[i+3+0] = byte(d.Height >> 24)

		buf[i+4+0] = byte(d.Height >> 32)

		buf[i+5+0] = byte(d.Height >> 40)

		buf[i+6+0] = byte(d.Height >> 48)

		buf[i+7+0] = byte(d.Height >> 56)

	}
	{
		l := uint64(len(d.Addr))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+8] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+8] = byte(t)
			i++

		}
		copy(buf[i+8:], d.Addr)
		i += l
	}
	return buf[:i+8], nil
}

func (d *AccountHashH) Unmarshal(buf []byte) (uint64, error) {
	i := uint64(0)

	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+0] & 0x7F)
			for buf[i+0]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+0]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.Hash = string(buf[i+0 : i+0+l])
		i += l
	}
	{

		d.Height = 0 | (uint64(buf[i+0+0]) << 0) | (uint64(buf[i+1+0]) << 8) | (uint64(buf[i+2+0]) << 16) | (uint64(buf[i+3+0]) << 24) | (uint64(buf[i+4+0]) << 32) | (uint64(buf[i+5+0]) << 40) | (uint64(buf[i+6+0]) << 48) | (uint64(buf[i+7+0]) << 56)

	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+8] & 0x7F)
			for buf[i+8]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+8]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.Addr = string(buf[i+8 : i+8+l])
		i += l
	}
	return i + 8, nil
}

type DBSnapshotBlock struct {
	Height    uint64
	Hash      string
	PreHash   string
	Signer    string
	Timestamp int64
	Accounts  []*AccountHashH
}

func (d *DBSnapshotBlock) Size() (s uint64) {

	{
		l := uint64(len(d.Hash))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	{
		l := uint64(len(d.PreHash))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	{
		l := uint64(len(d.Signer))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	{
		l := uint64(len(d.Accounts))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}

		for k0 := range d.Accounts {

			{
				if d.Accounts[k0] != nil {

					{
						s += (*d.Accounts[k0]).Size()
					}
					s += 0
				}
			}

			s += 1

		}

	}
	s += 16
	return
}
func (d *DBSnapshotBlock) Marshal(buf []byte) ([]byte, error) {
	size := d.Size()
	{
		if uint64(cap(buf)) >= size {
			buf = buf[:size]
		} else {
			buf = make([]byte, size)
		}
	}
	i := uint64(0)

	{

		buf[0+0] = byte(d.Height >> 0)

		buf[1+0] = byte(d.Height >> 8)

		buf[2+0] = byte(d.Height >> 16)

		buf[3+0] = byte(d.Height >> 24)

		buf[4+0] = byte(d.Height >> 32)

		buf[5+0] = byte(d.Height >> 40)

		buf[6+0] = byte(d.Height >> 48)

		buf[7+0] = byte(d.Height >> 56)

	}
	{
		l := uint64(len(d.Hash))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+8] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+8] = byte(t)
			i++

		}
		copy(buf[i+8:], d.Hash)
		i += l
	}
	{
		l := uint64(len(d.PreHash))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+8] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+8] = byte(t)
			i++

		}
		copy(buf[i+8:], d.PreHash)
		i += l
	}
	{
		l := uint64(len(d.Signer))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+8] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+8] = byte(t)
			i++

		}
		copy(buf[i+8:], d.Signer)
		i += l
	}
	{

		buf[i+0+8] = byte(d.Timestamp >> 0)

		buf[i+1+8] = byte(d.Timestamp >> 8)

		buf[i+2+8] = byte(d.Timestamp >> 16)

		buf[i+3+8] = byte(d.Timestamp >> 24)

		buf[i+4+8] = byte(d.Timestamp >> 32)

		buf[i+5+8] = byte(d.Timestamp >> 40)

		buf[i+6+8] = byte(d.Timestamp >> 48)

		buf[i+7+8] = byte(d.Timestamp >> 56)

	}
	{
		l := uint64(len(d.Accounts))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+16] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+16] = byte(t)
			i++

		}
		for k0 := range d.Accounts {

			{
				if d.Accounts[k0] == nil {
					buf[i+16] = 0
				} else {
					buf[i+16] = 1

					{
						nbuf, err := (*d.Accounts[k0]).Marshal(buf[i+17:])
						if err != nil {
							return nil, err
						}
						i += uint64(len(nbuf))
					}
					i += 0
				}
			}

			i += 1

		}
	}
	return buf[:i+16], nil
}

func (d *DBSnapshotBlock) Unmarshal(buf []byte) (uint64, error) {
	i := uint64(0)

	{

		d.Height = 0 | (uint64(buf[i+0+0]) << 0) | (uint64(buf[i+1+0]) << 8) | (uint64(buf[i+2+0]) << 16) | (uint64(buf[i+3+0]) << 24) | (uint64(buf[i+4+0]) << 32) | (uint64(buf[i+5+0]) << 40) | (uint64(buf[i+6+0]) << 48) | (uint64(buf[i+7+0]) << 56)

	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+8] & 0x7F)
			for buf[i+8]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+8]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.Hash = string(buf[i+8 : i+8+l])
		i += l
	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+8] & 0x7F)
			for buf[i+8]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+8]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.PreHash = string(buf[i+8 : i+8+l])
		i += l
	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+8] & 0x7F)
			for buf[i+8]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+8]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.Signer = string(buf[i+8 : i+8+l])
		i += l
	}
	{

		d.Timestamp = 0 | (int64(buf[i+0+8]) << 0) | (int64(buf[i+1+8]) << 8) | (int64(buf[i+2+8]) << 16) | (int64(buf[i+3+8]) << 24) | (int64(buf[i+4+8]) << 32) | (int64(buf[i+5+8]) << 40) | (int64(buf[i+6+8]) << 48) | (int64(buf[i+7+8]) << 56)

	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+16] & 0x7F)
			for buf[i+16]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+16]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		if uint64(cap(d.Accounts)) >= l {
			d.Accounts = d.Accounts[:l]
		} else {
			d.Accounts = make([]*AccountHashH, l)
		}
		for k0 := range d.Accounts {

			{
				if buf[i+16] == 1 {
					if d.Accounts[k0] == nil {
						d.Accounts[k0] = new(AccountHashH)
					}

					{
						ni, err := (*d.Accounts[k0]).Unmarshal(buf[i+17:])
						if err != nil {
							return 0, err
						}
						i += ni
					}
					i += 0
				} else {
					d.Accounts[k0] = nil
				}
			}

			i += 1

		}
	}
	return i + 16, nil
}
