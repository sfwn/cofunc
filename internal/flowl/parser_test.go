package flowl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func loadTestingdata(data string) ([]*Block, error) {
	rd := strings.NewReader(data)
	bs, err := ParseBlocks(rd)
	if err != nil {
		return nil, err
	}
	var blocks []*Block
	bs.Foreach(func(b *Block) error {
		blocks = append(blocks, b)
		return nil
	})
	return blocks, nil
}

func TestParseBlocksFull(t *testing.T) {
	const testingdata string = `
	load cmd:root/function1
	load cmd:url/function2
	load cmd:path/function3
	load go:function4
	 
	run f1
	run	f2
	run	function3
	run function4

	fn f1 = function1 {
		args = {
			k1: v1
		}
	}
	
	fn f2=function2 {
	}
	`
	blocks, err := loadTestingdata(testingdata)
	if err != nil {
		assert.FailNow(t, err.Error())
	}
	_ = blocks
}

// Only load part
func TestParseBlocksOnlyLoad(t *testing.T) {
	const testingdata string = `
load cmd:function1
  load 			 go:function2

load cmd:function3

	load 	go:function4
	`
	blocks, err := loadTestingdata(testingdata)
	if err != nil {
		assert.FailNow(t, err.Error())
	}
	check := func(b *Block, path string) {
		assert.Equal(t, "load", b.Kind.Value)
		assert.Equal(t, path, b.Object.Value)
	}
	check(blocks[0], "cmd:function1")
	check(blocks[1], "go:function2")
	check(blocks[2], "cmd:function3")
	check(blocks[3], "go:function4")
}

func TestParseBlocksOnlyfn(t *testing.T) {
	const testingdata string = `
	fn f1 = function1 {
		args = {
			k1:v1
			k3:v3
		}
	}

fn f2=function2{ 
}
	`
	blocks, err := loadTestingdata(testingdata)
	if err != nil {
		assert.FailNow(t, err.Error())
	}
	_ = blocks
}

func TestParseBlocksSetWithError(t *testing.T) {
	// testingdata is an error data
	const testingdata1 string = `
fn f1= function1 {
	args = {
		k: v
	}


fn f2= function2 {
}
	`
	_, err := loadTestingdata(testingdata1)
	assert.Error(t, err)

	const testingdata2 string = `
	fn f1 = function1 {
		args = {
			k1:v1
			k2: v2
			k3:v3
		}
	}
	}
	`
	_, err = loadTestingdata(testingdata2)
	assert.Error(t, err)

}

func TestParseBlocksOnlyRun(t *testing.T) {
	const testingdata string = `
	run function1
	run 	function2{
		k1:v1
		k2:v2
	}

run function3 {
	k : {(1+2+3)}

	multi1: ***hello1
	hello2
	***

	multi2: *** 
	hello1
	hello2
	***

	multi3:*** 
	hello1
	hello2***
}

	`
	blocks, err := loadTestingdata(testingdata)
	if err != nil {
		assert.FailNow(t, err.Error())
	}
	check := func(b *Block, obj string) {
		assert.Nil(t, b.Child)
		assert.Nil(t, b.Parent)
		assert.Equal(t, LevelParent, b.Level)
		assert.Equal(t, "run", b.Kind.Value)
		assert.Equal(t, obj, b.Object.Value)

		if obj == "function2" {
			kvs := b.BlockBody.(*FlMap).ToMap()
			assert.Len(t, kvs, 2)
		}
		if obj == "function3" {
			kvs := b.BlockBody.(*FlMap).ToMap()
			assert.Len(t, kvs, 4)
			assert.Equal(t, "{(1+2+3)}", kvs["k"])
			assert.Equal(t, "hello1\nhello2\n", kvs["multi1"])
			assert.Equal(t, "\nhello1\nhello2\n", kvs["multi2"])
			assert.Equal(t, "\nhello1\nhello2", kvs["multi3"])
		}
	}
	check(blocks[0], "function1")
	check(blocks[1], "function2")
	check(blocks[2], "function3")
}

// Parallel run testing
func TestParseBlocksOnlyRun2(t *testing.T) {
	{
		const testingdata string = `
run    {

	function1
	function2

	function3

}
	`
		blocks, err := loadTestingdata(testingdata)
		if err != nil {
			assert.FailNow(t, err.Error())
		}
		assert.Len(t, blocks, 1)
		b := blocks[0]
		assert.Equal(t, "run", b.Kind.Value)
		assert.True(t, b.Receiver.IsEmpty())
		assert.True(t, b.Symbol.IsEmpty())
		assert.True(t, b.Object.IsEmpty())

		slice := b.BlockBody.(*FlList).ToSlice()
		assert.Len(t, slice, 3)
		e1, e2, e3 := slice[0], slice[1], slice[2]
		assert.Equal(t, "function1", e1)
		assert.Equal(t, "function2", e2)
		assert.Equal(t, "function3", e3)
	}

	{
		const testingdata string = `
		run{
	function1
	function2

	function3

}
	`
		blocks, err := loadTestingdata(testingdata)
		if err != nil {
			assert.FailNow(t, err.Error())
		}
		assert.Len(t, blocks, 1)
		b := blocks[0]
		assert.Equal(t, "run", b.Kind.Value)
		assert.True(t, b.Receiver.IsEmpty())
		assert.True(t, b.Symbol.IsEmpty())
		assert.True(t, b.Object.IsEmpty())

		slice := b.BlockBody.(*FlList).ToSlice()
		assert.Len(t, slice, 3)
		e1, e2, e3 := slice[0], slice[1], slice[2]
		assert.Equal(t, "function1", e1)
		assert.Equal(t, "function2", e2)
		assert.Equal(t, "function3", e3)
	}
}

func TestParseBlocksOnlyRun2WithError(t *testing.T) {
	{
		const testingdata string = `
run {
	function1

	load xxxx
	function2

	function3
}
	`
		_, err := loadTestingdata(testingdata)
		assert.Error(t, err)
	}

	{
		const testingdata string = `
run {
	function1
	run function2

	function3
}
	`
		_, err := loadTestingdata(testingdata)
		assert.Error(t, err)
	}

	{
		const testingdata string = `
run {
	function1
	input k v

	function3
}
	`
		_, err := loadTestingdata(testingdata)
		assert.Error(t, err)
	}

	{
		const testingdata string = `
run xyz {
	function1

	function3
}
	`
		_, err := loadTestingdata(testingdata)
		assert.Error(t, err)
	}

	{
		const testingdata string = `
run 3 {
	function1
	function3
}
	`
		_, err := loadTestingdata(testingdata)
		assert.Error(t, err)
	}

}

//
// Test Run queue and blockstore
func loadTestingdata2(data string) ([]*Block, *BlockStore, *RunQueue, error) {
	rd := strings.NewReader(data)
	rq, bs, err := Parse(rd)
	if err != nil {
		return nil, nil, nil, err
	}
	var blocks []*Block
	bs.Foreach(func(b *Block) error {
		blocks = append(blocks, b)
		return nil
	})
	return blocks, bs, rq, nil
}

/*
func TestParseFull(t *testing.T) {
	const testingdata string = `
	load go:function1
	load go:function2
	load cmd:/tmp/function3
	load cmd:/tmp/function4
	load cmd:/tmp/function5

	fn f1 = function1 {
	}

	run f1
	run	function2
	run	function3
	run {
		function4
		function5
	}
	`

	blocks, bs, rq, err := loadTestingdata2(testingdata)
	assert.NoError(t, err)
	assert.NotNil(t, blocks)
	assert.NotNil(t, bs)
	assert.NotNil(t, rq)

	assert.Len(t, rq.FNodes, 5)

	assert.Equal(t, "function1", rq.FNodes["function1"].Name)
	assert.Equal(t, "function3", rq.FNodes["function3"].Name)

	assert.Len(t, rq.FNodes["function1"].Args(), 2)
	assert.Equal(t, 4, rq.Queue.Len())
}

*/
