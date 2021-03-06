package feedx_test

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/bsm/bfs"
	"github.com/bsm/feedx"
	"github.com/bsm/feedx/internal/testdata"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reader", func() {
	var subject *feedx.Reader
	var obj *bfs.Object
	var ctx = context.Background()

	BeforeEach(func() {
		obj = bfs.NewInMemObject("path/to/file.json")
		Expect(writeMulti(obj, 3)).To(Succeed())

		var err error
		subject, err = feedx.NewReader(ctx, obj, nil)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(subject.Close()).To(Succeed())
	})

	It("should read", func() {
		data, err := ioutil.ReadAll(subject)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(data)).To(BeNumerically("~", 110, 20))
		Expect(subject.NumRead()).To(Equal(0))
	})

	It("should decode", func() {
		var msgs []*testdata.MockMessage
		for {
			var msg testdata.MockMessage
			err := subject.Decode(&msg)
			if err == io.EOF {
				break
			}
			Expect(err).NotTo(HaveOccurred())
			msgs = append(msgs, &msg)
		}

		Expect(msgs).To(ConsistOf(seed(), seed(), seed()))
		Expect(subject.NumRead()).To(Equal(3))
	})
})
