package handlers_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/alphagov/router/handlers"
	log "github.com/alphagov/router/logger"
)

var _ = Describe("Backend handler", func() {
	var (
		timeout = 1 * time.Second
		logger  log.Logger

		backend    *ghttp.Server
		backendURL *url.URL

		rw     *httptest.ResponseRecorder
		router http.Handler
	)

	BeforeEach(func() {
		var err error

		logger, err = log.New(GinkgoWriter)
		Expect(err).NotTo(HaveOccurred(), "Could not create logger")

		backend = ghttp.NewServer()

		backendURL, err = url.Parse(backend.URL())
		Expect(err).NotTo(HaveOccurred(), "Could not parse backend URL")

		rw = httptest.NewRecorder()
	})

	AfterEach(func() {
		backend.Close()
	})

	Context("when the backend times out", func() {
		BeforeEach(func() {
			router = handlers.NewBackendHandler(
				backendURL,
				timeout, timeout,
				logger,
			)

			backend.AppendHandlers(func(rw http.ResponseWriter, r *http.Request) {
				time.Sleep(timeout * 2)
				rw.WriteHeader(http.StatusOK)
			})

			router.ServeHTTP(
				rw,
				httptest.NewRequest("GET", backendURL.String(), nil),
			)
		})

		It("should return HTTP 504", func() {
			Expect(rw.Result().StatusCode).To(Equal(http.StatusGatewayTimeout))
		})

		It("should not populate the Via header", func() {
			Expect(rw.Result().Header.Get("Via")).To(Equal(""))
		})
	})

	Context("when the backend handles the connection", func() {
		BeforeEach(func() {
			router = handlers.NewBackendHandler(
				backendURL,
				timeout, timeout,
				logger,
			)
		})

		Context("when the proxied request returns 200", func() {
			BeforeEach(func() {
				backend.AppendHandlers(ghttp.RespondWith(200, "Hello World"))

				router.ServeHTTP(
					rw,
					httptest.NewRequest("GET", backendURL.String(), nil),
				)
			})

			It("should return 200", func() {
				Expect(rw.Result().StatusCode).To(Equal(http.StatusOK))
			})

			It("should return the body", func() {
				body, err := ioutil.ReadAll(rw.Result().Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal("Hello World"))
			})

			It("should populate the Via header", func() {
				Expect(rw.Result().Header.Get("Via")).To(Equal("1.1 router"))
			})
		})

		Context("when the proxied request returns 403", func() {
			BeforeEach(func() {
				backend.AppendHandlers(ghttp.RespondWith(403, "Forbidden"))

				router.ServeHTTP(
					rw,
					httptest.NewRequest("GET", backendURL.String(), nil),
				)
			})

			It("should return 403", func() {
				Expect(rw.Result().StatusCode).To(Equal(http.StatusForbidden))
			})

			It("should return the body", func() {
				body, err := ioutil.ReadAll(rw.Result().Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal("Forbidden"))
			})

			It("should populate the Via header", func() {
				Expect(rw.Result().Header.Get("Via")).To(Equal("1.1 router"))
			})
		})
	})
})