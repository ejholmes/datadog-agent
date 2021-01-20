package serverless

import (
	"fmt"
	"testing"
	"time"

	"github.com/DataDog/datadog-agent/pkg/serverless/flush"
	"github.com/stretchr/testify/assert"
)

func TestAutoSelectStrategy(t *testing.T) {
	assert := assert.New(t)
	d := Daemon{
		lastInvocations: make([]time.Time, 0),
		flushStrategy:   &flush.AtTheEnd{},
	}

	now := time.Now()

	// when not enough data, the flush at the end strategy should be selected
	// -----

	assert.Equal((&flush.AtTheEnd{}).String(), d.AutoSelectStrategy().String(), "not the good strategy has been selected") // default strategy

	assert.True(d.StoreInvocationTime(now.Add(-time.Second * 140)))
	assert.Equal((&flush.AtTheEnd{}).String(), d.AutoSelectStrategy().String(), "not the good strategy has been selected")
	assert.True(d.StoreInvocationTime(now.Add(-time.Second * 70)))
	assert.Equal((&flush.AtTheEnd{}).String(), d.AutoSelectStrategy().String(), "not the good strategy has been selected")

	// add a third invocation, after this, we have enough data to decide to switch
	// to the "flush at the start" strategy since the function is invoked more often
	// than 1 time a minute.
	// -----

	assert.True(d.StoreInvocationTime(now.Add(-time.Second * 1)))
	assert.Equal((&flush.AtTheStart{}).String(), d.AutoSelectStrategy().String(), "not the good strategy has been selected")

	// simulate a function invoked less than 1 time a minute
	// -----

	// reset the data
	d.lastInvocations = make([]time.Time, 0)
	assert.Equal((&flush.AtTheEnd{}).String(), d.AutoSelectStrategy().String(), "not the good strategy has been selected") // default strategy

	assert.True(d.StoreInvocationTime(now.Add(-time.Minute * 16)))
	assert.Equal((&flush.AtTheEnd{}).String(), d.AutoSelectStrategy().String(), "not the good strategy has been selected")
	assert.True(d.StoreInvocationTime(now.Add(-time.Minute * 10)))
	assert.Equal((&flush.AtTheEnd{}).String(), d.AutoSelectStrategy().String(), "not the good strategy has been selected")
	assert.True(d.StoreInvocationTime(now.Add(-time.Minute * 6)))
	// because of the frequency, we should kept the "flush at the end" strategy
	fmt.Println(d.InvocationFrequency())
	assert.Equal((&flush.AtTheEnd{}).String(), d.AutoSelectStrategy().String(), "not the good strategy has been selected")
}

func TestStoreInvocationTime(t *testing.T) {
	assert := assert.New(t)
	d := Daemon{
		lastInvocations: make([]time.Time, 0),
		flushStrategy:   &flush.AtTheEnd{},
	}

	now := time.Now()
	for i := 100; i > 0; i-- {
		d.StoreInvocationTime(now.Add(-time.Second * time.Duration(i)))
	}

	assert.True(len(d.lastInvocations) <= maxInvocationsStored, "the amount of stored invocations should be lower or equal to 50")
	// validate that the proper entries were removed
	assert.Equal(now.Add(-time.Second*50), d.lastInvocations[0])
	assert.Equal(now.Add(-time.Second*49), d.lastInvocations[1])
}

func TestInvocationFrequency(t *testing.T) {
	assert := assert.New(t)

	d := Daemon{
		lastInvocations: make([]time.Time, 0),
		flushStrategy:   &flush.AtTheEnd{},
	}

	// first scenario, validate that we're not computing the frequency if we only have 2 invocations done
	// -----

	for i := 0; i < 2; i++ {
		time.Sleep(100 * time.Millisecond)
		d.lastInvocations = append(d.lastInvocations, time.Now())
		assert.Equal(time.Duration(0), d.InvocationFrequency(), "we should not compute any frequency just yet since we don't have enough data")
	}
	time.Sleep(100 * time.Millisecond)
	d.lastInvocations = append(d.lastInvocations, time.Now())

	//	assert.Equal(d.InvocationFrequency(), time.Duration(0), "we should not compute any frequency just yet since we don't have enough data")
	assert.NotEqual(time.Duration(0), d.InvocationFrequency(), "we should compute some frequency now")

	// second scenario, validate the frequency that has been computed
	// -----

	// reset the data
	d.lastInvocations = make([]time.Time, 0)

	// function executed every second

	now := time.Now()
	for i := 100; i > 1; i-- {
		d.StoreInvocationTime(now.Add(-time.Second * time.Duration(i)))
	}

	// because we've added 50 execution, one every last 50 seconds, the frequency
	// computed between each function execution should be 1s
	assert.Equal(maxInvocationsStored, len(d.lastInvocations), fmt.Sprintf("the amount of invocations stored should be %d", maxInvocationsStored))
	assert.Equal(time.Second, d.InvocationFrequency(), "the compute frequency should be 1s")

	// function executed 100ms

	for i := 100; i > 1; i-- {
		d.StoreInvocationTime(now.Add(-time.Millisecond * 10 * time.Duration(i)))
	}

	// because we've added 50 execution, one every last 50 seconds, the frequency
	// computed between each function execution should be 1s
	assert.Equal(maxInvocationsStored, len(d.lastInvocations), fmt.Sprintf("the amount of invocations stored should be %d", maxInvocationsStored))
	assert.Equal(time.Millisecond*10, d.InvocationFrequency(), "the compute frequency should be 100ms")
}
