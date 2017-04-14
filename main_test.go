package main_test

import (
	. "github.com/kurrels/dontfearthesweeper"

	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http/httptest"
)

var _ = Describe("Main", func() {
	Describe("CalculateNextCall", func() {
		It("should calculate the next date to send an alert", func() {
			location, err := time.LoadLocation("America/New_York")
			Expect(err).NotTo(HaveOccurred())

			oldNowFunc := Now
			Now = func() time.Time {
				return time.Date(2017, 4, 6, 0, 0, 0, 0, location)
			}
			defer func() {
				Now = oldNowFunc
			}()

			alert := Alert{
				Timezone:    "America/New_York",
				NthDay:      1,
				Weekday:     0,
				CountryCode: 1,
				PhoneNumber: 8054234224,
			}

			nextAlertTime, err := CalculateNextCall(alert)
			fmt.Println("time: ", time.Unix(nextAlertTime, 0).In(location))
			Expect(err).NotTo(HaveOccurred())
			Expect(nextAlertTime).To(Equal(int64(1494198000))) //2017-05-07 19:00:00 -0400 EDT

			// change the weekday to friday to make sure that the function determines that the next alert is this month
			alert.Weekday = 5

			nextAlertTime, err = CalculateNextCall(alert)
			fmt.Println("time: ", time.Unix(nextAlertTime, 0).In(location))
			Expect(err).NotTo(HaveOccurred())
			Expect(nextAlertTime).To(Equal(int64(1491606000))) // 2017-04-07 19:00:00 -0400 EDT
		})
	})

	Describe("application", func() {
		It("should alert people who's alert is past due", func() {
			// Create a new request.
			req := httptest.NewRequest("GET", "http://example.com/foo", nil)

			// Create a new ResponseRecorder which implements
			// the ResponseWriter interface.
			res := httptest.NewRecorder()

			// Execute the handler function directly.
			VerificationVerifyHandler(res, req)

			// Validate we received the expected response.
			got := res.Body.String()
			want := "Hello World!"
			if got != want {
				t.Log("Wanted:", want)
				t.Log("Got   :", got)
				t.Fatal("Mismatch")
			}
		})
	})
})
