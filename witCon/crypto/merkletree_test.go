package crypto

import (
	"testing"
	"witCon/common"
	"witCon/log"
)

func TestMerkleTree(t *testing.T) {
	mt := &CompressMerkleTree{}
	log.Create("", "testExecute", 4)
	log.Init(2, 15)
	hashes := []common.Hash{Sha256([]byte{0x01}), common.Hash{}, Sha256([]byte{0x02}), Sha256([]byte{0x03}), Sha256([]byte{0x04}), Sha256([]byte{0x05})}
	for i := 0; i < 6; i++ {
		mt.WriteItem(hashes[i])
	}
	mt.CommitTree()
	mt.Print()
	r := mt.GetRoot()
	for i := 0; i < 6; i++ {
		pr := mt.MakeProof(uint16(i))
		r2 := pr.VerifyProof(hashes[i])
		t.Log(i, r.String(), r2.String())
		pr.Print()
		if r != r2 {
			t.Log("err!")
		}
	}
}

func TestMultiMerkleTree(t *testing.T) {
	mt := &MerkleTree{}
	log.Create("", "testExecute", 4)
	log.Init(2, 15)
	hashes := []common.Hash{Sha256([]byte{0x01}), common.Hash{}, common.Hash{}, common.Hash{},
		Sha256([]byte{0x04}), Sha256([]byte{0x05}), Sha256([]byte{0x06}), Sha256([]byte{0x07})}
	mt.MakeTree(hashes)
	mt.Print()
	r := mt.GetRoot()
	for i := 0; i < 8; i++ {
		pr := mt.MakeProof(uint16(i))
		r2 := pr.VerifyProof(hashes[i])
		t.Log(i, r.String(), r2.String())
		pr.Print()
		if r != r2 {
			t.Log("err!")
		}

		s := []bool{false, false, false, false, false, true, false, true}
		s[i] = true
		h := make([]common.Hash, 0)
		for index, b := range s {
			if b {
				h = append(h, hashes[index])
			}
		}
		mp := mt.MakeMultiProof(s)
		hash := mp.VerifyProof(h)

		t.Log(i, r.String(), hash.String())
		if r != hash {
			t.Log("err2!")
		}
	}
}
