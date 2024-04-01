// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openidmeta_test

import (
	"sync"

	"github.com/gardener/gardener-discovery-server/internal/store/openidmeta"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	const (
		fooKey string = "foo"
	)
	var (
		store          *openidmeta.Store
		data           openidmeta.Data
		expectedData   openidmeta.Data
		assertExpected = func(store *openidmeta.Store, key string, expected openidmeta.Data) {
			retrieved, ok := store.Read(key)
			Expect(ok).To(BeTrue())
			Expect(retrieved).To(Equal(expected))
		}
		assertNotFound = func(store *openidmeta.Store, key string) {
			retrieved, ok := store.Read(key)
			Expect(ok).To(BeFalse())
			Expect(retrieved).To(Equal(openidmeta.Data{}))
		}
	)
	BeforeEach(func() {
		store = openidmeta.NewStore()
		data = openidmeta.Data{
			Config: []byte("config"),
			JWKS:   []byte("jwks"),
		}
		expectedData = openidmeta.Data{
			Config: []byte("config"),
			JWKS:   []byte("jwks"),
		}
	})

	It("should be empty", func() {
		Expect(store.Len()).To(Equal(0))
	})

	It("should not find an entry", func() {
		store.Write(fooKey, data)
		Expect(store.Len()).To(Equal(1))

		assertNotFound(store, "bar")
	})

	It("should correctly set an entry", func() {
		store.Write(fooKey, data)

		Expect(store.Len()).To(Equal(1))
		assertExpected(store, fooKey, expectedData)
	})

	It("should correctly overwrite an entry", func() {
		store.Write(fooKey, data)

		Expect(store.Len()).To(Equal(1))
		assertExpected(store, fooKey, expectedData)

		newData := openidmeta.Data{Config: []byte("foo"), JWKS: []byte("bar")}
		expectedNewData := openidmeta.Data{Config: []byte("foo"), JWKS: []byte("bar")}
		store.Write(fooKey, newData)

		Expect(store.Len()).To(Equal(1))
		assertExpected(store, fooKey, expectedNewData)
	})

	It("should not be able to modify the unredlying entry", func() {
		store.Write(fooKey, data)

		Expect(store.Len()).To(Equal(1))
		retrieved, ok := store.Read(fooKey)

		Expect(ok).To(BeTrue())
		Expect(retrieved).To(Equal(expectedData))

		// modify a single byte
		retrieved.Config[0] = retrieved.Config[0] + 1
		retrieved.JWKS[0] = retrieved.JWKS[0] + 1

		assertExpected(store, fooKey, expectedData)
	})

	It("should correctly remove an entry", func() {
		store.Write(fooKey, data)

		assertExpected(store, fooKey, expectedData)

		store.Delete(fooKey)
		assertNotFound(store, fooKey)
		Expect(store.Len()).To(Equal(0))
	})

	It("should be able to use the store in parallel", func() {
		initialEntries := map[string]openidmeta.Data{
			"0": {Config: []byte("0"), JWKS: []byte("0")},
			"1": {Config: []byte("1"), JWKS: []byte("1")},
			"2": {Config: []byte("2"), JWKS: []byte("2")},
			"3": {Config: []byte("3"), JWKS: []byte("3")},
			"4": {Config: []byte("4"), JWKS: []byte("4")},
			"5": {Config: []byte("5"), JWKS: []byte("5")},
		}

		var wg sync.WaitGroup
		wg.Add(len(initialEntries))
		for k, e := range initialEntries {
			go func() {
				store.Write(k, e)
				wg.Done()
			}()
		}
		wg.Wait()

		expectedEntries := map[string]openidmeta.Data{
			"0": {Config: []byte("0"), JWKS: []byte("0")},
			"1": {Config: []byte("1"), JWKS: []byte("1")},
			"2": {Config: []byte("2"), JWKS: []byte("2")},
			"3": {Config: []byte("3"), JWKS: []byte("3")},
			"4": {Config: []byte("4"), JWKS: []byte("4")},
			"5": {Config: []byte("5"), JWKS: []byte("5")},
		}

		Expect(store.Len()).To(Equal(len(expectedEntries)))
		wg.Add(len(expectedEntries))
		for k, e := range expectedEntries {
			go func() {
				assertExpected(store, k, e)
				wg.Done()
			}()
		}
		wg.Wait()

		modifyEntries := map[string]openidmeta.Data{
			"2": {Config: []byte("22"), JWKS: []byte("22")},
			"3": {Config: []byte("33"), JWKS: []byte("33")},
			"5": {Config: []byte("55"), JWKS: []byte("55")},
		}

		wg.Add(len(modifyEntries))
		for k, e := range modifyEntries {
			go func() {
				store.Write(k, e)
				wg.Done()
			}()
		}
		wg.Wait()

		expectedEntries = map[string]openidmeta.Data{
			"0": {Config: []byte("0"), JWKS: []byte("0")},
			"1": {Config: []byte("1"), JWKS: []byte("1")},
			"2": {Config: []byte("22"), JWKS: []byte("22")},
			"3": {Config: []byte("33"), JWKS: []byte("33")},
			"4": {Config: []byte("4"), JWKS: []byte("4")},
			"5": {Config: []byte("55"), JWKS: []byte("55")},
		}

		Expect(store.Len()).To(Equal(len(expectedEntries)))
		wg.Add(len(expectedEntries))
		for k, e := range expectedEntries {
			go func() {
				assertExpected(store, k, e)
				wg.Done()
			}()
		}
		wg.Wait()

		keysToDelete := []string{"0", "1", "5", "111"}
		wg.Add(len(keysToDelete))
		for _, k := range keysToDelete {
			go func() {
				store.Delete(k)
				wg.Done()
			}()
		}
		wg.Wait()

		expectedEntries = map[string]openidmeta.Data{
			"2": {Config: []byte("22"), JWKS: []byte("22")},
			"3": {Config: []byte("33"), JWKS: []byte("33")},
			"4": {Config: []byte("4"), JWKS: []byte("4")},
		}
		Expect(store.Len()).To(Equal(len(expectedEntries)))
		wg.Add(len(expectedEntries))
		for k, e := range expectedEntries {
			go func() {
				assertExpected(store, k, e)
				wg.Done()
			}()
		}
		wg.Wait()
	})
})
