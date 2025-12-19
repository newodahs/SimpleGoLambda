package credparser

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type CredentialInfo struct {
	User     string   `json:"username,omitempty" dynamodbav:"username,omitempty"`
	Domain   string   `json:"domain,omitempty" dynamodbav:"domainname,omitempty"`
	Email    string   `json:"email" dynamodbav:"email"`
	Password []string `json:"password,omitempty" dynamodbav:"password,omitempty"`
}

func (ci CredentialInfo) String() string {
	return fmt.Sprintf("%s:%s", ci.Email, ci.Password)
}

func (ci CredentialInfo) GetAttrDefs() []types.AttributeDefinition {
	return []types.AttributeDefinition{
		{
			AttributeName: aws.String("domainname"),
			AttributeType: types.ScalarAttributeTypeS,
		},
		{
			AttributeName: aws.String("username"),
			AttributeType: types.ScalarAttributeTypeS,
		},
	}
}

func (ci CredentialInfo) GetKeySchema() []types.KeySchemaElement {
	return []types.KeySchemaElement{
		{
			AttributeName: aws.String("domainname"),
			KeyType:       types.KeyTypeHash,
		},
		{
			AttributeName: aws.String("username"),
			KeyType:       types.KeyTypeRange,
		},
	}
}

func (ci CredentialInfo) GetKey() map[string]types.AttributeValue {
	ret, err := attributevalue.MarshalMap(ci)
	if err != nil {
		panic(err) //TODO: I don't like panicing in library functions
	}
	return ret
}

func GetCredentialInfoFile(filename string) (map[string]*CredentialInfo, error) {

	credFile, fErr := os.Open(filename)
	if fErr != nil {
		return nil, fmt.Errorf("failed to open credentials file: %w", fErr)
	}
	defer credFile.Close()

	return GetCredentialInfo(bufio.NewScanner(credFile))
}

// refactored to these regexes but not a lot of time to test them; cursory testing shows they do what I need though
var (
	passwordNormRegex     = regexp.MustCompile(`^([\w\d_\.\-\s]+@[a-zA-Z0-9_\.\-\s]+)(?:[:;,]{1}|~{3})(.+)$`)
	passwordBackwardRegex = regexp.MustCompile(`^(.+)(?:[:;,]{1}|~{3})([a-zA-Z0-9_\.\-\s]+@[\w\d_\.\-\s]+)$`)
)

var ErrBadParameter = errors.New("bad parameter passed")

// trying to keep this as simple as possible
func GetCredentialInfo(scanner *bufio.Scanner) (map[string]*CredentialInfo, error) {
	if scanner == nil {
		return nil, ErrBadParameter
	}

	//for ignoring duplicates..
	credList := map[string]*CredentialInfo{}
	dupChk := map[string]struct{}{}

	// var ret []*CredentialInfo
	var runningErr error
	for lineCnt := 1; scanner.Scan(); lineCnt++ {
		cur := scanner.Text()

		if _, exists := dupChk[cur]; exists {
			runningErr = errors.Join(runningErr, fmt.Errorf("duplicate found; already processed [%s] (duplicate @ line [%d])", cur, lineCnt))
			continue
		}
		dupChk[cur] = struct{}{}

		var email, passwd string
		// split the line apart
		if splitLine := passwordNormRegex.FindAllStringSubmatch(cur, -1); splitLine == nil {
			// try the backward regex as this may be out of expected order
			if splitLine = passwordBackwardRegex.FindAllStringSubmatch(cur, -1); splitLine == nil {
				runningErr = errors.Join(runningErr, fmt.Errorf("failed to parse credentials [%s] at line [%d]", cur, lineCnt))
				continue
			}
			email = splitLine[0][2]
			passwd = splitLine[0][1]
		} else {
			email = splitLine[0][1]
			passwd = splitLine[0][2]
		}

		// TODO: Remove; old logic pre-regex change
		//some of this data isn't well formed (mostly it appears colon separated, however, sometimes it's semi-colon, comma, or even in one case, three tildes)
		//in the future we could refactor to use a regex for an email to help improve consistency
		// idx, offset := findPasswordIdx(cur)
		// if idx == 0 && offset == 0 {
		// 	runningErr = errors.Join(runningErr, fmt.Errorf("failed to parse credentials [%s] at line [%d]", cur, lineCnt))
		// 	continue
		// }
		// we assume the email is in position 1, if it's not we need to swap these
		// email := cur[:idx]
		// passwd := cur[idx+offset:]

		// with the change to regex, we don't need to have complicated logic here, we can just call split once...
		username, domain := splitEmailUserDomain(email) // try the split, swap if not successful
		if username == "" || domain == "" {
			runningErr = errors.Join(runningErr, fmt.Errorf("failed to find username and password from credential [%s] at line [%d]; could not determine email field", cur, lineCnt))
			continue

			// TODO: Remove; old logic pre-regex change
			// 	//swap the email and password, they may be in reverse order
			// 	tmp := email
			// 	email = passwd
			// 	passwd = tmp
			// 	if username, domain = splitEmailUserDomain(email); username == "" || domain == "" {
			// 		runningErr = errors.Join(runningErr, fmt.Errorf("failed to find username and password from credential [%s] at line [%d]; could not determine email field", cur, lineCnt))
			// 		continue
			// 	}
		}

		//see if we already processed this entry...
		if _, exists := credList[email]; exists { // TODO: filter out duplicate passwords?
			credList[email].Password = append(credList[email].Password, passwd)
			continue
		}

		credList[email] = &CredentialInfo{User: username, Domain: domain, Email: email, Password: []string{passwd}}
	}

	return credList, runningErr
}

func splitEmailUserDomain(email string) (user, domain string) {
	if email == "" { // quick out; nothing to do...
		return "", ""
	}

	idx := strings.Index(email, `@`)
	if idx < 0 {
		return "", ""
	}

	return email[:idx], email[idx+1:]
}

// TODO: Remove; old logic pre-regex change
// func findPasswordIdx(cred string) (index int, offset int) {
// 	idx := strings.Index(cred, `:`)
// 	if idx > 0 {
// 		return idx, 1
// 	}

// 	idx = strings.Index(cred, `;`)
// 	if idx > 0 {
// 		return idx, 1
// 	}

// 	idx = strings.Index(cred, `,`)
// 	if idx > 0 {
// 		return idx, 1
// 	}

// 	idx = strings.Index(cred, `~~~`)
// 	if idx > 0 {
// 		return idx, 3
// 	}

// 	return 0, 0
// }
