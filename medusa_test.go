package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_checkStatusCode(t *testing.T) {
	status := checkStatusCode([]string{"200", "301"}, "200")
	assert.True(t, status)

	status = checkStatusCode([]string{"200", "301"}, "404")
	assert.False(t, status)
}

func Test_generateURL(t *testing.T) {
	url := generateURL("rizasabuncu.com")
	assert.Equal(t, "http://rizasabuncu.com/", url)

	url = generateURL("rizasabuncu.com/admin")
	assert.Equal(t, "http://rizasabuncu.com/admin/", url)

	url = generateURL("https://rizasabuncu.com/")
	assert.Equal(t, "https://rizasabuncu.com/", url)

	url = generateURL("https://rizasabuncu.com/admin")
	assert.Equal(t, "https://rizasabuncu.com/admin/", url)

	url = generateURL("https://rizasabuncu.com/admin/dashboard")
	assert.Equal(t, "https://rizasabuncu.com/admin/dashboard/", url)
}

func Test_parseStatusCode(t *testing.T) {
	statusCodes := parseStatusCode("200,301")
	assert.Equal(t, []string{"200", "301"}, statusCodes)
}
