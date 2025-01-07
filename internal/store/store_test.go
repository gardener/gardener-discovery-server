// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/gardener-discovery-server/internal/store"
)

type data struct {
	bytes []byte
}

func copyData(d data) data {
	out := data{
		bytes: make([]byte, len(d.bytes)),
	}
	copy(out.bytes, d.bytes)
	return out
}

var _ = Describe("Store", func() {
	const (
		fooKey string = "foo"
	)
	var (
		s              *store.Store[data]
		d              data
		expectedData   data
		assertExpected = func(store *store.Store[data], key string, expected data) {
			retrieved, ok := store.Read(key)
			Expect(ok).To(BeTrue())
			Expect(retrieved).To(Equal(expected))
		}
		assertNotFound = func(store *store.Store[data], key string) {
			retrieved, ok := store.Read(key)
			Expect(ok).To(BeFalse())
			Expect(retrieved).To(Equal(data{}))
		}
	)
	BeforeEach(func() {
		s = store.MustNewStore(copyData)
		d = data{
			bytes: []byte("config"),
		}
		expectedData = data{
			bytes: []byte("config"),
		}
	})

	It("should be empty", func() {
		Expect(s.Len()).To(Equal(0))
	})

	It("should not find an entry", func() {
		s.Write(fooKey, d)

		Expect(s.Len()).To(Equal(1))
		assertNotFound(s, "bar")
	})

	It("should correctly set an entry", func() {
		s.Write(fooKey, d)

		Expect(s.Len()).To(Equal(1))
		assertExpected(s, fooKey, expectedData)
	})

	It("should correctly overwrite an entry", func() {
		s.Write(fooKey, d)

		Expect(s.Len()).To(Equal(1))
		assertExpected(s, fooKey, expectedData)

		newData := data{bytes: []byte("foo")}
		expectedNewData := data{bytes: []byte("foo")}
		s.Write(fooKey, newData)

		Expect(s.Len()).To(Equal(1))
		assertExpected(s, fooKey, expectedNewData)
	})

	It("should not be able to modify the unredlying entry", func() {
		s.Write(fooKey, d)

		Expect(s.Len()).To(Equal(1))
		retrieved, ok := s.Read(fooKey)

		Expect(ok).To(BeTrue())
		Expect(retrieved).To(Equal(expectedData))

		// modify a single byte
		retrieved.bytes[0]++

		assertExpected(s, fooKey, expectedData)
	})

	It("should correctly remove an entry", func() {
		s.Write(fooKey, d)

		assertExpected(s, fooKey, expectedData)

		s.Delete(fooKey)
		assertNotFound(s, fooKey)
		Expect(s.Len()).To(Equal(0))
	})

	It("should be able to use the store in parallel", func() {
		initialEntries := map[string]data{
			"0": {bytes: []byte("0")},
			"1": {bytes: []byte("1")},
			"2": {bytes: []byte("2")},
			"3": {bytes: []byte("3")},
			"4": {bytes: []byte("4")},
			"5": {bytes: []byte("5")},
		}

		var wg sync.WaitGroup
		wg.Add(len(initialEntries))
		for k, e := range initialEntries {
			go func() {
				s.Write(k, e)
				wg.Done()
			}()
		}
		wg.Wait()

		expectedEntries := map[string]data{
			"0": {bytes: []byte("0")},
			"1": {bytes: []byte("1")},
			"2": {bytes: []byte("2")},
			"3": {bytes: []byte("3")},
			"4": {bytes: []byte("4")},
			"5": {bytes: []byte("5")},
		}

		Expect(s.Len()).To(Equal(len(expectedEntries)))
		wg.Add(len(expectedEntries))
		for k, e := range expectedEntries {
			go func() {
				assertExpected(s, k, e)
				wg.Done()
			}()
		}
		wg.Wait()

		modifyEntries := map[string]data{
			"2": {bytes: []byte("22")},
			"3": {bytes: []byte("33")},
			"5": {bytes: []byte("55")},
		}

		wg.Add(len(modifyEntries))
		for k, e := range modifyEntries {
			go func() {
				s.Write(k, e)
				wg.Done()
			}()
		}
		wg.Wait()

		expectedEntries = map[string]data{
			"0": {bytes: []byte("0")},
			"1": {bytes: []byte("1")},
			"2": {bytes: []byte("22")},
			"3": {bytes: []byte("33")},
			"4": {bytes: []byte("4")},
			"5": {bytes: []byte("55")},
		}

		Expect(s.Len()).To(Equal(len(expectedEntries)))
		wg.Add(len(expectedEntries))
		for k, e := range expectedEntries {
			go func() {
				assertExpected(s, k, e)
				wg.Done()
			}()
		}
		wg.Wait()

		keysToDelete := []string{"0", "1", "5", "111"}
		wg.Add(len(keysToDelete))
		for _, k := range keysToDelete {
			go func() {
				s.Delete(k)
				wg.Done()
			}()
		}
		wg.Wait()

		expectedEntries = map[string]data{
			"2": {bytes: []byte("22")},
			"3": {bytes: []byte("33")},
			"4": {bytes: []byte("4")},
		}
		Expect(s.Len()).To(Equal(len(expectedEntries)))
		wg.Add(len(expectedEntries))
		for k, e := range expectedEntries {
			go func() {
				assertExpected(s, k, e)
				wg.Done()
			}()
		}
		wg.Wait()
	})
})
