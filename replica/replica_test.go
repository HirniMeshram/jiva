/*
 Copyright © 2020 The OpenEBS Authors

 This file was originally authored by Rancher Labs
 under Apache License 2018.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package replica

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	inject "github.com/openebs/jiva/error-inject"
	"github.com/openebs/jiva/types"

	"github.com/openebs/jiva/util"
	. "gopkg.in/check.v1"
)

const (
	b  = 4096
	bs = 512
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func (s *TestSuite) TestCreate(c *C) {
	dir, err := ioutil.TempDir("", "replica")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := New(true, 9, 3, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)
}

func getNow() string {
	// Make sure timestamp is unique
	time.Sleep(1 * time.Second)
	return util.Now()
}

func (s *TestSuite) TestSnapshot(c *C) {
	dir, err := ioutil.TempDir("", "replica")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := New(true, 9, 3, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)

	createdTime0 := getNow()

	err = r.Snapshot("000", true, createdTime0)
	c.Assert(err, IsNil)

	createdTime1 := getNow()
	err = r.Snapshot("001", true, createdTime1)
	c.Assert(err, IsNil)

	c.Assert(len(r.activeDiskData), Equals, 4)
	c.Assert(len(r.volume.files), Equals, 4)

	c.Assert(r.info.Head, Equals, "volume-head-002.img")
	c.Assert(r.activeDiskData[3].Name, Equals, "volume-head-002.img")
	c.Assert(r.activeDiskData[3].UserCreated, Equals, false)
	c.Assert(r.activeDiskData[3].Parent, Equals, "volume-snap-001.img")
	c.Assert(r.activeDiskData[3].Created, Equals, createdTime1)

	c.Assert(r.activeDiskData[2].Name, Equals, "volume-snap-001.img")
	c.Assert(r.activeDiskData[2].UserCreated, Equals, true)
	c.Assert(r.activeDiskData[2].Parent, Equals, "volume-snap-000.img")
	c.Assert(r.activeDiskData[2].Created, Equals, createdTime1)

	c.Assert(r.activeDiskData[1].Name, Equals, "volume-snap-000.img")
	c.Assert(r.activeDiskData[1].UserCreated, Equals, true)
	c.Assert(r.activeDiskData[1].Parent, Equals, "")
	c.Assert(r.activeDiskData[1].Created, Equals, createdTime0)

	c.Assert(len(r.diskData), Equals, 3)
	c.Assert(r.diskData["volume-snap-000.img"].Parent, Equals, "")
	c.Assert(r.diskData["volume-snap-000.img"].UserCreated, Equals, true)
	c.Assert(r.diskData["volume-snap-000.img"].Created, Equals, createdTime0)
	c.Assert(r.diskData["volume-snap-001.img"].Parent, Equals, "volume-snap-000.img")
	c.Assert(r.diskData["volume-snap-001.img"].UserCreated, Equals, true)
	c.Assert(r.diskData["volume-snap-001.img"].Created, Equals, createdTime1)
	c.Assert(r.diskData["volume-head-002.img"].Parent, Equals, "volume-snap-001.img")
	c.Assert(r.diskData["volume-head-002.img"].UserCreated, Equals, false)
	c.Assert(r.diskData["volume-head-002.img"].Created, Equals, createdTime1)

	c.Assert(len(r.diskChildrenMap), Equals, 3)
	c.Assert(len(r.diskChildrenMap["volume-snap-000.img"]), Equals, 1)
	c.Assert(r.diskChildrenMap["volume-snap-000.img"]["volume-snap-001.img"], Equals, true)
	c.Assert(len(r.diskChildrenMap["volume-snap-001.img"]), Equals, 1)
	c.Assert(r.diskChildrenMap["volume-snap-001.img"]["volume-head-002.img"], Equals, true)
	c.Assert(r.diskChildrenMap["volume-head-002.img"], IsNil)

	disks := r.ListDisks()
	c.Assert(len(disks), Equals, 3)
	c.Assert(disks["volume-snap-000.img"].Parent, Equals, "")
	c.Assert(disks["volume-snap-000.img"].UserCreated, Equals, true)
	c.Assert(disks["volume-snap-000.img"].Removed, Equals, false)
	c.Assert(len(disks["volume-snap-000.img"].Children), Equals, 1)
	c.Assert(disks["volume-snap-000.img"].Children[0], Equals, "volume-snap-001.img")
	c.Assert(disks["volume-snap-000.img"].Created, Equals, createdTime0)
	c.Assert(disks["volume-snap-000.img"].Size, Equals, "0")

	c.Assert(disks["volume-snap-001.img"].Parent, Equals, "volume-snap-000.img")
	c.Assert(disks["volume-snap-001.img"].UserCreated, Equals, true)
	c.Assert(disks["volume-snap-001.img"].Removed, Equals, false)
	c.Assert(len(disks["volume-snap-001.img"].Children), Equals, 1)
	c.Assert(disks["volume-snap-001.img"].Children[0], Equals, "volume-head-002.img")
	c.Assert(disks["volume-snap-001.img"].Created, Equals, createdTime1)
	c.Assert(disks["volume-snap-001.img"].Size, Equals, "0")

	c.Assert(disks["volume-head-002.img"].Parent, Equals, "volume-snap-001.img")
	c.Assert(disks["volume-head-002.img"].UserCreated, Equals, false)
	c.Assert(disks["volume-head-002.img"].Removed, Equals, false)
	c.Assert(len(disks["volume-head-002.img"].Children), Equals, 0)
	c.Assert(disks["volume-head-002.img"].Created, Equals, createdTime1)
	c.Assert(disks["volume-head-002.img"].Size, Equals, "0")
}

func (s *TestSuite) TestFileAlreadyExists(c *C) {
	dir, err := ioutil.TempDir("", "replica")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := New(true, 9, 3, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)
	now := getNow()
	f, err := os.Create(dir + "/volume-head-001.img")
	c.Assert(err, IsNil)
	f.Close()
	// head already exists, and will be deleted
	err = r.Snapshot("000", true, now)
	c.Assert(err, IsNil)

	// Snapshot already exists, linkDisk err
	err = r.Snapshot("000", true, now)
	c.Assert(err, NotNil)

	f, err = os.Create(dir + "/volume-head-002.img")
	c.Assert(err, IsNil)
	buf := make([]byte, 9)
	fill(buf, 9)
	n, err := f.Write(buf)
	c.Assert(n, Equals, 9)
	c.Assert(err, IsNil)
	f.Close()

	// head already exists, and it has some data
	err = r.Snapshot("001", true, now)
	c.Assert(err, NotNil)

	f, err = os.Create(dir + "/volume-snap-001.img.meta")
	c.Assert(err, IsNil)
	f.Close()

	// linkDisk error metafile already exists
	err = r.Snapshot("001", true, now)
	c.Assert(err, NotNil)
}

func (s *TestSuite) TestRmdisk(c *C) {
	dir, err := ioutil.TempDir("", "replica")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := New(true, 9, 3, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)
	now := getNow()
	err = r.Snapshot("000", true, now)
	c.Assert(err, IsNil)

	err = r.Snapshot("001", true, now)
	c.Assert(err, IsNil)

	err = r.rmDisk("volume-snap-000")
	c.Assert(err, IsNil)

	// snap-002 does not exist
	err = r.rmDisk("volume-snap-002")
	c.Assert(err, IsNil)

}

func (s *TestSuite) TestRemoveMiddle(c *C) {
	dir, err := ioutil.TempDir("", "replica")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := New(true, 9, 3, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)

	now := getNow()
	err = r.Snapshot("000", true, now)
	c.Assert(err, IsNil)

	err = r.Snapshot("001", true, now)
	c.Assert(err, IsNil)

	c.Assert(len(r.activeDiskData), Equals, 4)
	c.Assert(len(r.volume.files), Equals, 4)

	c.Assert(r.info.Head, Equals, "volume-head-002.img")
	c.Assert(r.activeDiskData[3].Name, Equals, "volume-head-002.img")
	c.Assert(r.activeDiskData[3].Parent, Equals, "volume-snap-001.img")
	c.Assert(r.activeDiskData[2].Name, Equals, "volume-snap-001.img")
	c.Assert(r.activeDiskData[2].Parent, Equals, "volume-snap-000.img")
	c.Assert(r.activeDiskData[1].Name, Equals, "volume-snap-000.img")
	c.Assert(r.activeDiskData[1].Parent, Equals, "")

	r.holeDrainer = func() {}
	err = r.RemoveDiffDisk("volume-snap-001.img")
	c.Assert(err, NotNil)
	c.Assert(len(r.activeDiskData), Equals, 4)
	c.Assert(len(r.volume.files), Equals, 4)
	c.Assert(r.info.Head, Equals, "volume-head-002.img")
	c.Assert(r.activeDiskData[3].Name, Equals, "volume-head-002.img")
	c.Assert(r.activeDiskData[3].Parent, Equals, "volume-snap-001.img")
	c.Assert(r.activeDiskData[2].Name, Equals, "volume-snap-001.img")
	c.Assert(r.activeDiskData[2].Parent, Equals, "volume-snap-000.img")
	c.Assert(r.activeDiskData[1].Name, Equals, "volume-snap-000.img")
	c.Assert(r.activeDiskData[1].Parent, Equals, "")

	c.Assert(len(r.diskData), Equals, 3)
	c.Assert(r.diskData["volume-snap-000.img"].Parent, Equals, "")
	c.Assert(r.diskData["volume-head-002.img"].Parent, Equals, "volume-snap-001.img")

	c.Assert(len(r.diskChildrenMap["volume-snap-000.img"]), Equals, 1)
	c.Assert(r.diskChildrenMap["volume-head-002.img"], IsNil)
}

func (s *TestSuite) TestPrepareRemove(c *C) {
	dir, err := ioutil.TempDir("", "replica")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := New(true, 9, 3, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)

	now := getNow()
	err = r.Snapshot("000", true, now)
	c.Assert(err, IsNil)

	err = r.Snapshot("001", true, now)
	c.Assert(err, IsNil)

	c.Assert(len(r.activeDiskData), Equals, 4)
	c.Assert(len(r.volume.files), Equals, 4)

	/*
		volume-snap-000.img
		volume-snap-001.img
		volume-head-002.img
	*/

	actions, err := r.PrepareRemoveDisk("001")
	c.Assert(err, NotNil)
	c.Assert(actions, HasLen, 0)
	c.Assert(r.activeDiskData[2].Removed, Equals, false)

	actions, err = r.PrepareRemoveDisk("volume-snap-000.img")
	c.Assert(err, NotNil)
	c.Assert(actions, HasLen, 0)
	c.Assert(r.activeDiskData[1].Removed, Equals, false)

	err = r.Snapshot("002", true, now)
	c.Assert(err, IsNil)

	/*
		volume-snap-000.img
		volume-snap-001.img
		volume-snap-002.img
		volume-head-003.img
	*/

	c.Assert(len(r.activeDiskData), Equals, 5)
	c.Assert(len(r.volume.files), Equals, 5)

	/* https://github.com/openebs/jiva/issues/184 */
	actions, err = r.PrepareRemoveDisk("002")
	c.Assert(err, NotNil)
	c.Assert(actions, HasLen, 0)

	err = r.Snapshot("003", true, now)
	c.Assert(err, IsNil)

	err = r.Snapshot("004", true, now)
	c.Assert(err, IsNil)

	/*
		volume-snap-000.img
		volume-snap-001.img
		volume-snap-002.img
		volume-snap-003.img
		volume-snap-004.img
		volume-head-005.img
	*/

	c.Assert(len(r.activeDiskData), Equals, 7)
	c.Assert(len(r.volume.files), Equals, 7)

	actions, err = r.PrepareRemoveDisk("003")
	c.Assert(err, IsNil)
	c.Assert(actions, HasLen, 2)
	c.Assert(actions[0].Action, Equals, OpCoalesce)
	c.Assert(actions[0].Source, Equals, "volume-snap-003.img")
	c.Assert(actions[0].Target, Equals, "volume-snap-002.img")
	c.Assert(actions[1].Action, Equals, OpRemove)
	c.Assert(actions[1].Source, Equals, "volume-snap-003.img")
	c.Assert(r.activeDiskData[4].Removed, Equals, true)
}

func byteEquals(c *C, expected, obtained []byte) {
	c.Assert(len(expected), Equals, len(obtained))

	for i := range expected {
		l := fmt.Sprintf("%d=%x", i, expected[i])
		r := fmt.Sprintf("%d=%x", i, obtained[i])
		c.Assert(r, Equals, l)
	}
}

func byteEqualsLocation(c *C, expected, obtained []uint16) {
	c.Assert(len(expected), Equals, len(obtained))

	for i := range expected {
		l := fmt.Sprintf("%d=%x", i, expected[i])
		r := fmt.Sprintf("%d=%x", i, obtained[i])
		c.Assert(r, Equals, l)
	}
}

func md5Equals(c *C, expected, obtained []byte) {
	c.Assert(len(expected), Equals, len(obtained))

	expectedMd5 := md5.Sum(expected)
	obtainedMd5 := md5.Sum(obtained)
	for i := range expectedMd5 {
		l := fmt.Sprintf("%d=%x", i, expectedMd5[i])
		r := fmt.Sprintf("%d=%x", i, obtainedMd5[i])
		c.Assert(r, Equals, l)
	}
}
func fill(buf []byte, val byte) {
	for i := 0; i < len(buf); i++ {
		buf[i] = val
	}
}

func (s *TestSuite) TestRead(c *C) {
	dir, err := ioutil.TempDir("", "replica")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := New(true, 9*b, b, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)

	buf := make([]byte, 3*b)
	_, err = r.ReadAt(buf, 0)
	c.Assert(err, IsNil)
	byteEquals(c, buf, make([]byte, 3*b))
}

func (s *TestSuite) TestWrite(c *C) {
	dir, err := ioutil.TempDir("", "replica")
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := New(true, 9*b, b, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)

	buf := make([]byte, 9*b)
	fill(buf, 1)
	_, err = r.WriteAt(buf, 0)
	c.Assert(err, IsNil)

	readBuf := make([]byte, 9*b)
	_, err = r.ReadAt(readBuf, 0)
	c.Assert(err, IsNil)

	byteEquals(c, readBuf, buf)
}

func (s *TestSuite) TestSnapshotReadWrite(c *C) {
	dir, err := ioutil.TempDir("", "replica")
	c.Logf("Volume: %s", dir)
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := New(true, 3*b, b, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)

	buf := make([]byte, 3*b)
	fill(buf, 3)
	count, err := r.WriteAt(buf, 0)
	c.Assert(err, IsNil)
	c.Assert(count, Equals, 3*b)
	err = r.Snapshot("000", true, getNow())
	c.Assert(err, IsNil)

	fill(buf[b:2*b], 2)
	count, err = r.WriteAt(buf[b:2*b], b)
	c.Assert(count, Equals, b)
	err = r.Snapshot("001", true, getNow())
	c.Assert(err, IsNil)

	fill(buf[:b], 1)
	count, err = r.WriteAt(buf[:b], 0)
	c.Assert(count, Equals, b)
	err = r.Snapshot("002", true, getNow())
	c.Assert(err, IsNil)

	readBuf := make([]byte, 3*b)
	_, err = r.ReadAt(readBuf, 0)
	c.Logf("%v", r.volume.location)
	c.Assert(err, IsNil)
	byteEquals(c, readBuf, buf)
	byteEqualsLocation(c, r.volume.location, []uint16{3, 2, 1})

	r, err = r.Reload(true)
	c.Assert(err, IsNil)

	_, err = r.ReadAt(readBuf, 0)
	c.Assert(err, IsNil)
	byteEquals(c, readBuf, buf)
	byteEqualsLocation(c, r.volume.location, []uint16{3, 2, 1})
}

func (s *TestSuite) TestBackingFile(c *C) {
	dir, err := ioutil.TempDir("", "replica")
	c.Logf("Volume: %s", dir)
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	buf := make([]byte, 3*b)
	fill(buf, 3)

	f, err := os.Create(path.Join(dir, "backing"))
	c.Assert(err, IsNil)
	defer f.Close()
	_, err = f.Write(buf)
	c.Assert(err, IsNil)

	backing := &BackingFile{
		Name: "backing",
		Disk: f,
	}

	r, err := New(true, 3*b, b, dir, backing, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)

	chain, err := r.Chain()
	c.Assert(err, IsNil)
	c.Assert(len(chain), Equals, 1)
	c.Assert(chain[0], Equals, "volume-head-000.img")

	newBuf := make([]byte, 1*b)
	_, err = r.WriteAt(newBuf, b)
	c.Assert(err, IsNil)

	newBuf2 := make([]byte, 3*b)
	fill(newBuf2, 3)
	fill(newBuf2[b:2*b], 0)

	_, err = r.ReadAt(buf, 0)
	c.Assert(err, IsNil)

	byteEquals(c, buf, newBuf2)
}

func (s *TestSuite) partialWriteRead(c *C, totalLength, writeLength, writeOffset int64) {
	fmt.Println("Starting partialWriteRead")
	dir, err := ioutil.TempDir("", "replica")
	c.Logf("Volume: %s", dir)
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	buf := make([]byte, totalLength)
	fill(buf, 3)

	r, err := New(true, totalLength, b, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)

	_, err = r.WriteAt(buf, 0)
	c.Assert(err, IsNil)

	err = r.Snapshot("000", true, getNow())
	c.Assert(err, IsNil)

	buf = make([]byte, writeLength)
	fill(buf, 1)

	_, err = r.WriteAt(buf, writeOffset)
	c.Assert(err, IsNil)

	buf = make([]byte, totalLength)
	_, err = r.ReadAt(buf, 0)
	c.Assert(err, IsNil)

	expected := make([]byte, totalLength)
	fill(expected, 3)
	fill(expected[writeOffset:writeOffset+writeLength], 1)

	byteEquals(c, expected, buf)
}

func (s *TestSuite) TestPartialWriteRead(c *C) {
	s.partialWriteRead(c, 3*b, 3*bs, 2*bs)
	s.partialWriteRead(c, 3*b, 3*bs, 21*bs)

	s.partialWriteRead(c, 3*b, 11*bs, 7*bs)
	s.partialWriteRead(c, 4*b, 19*bs, 7*bs)

	s.partialWriteRead(c, 3*b, 19*bs, 5*bs)
	s.partialWriteRead(c, 3*b, 19*bs, 0*bs)
}

func (s *TestSuite) testPartialRead(c *C, totalLength int64, readBuf []byte, offset int64) (int, error) {
	fmt.Println("Filling data for partialRead")
	dir, err := ioutil.TempDir("", "replica")
	fmt.Printf("Volume: %s\n", dir)
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	buf := make([]byte, totalLength)
	fill(buf, 3)

	r, err := New(true, totalLength, b, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("RW")
	c.Assert(err, IsNil)

	for i := int64(0); i < totalLength; i += b {
		buf := make([]byte, totalLength-i)
		fill(buf, byte(i/b+1))
		err := r.Snapshot(strconv.Itoa(int(i)), true, getNow())
		c.Assert(err, IsNil)
		_, err = r.WriteAt(buf, i)
		c.Assert(err, IsNil)
	}

	fmt.Println("Starting partialRead", r.volume.location)
	return r.ReadAt(readBuf, offset)
}

func (s *TestSuite) TestPartialRead(c *C) {
	buf := make([]byte, b)
	_, err := s.testPartialRead(c, 3*b, buf, b/2)
	c.Assert(err, IsNil)

	expected := make([]byte, b)
	fill(expected[:b/2], 1)
	fill(expected[b/2:], 2)

	byteEquals(c, expected, buf)
}

func (s *TestSuite) TestPartialReadZeroStartOffset(c *C) {
	buf := make([]byte, b+b/2)
	_, err := s.testPartialRead(c, 3*b, buf, 0)
	c.Assert(err, IsNil)

	expected := make([]byte, b+b/2)
	fill(expected[:b], 1)
	fill(expected[b:], 2)

	byteEquals(c, expected, buf)
}

func (s *TestSuite) TestPartialFullRead(c *C) {
	// Sanity test that filling data works right
	buf := make([]byte, 2*b)
	_, err := s.testPartialRead(c, 2*b, buf, 0)
	c.Assert(err, IsNil)

	expected := make([]byte, 2*b)
	fill(expected[:b], 1)
	fill(expected[b:], 2)

	byteEquals(c, expected, buf)
}

func (s *TestSuite) TestPartialReadZeroEndOffset(c *C) {
	buf := make([]byte, b+b/2)
	_, err := s.testPartialRead(c, 2*b, buf, b/2)
	c.Assert(err, IsNil)

	expected := make([]byte, b+b/2)
	fill(expected[:b/2], 1)
	fill(expected[b/2:], 2)

	byteEquals(c, expected, buf)
}

// TestUpdateLUNMap tests can be used to verify if UpdateLUNMap properly compares
// the active LUNMap and preloaded lunMap and punch holes whenever necessary.
// It tests if the writes done after preload operation are accomodated in the
// lunmap and holes are punched for the same.
func (s *TestSuite) TestUpdateLUNMap(c *C) {
	dir, err := ioutil.TempDir("", "replica")
	c.Logf("Volume: %s", dir)
	c.Assert(err, IsNil)
	defer os.RemoveAll(dir)

	r, err := New(true, 100*b, b, dir, nil, "Backend")
	c.Assert(err, IsNil)
	defer r.Close()
	err = r.SetReplicaMode("WO")
	types.ShouldPunchHoles = true
	go CreateHoles()
	c.Assert(err, IsNil)
	server := &Server{
		r:   r,
		dir: dir,
	}
	// Fill data for S0
	lunMapS0 := []int{1, 3, 5, 6}
	fillMappedData(c, r, lunMapS0, 1)
	c.Assert(verifyLunMap(r, r.volume.files[1], lunMapS0), Equals, true)
	err = r.Snapshot("000", false, getNow())
	r.Close()
	r, err = New(false, 100*b, b, dir, nil, "Backend")
	server.r = r
	err = r.SetReplicaMode("WO")
	c.Assert(err, IsNil)
	c.Assert(verifyLunMap(r, r.volume.files[1], lunMapS0), Equals, true)
	// Fill data for head before preload is called
	lunMapHB := []int{4, 8, 10, 11}
	fillMappedData(c, r, lunMapHB, 2)
	c.Assert(verifyLunMap(r, r.volume.files[1], lunMapS0), Equals, true)
	c.Assert(verifyLunMap(r, r.volume.files[2], lunMapHB), Equals, true)

	err = r.SetRebuilding(true)
	c.Assert(err, IsNil)
	err = os.Setenv("UpdateLUNMap_TIMEOUT", "5")
	c.Assert(err, IsNil)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		server.UpdateLUNMap()
		wg.Done()
	}()
	for !inject.UpdateLUNMapTimeoutTriggered {
		time.Sleep(1 * time.Second)
	}
	// Fill data for head just after preload is called and before lunMaps are
	// compared
	lunMapHA := []int{1, 3, 5, 6, 8, 10, 11}
	fillMappedData(c, r, lunMapHA, 3)

	wg.Wait()
	expLunMapS0 := []int{}
	expLunMapH := []int{1, 3, 4, 5, 6, 8, 10, 11}
	// Waiting for holes to be punched
	time.Sleep(3 * time.Second)
	c.Assert(verifyLunMap(r, r.volume.files[1], expLunMapS0), Equals, true)
	c.Assert(verifyLunMap(r, r.volume.files[2], expLunMapH), Equals, true)
}

func verifyLunMap(r *Replica, f types.DiffDisk, expLunMap []int) bool {
	generator := newGenerator(&r.volume, f)
	outLunMap := []int{}
	for offset := range generator.Generate() {
		outLunMap = append(outLunMap, int(offset))
	}
	return reflect.DeepEqual(expLunMap, outLunMap)
}

func fillMappedData(c *C, r *Replica, LUNMap []int, val byte) {
	buf := make([]byte, b)
	fill(buf, val)
	for _, i := range LUNMap {
		count, err := r.WriteAt(buf, int64(i)*b)
		c.Assert(err, IsNil)
		c.Assert(count, Equals, b)
	}
}
