package credparser

import (
	"bufio"
	"strings"
	"testing"
)

// I normally write a lot more unit tests than this, but as this is a throw-away challenge
// I'm only going to put together the base-set of items I watch to look for in my parser to
// ensure some level of sanity
func Test_CredentialParser_DirectString(t *testing.T) {
	// grouping test cases together is useful as the project builds out, in this case I have three separate cases
	// under testing the 'Direct String' parsing mode; I could also write a test for parsing from file, etc...
	// but this test is just focused on the parser itself
	testSet := []struct {
		Name           string
		Creds          []string
		ExpectedOutput map[string]*CredentialInfo
		SkipTest       bool
	}{
		{
			Name:  "Simple Case-Three Entries",
			Creds: []string{"testName@blah.com:somePassword", "second+another@another.com:135324", "name Withspace@test-domain.com:33  @@31!"},
			ExpectedOutput: map[string]*CredentialInfo{
				"testName@blah.com": {
					User:     "testName",
					Domain:   "blah.com",
					Email:    "testName@blah.com",
					Password: []string{"somePassword"},
				},
				"second+another@another.com": {
					User:     "second+another",
					Domain:   "another.com",
					Email:    "second+another@another.com",
					Password: []string{"135324"},
				},
				"name Withspace@test-domain.com": {
					User:     "name Withspace",
					Domain:   "test-domain.com",
					Email:    "name Withspace@test-domain.com",
					Password: []string{"33  @@31!"},
				},
			},
		},
		{
			Name:  "Non-Standard Delimiters",
			Creds: []string{"testName@blah.com;somePassword", "second+another@another.com,135324", "name Withspace@test-domain.com~~~33  @@31!"},
			ExpectedOutput: map[string]*CredentialInfo{
				"testName@blah.com": {
					User:     "testName",
					Domain:   "blah.com",
					Email:    "testName@blah.com",
					Password: []string{"somePassword"},
				},
				"second+another@another.com": {
					User:     "second+another",
					Domain:   "another.com",
					Email:    "second+another@another.com",
					Password: []string{"135324"},
				},
				"name Withspace@test-domain.com": {
					User:     "name Withspace",
					Domain:   "test-domain.com",
					Email:    "name Withspace@test-domain.com",
					Password: []string{"33  @@31!"},
				},
			},
		},
		{
			Name:  "Whitespace In Passwords",
			Creds: []string{"testName@blah.com;somePassword   ", "second+another@another.com.com,   135324", "name Withspace@test-domain.com~~~33  @@31!"},
			ExpectedOutput: map[string]*CredentialInfo{
				"testName@blah.com": {
					User:     "testName",
					Domain:   "blah.com",
					Email:    "testName@blah.com",
					Password: []string{"somePassword   "},
				},
				"second+another@another.com": {
					User:     "second+another",
					Domain:   "another.com",
					Email:    "second+another@another.com",
					Password: []string{"   135324"},
				},
				"name Withspace@test-domain.com": {
					User:     "name Withspace",
					Domain:   "test-domain.com",
					Email:    "name Withspace@test-domain.com",
					Password: []string{"33  @@31!"},
				},
			},
		},
	}

	for _, curTest := range testSet {
		t.Run(curTest.Name, func(t *testing.T) {
			if curTest.SkipTest {
				t.Skipf("skipped '%s' due to SkipTest being set", curTest.Name)
			}
		})

		credList, err := GetCredentialInfo(bufio.NewScanner(strings.NewReader(strings.Join(curTest.Creds, "\n"))))
		if err != nil {
			t.Fatalf("failed while testing GetCredentialInfo: %s", err)
		}

		if len(credList) != len(curTest.Creds) {
			t.Fatalf("returned list of credentials length (%d) does not match expected length (%d)", len(credList), len(curTest.Creds))
		}

		for _, expectCred := range curTest.ExpectedOutput {
			cred, found := credList[expectCred.Email]
			if !found {
				t.Fatalf("missing expected credential for [%s]", expectCred.Email)
			}

			// ensure we got our fields right...
			if cred.User != expectCred.User {
				t.Errorf("parsed username [%s] does not match expected username [%s]", cred.User, expectCred.User)
			}
			if cred.Domain != expectCred.Domain {
				t.Errorf("parsed domain [%s] does not match expected domain [%s]", cred.Domain, expectCred.Domain)
			}
			if cred.Email != expectCred.Email {
				t.Errorf("parsed email [%s] does not match expected domain [%s]", cred.Email, expectCred.Email)
			}
			if len(cred.Password) != len(expectCred.Password) {
				t.Errorf("password count mismatch: (got) %d vs %d (expected)", len(cred.Password), len(expectCred.Password))
			} else {
				for _, expectPwd := range expectCred.Password {
					found := false
					for _, pwd := range cred.Password {
						if pwd == expectPwd {
							found = true
							break
						}
						if !found {
							t.Errorf("failed to find expected password [%s] in parsed credentials", expectPwd)
						}
					}
				}
			}
		}
	}
}
