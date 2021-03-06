package storage

import (
	"log"
	"sync"

	"github.com/coreos/mantle/Godeps/_workspace/src/github.com/google/btree"
)

type index interface {
	Get(key []byte, atRev int64) (rev, created revision, ver int64, err error)
	Range(key, end []byte, atRev int64) ([][]byte, []revision)
	Put(key []byte, rev revision)
	Restore(key []byte, created, modified revision, ver int64)
	Tombstone(key []byte, rev revision) error
	Compact(rev int64) map[revision]struct{}
	Equal(b index) bool
}

type treeIndex struct {
	sync.RWMutex
	tree *btree.BTree
}

func newTreeIndex() index {
	return &treeIndex{
		tree: btree.New(32),
	}
}

func (ti *treeIndex) Put(key []byte, rev revision) {
	keyi := &keyIndex{key: key}

	ti.Lock()
	defer ti.Unlock()
	item := ti.tree.Get(keyi)
	if item == nil {
		keyi.put(rev.main, rev.sub)
		ti.tree.ReplaceOrInsert(keyi)
		return
	}
	okeyi := item.(*keyIndex)
	okeyi.put(rev.main, rev.sub)
}

func (ti *treeIndex) Restore(key []byte, created, modified revision, ver int64) {
	keyi := &keyIndex{key: key}

	ti.Lock()
	defer ti.Unlock()
	item := ti.tree.Get(keyi)
	if item == nil {
		keyi.restore(created, modified, ver)
		ti.tree.ReplaceOrInsert(keyi)
		return
	}
	okeyi := item.(*keyIndex)
	okeyi.put(modified.main, modified.sub)
}

func (ti *treeIndex) Get(key []byte, atRev int64) (modified, created revision, ver int64, err error) {
	keyi := &keyIndex{key: key}

	ti.RLock()
	defer ti.RUnlock()
	item := ti.tree.Get(keyi)
	if item == nil {
		return revision{}, revision{}, 0, ErrRevisionNotFound
	}

	keyi = item.(*keyIndex)
	return keyi.get(atRev)
}

func (ti *treeIndex) Range(key, end []byte, atRev int64) (keys [][]byte, revs []revision) {
	if end == nil {
		rev, _, _, err := ti.Get(key, atRev)
		if err != nil {
			return nil, nil
		}
		return [][]byte{key}, []revision{rev}
	}

	keyi := &keyIndex{key: key}
	endi := &keyIndex{key: end}

	ti.RLock()
	defer ti.RUnlock()

	ti.tree.AscendGreaterOrEqual(keyi, func(item btree.Item) bool {
		if !item.Less(endi) {
			return false
		}
		curKeyi := item.(*keyIndex)
		rev, _, _, err := curKeyi.get(atRev)
		if err != nil {
			return true
		}
		revs = append(revs, rev)
		keys = append(keys, curKeyi.key)
		return true
	})

	return keys, revs
}

func (ti *treeIndex) Tombstone(key []byte, rev revision) error {
	keyi := &keyIndex{key: key}

	ti.Lock()
	defer ti.Unlock()
	item := ti.tree.Get(keyi)
	if item == nil {
		return ErrRevisionNotFound
	}

	ki := item.(*keyIndex)
	return ki.tombstone(rev.main, rev.sub)
}

func (ti *treeIndex) Compact(rev int64) map[revision]struct{} {
	available := make(map[revision]struct{})
	emptyki := make([]*keyIndex, 0)
	log.Printf("store.index: compact %d", rev)
	// TODO: do not hold the lock for long time?
	// This is probably OK. Compacting 10M keys takes O(10ms).
	ti.Lock()
	defer ti.Unlock()
	ti.tree.Ascend(compactIndex(rev, available, &emptyki))
	for _, ki := range emptyki {
		item := ti.tree.Delete(ki)
		if item == nil {
			log.Panic("store.index: unexpected delete failure during compaction")
		}
	}
	return available
}

func compactIndex(rev int64, available map[revision]struct{}, emptyki *[]*keyIndex) func(i btree.Item) bool {
	return func(i btree.Item) bool {
		keyi := i.(*keyIndex)
		keyi.compact(rev, available)
		if keyi.isEmpty() {
			*emptyki = append(*emptyki, keyi)
		}
		return true
	}
}

func (a *treeIndex) Equal(bi index) bool {
	b := bi.(*treeIndex)

	if a.tree.Len() != b.tree.Len() {
		return false
	}

	equal := true

	a.tree.Ascend(func(item btree.Item) bool {
		aki := item.(*keyIndex)
		bki := b.tree.Get(item).(*keyIndex)
		if !aki.equal(bki) {
			equal = false
			return false
		}
		return true
	})

	return equal
}
