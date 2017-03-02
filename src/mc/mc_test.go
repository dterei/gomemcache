package mc

import (
	"github.com/bmizerany/assert"
	"testing"
)

const (
	mcAddr    = "localhost:11211"
	badAddr   = "127.0.0.2:23111"
	doAuth    = false
	authOnMac = true
	user      = "user-1"
	pass      = "pass"
)

var mcNil error

// shared connection
var cn *Conn

// Some basic tests that functions work
func TestMCSimple(t *testing.T) {
	testInit(t)

	const (
		Key1 = "foo"
		Val1 = "bar"
		Val2 = "bar-bad"
		Val3 = "bar-good"
	)

	_, _, _, err := cn.Get(Key1)
	assert.Equalf(t, ErrNotFound, err, "expected missing key: %v", err)

	// unconditional SET
	_, err = cn.Set(Key1, Val1, 0, 0, 0)
	assert.Equalf(t, mcNil, err, "unexpected error: %v", err)
	cas, err := cn.Set(Key1, Val1, 0, 0, 0)
	assert.Equalf(t, mcNil, err, "unexpected error: %v", err)

	// make sure CAS works
	_, err = cn.Set(Key1, Val2, 0, 0, cas+1)
	assert.Equalf(t, ErrKeyExists, err, "expected CAS mismatch: %v", err)

	// check SET actually set the correct value...
	v, _, cas2, err := cn.Get(Key1)
	assert.Equalf(t, mcNil, err, "unexpected error: %v", err)
	assert.Equalf(t, Val1, v, "wrong value: %s", v)
	assert.Equalf(t, cas, cas2, "CAS shouldn't have changed: %d, %d", cas, cas2)

	// use correct CAS...
	cas2, err = cn.Set(Key1, Val3, 0, 0, cas)
	assert.Equalf(t, mcNil, err, "unexpected error: %v", err)
	assert.NotEqual(t, cas, cas2)
}

// Test GET, does it care about CAS?
// NOTE: No it shouldn't, memcached mainline doesn't...
func TestGet(t *testing.T) {
	testInit(t)

	const (
		Key1 = "fab"
		Val1 = "faz"
	)

	_, err := cn.Set(Key1, Val1, 0, 0, 0)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)

	// retrieve value with 0 CAS...
	v1, _, cas1, err := cn.getCAS(Key1, 0)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)
	assert.Equalf(t, Val1, v1, "wrong value: %s", v1)

	// retrieve value with good CAS...
	v2, _, cas2, err := cn.getCAS(Key1, cas1)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)
	assert.Equalf(t, v1, v2, "value changed when it shouldn't: %s, %s", v1, v2)
	assert.Equalf(t, cas1, cas2, "CAS changed when it shouldn't: %d, %d", cas1, cas2)

	// retrieve value with bad CAS...
	v3, _, cas1, err := cn.getCAS(Key1, cas1+1)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)
	assert.Equalf(t, v3, v2, "value changed when it shouldn't: %s, %s", v3, v2)
	assert.Equalf(t, cas1, cas2, "CAS changed when it shouldn't: %d, %d", cas1, cas2)

	// really make sure CAS is bad (above could be an off by one bug...)
	v4, _, cas1, err := cn.getCAS(Key1, cas1+992313128)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)
	assert.Equalf(t, v4, v2, "value changed when it shouldn't: %s, %s", v4, v2)
	assert.Equalf(t, cas1, cas2, "CAS changed when it shouldn't: %d, %d", cas1, cas2)
}

// Test some edge cases of memcached. This was originally done to better
// understand the protocol but servers as a good test for the client and
// server...

// Test SET behaviour with CAS...
func TestSet(t *testing.T) {
	testInit(t)

	const (
		Key1 = "foo"
		Key2 = "goo"
		Val1 = "bar"
		Val2 = "zar"
	)

	cas1, err := cn.Set(Key1, Val1, 0, 0, 0)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)
	v, _, cas2, err := cn.Get(Key1)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)
	assert.Equalf(t, Val1, v, "wrong value: %v", v)
	assert.Equal(t, cas1, cas2, "CAS don't match: %d != %d", cas1, cas2)

	// do two sets of same key, make sure CAS changes...
	cas1, err = cn.Set(Key2, Val1, 0, 0, 0)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)
	cas2, err = cn.Set(Key2, Val1, 0, 0, 0)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)
	assert.NotEqual(t, cas1, cas2, "CAS don't match: %d == %d", cas1, cas2)

	// get back the val from Key2...
	v, _, cas2, err = cn.Get(Key2)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)
	assert.Equalf(t, Val1, v, "wrong value: %v", v)

	// make sure changing value works...
	_, err = cn.Set(Key1, Val2, 0, 0, 0)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)
	v, _, cas1, err = cn.Get(Key1)
	assert.Equalf(t, Val2, v, "wrong value: %s", v)

	// Delete Key1 and check it worked, needed for next test...
	err = cn.Del(Key1)
	assert.Equalf(t, mcNil, err, "shouldn't be an error: %v", err)
	_, _, _, err = cn.Get(Key1)
	assert.Equalf(t, ErrNotFound, err, "wrong error: %v", err)

	// What happens when I set a new key and specify a CAS?
	// (should fail, bad CAS, can't specify a CAS for a non-existent key, it fails,
	// doesn't just ignore the CAS...)
	cas, err := cn.Set(Key1, Val1, 0, 0, 1)
	assert.Equalf(t, ErrNotFound, err, "wrong error: %v", err)
	assert.Equalf(t, uint64(0), cas, "CAS should be nil: %d", cas)

	// make sure it really didn't set it...
	v, _, _, err = cn.Get(Key1)
	assert.Equalf(t, ErrNotFound, err, "wrong error: %v", err)
	// TODO: On errors a human readable error description should be returned. So
	// could test that.

	// Setting an existing value with bad CAS... should fail
	_, err = cn.Set(Key2, Val2, 0, 0, cas2+1)
	assert.Equalf(t, ErrKeyExists, err, "wrong error: %v", err)
	v, _, cas1, _ = cn.Get(Key2)
	assert.Equalf(t, Val1, v, "value shouldn't have changed: %s", v)
	assert.Equalf(t, cas1, cas2, "CAS shouldn't have changed: %d, %d", cas1, cas2)
}

// Test Delete.
func TestDelete(t *testing.T) {
	testInit(t)

	const (
		Key1 = "foo"
		Val1 = "bar"
	)

	// delete existing key...
	_, err := cn.Set(Key1, Val1, 0, 0, 0)
	assert.Equalf(t, mcNil, err, "unexpected error: %v", err)
	err = cn.Del(Key1)
	assert.Equalf(t, mcNil, err, "error deleting key: %v", err)

	// delete non-existent key...
	err = cn.Del(Key1)
	assert.Equalf(t, ErrNotFound, err,
		"no error deleting non-existent key: %v", err)

	// delete existing key with 0 CAS...
	cas1, err := cn.Set(Key1, Val1, 0, 0, 0)
	assert.Equalf(t, mcNil, err, "unexpected error: %v", err)
	err = cn.DelCAS(Key1, cas1+1)
	assert.Equalf(t, ErrKeyExists, err,
		"expected an error for deleting key with wrong CAS: %v", err)

	// confirm it isn't gone...
	v, _, cas1, err := cn.Get(Key1)
	assert.Equalf(t, mcNil, err,
		"delete with wrong CAS seems to have succeeded: %v", err)
	assert.Equalf(t, v, Val1, "corrupted value in cache: %v", v)

	// now delete with good CAS...
	err = cn.DelCAS(Key1, cas1)
	assert.Equalf(t, mcNil, err,
		"unexpected error for deleting key with correct CAS: %v", err)

	// delete existing key with good CAS...
	cas1, err = cn.Set(Key1, Val1, 0, 0, 0)
	assert.Equalf(t, mcNil, err, "unexpected error: %v", err)
	err = cn.DelCAS(Key1, cas1)
	assert.Equalf(t, mcNil, err,
		"unexpected error for deleting key with correct CAS: %v", err)
	_, _, _, err = cn.Get(Key1)
	assert.Equalf(t, ErrNotFound, err,
		"delete with wrong CAS seems to have succeeded: %v", err)

	// delete existing key with 0 CAS...
	_, err = cn.Set(Key1, Val1, 0, 0, 0)
	assert.Equalf(t, mcNil, err, "unexpected error: %v", err)
	err = cn.DelCAS(Key1, 0)
	assert.Equalf(t, mcNil, err,
		"unexpected error for deleting key with 0 CAS: %v", err)
	_, _, _, err = cn.Get(Key1)
	assert.Equalf(t, ErrNotFound, err,
		"delete with wrong CAS seems to have succeeded: %v", err)
}
