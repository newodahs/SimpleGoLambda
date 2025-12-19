# To build the lambda (deploying to an arm64 lambda):

This readerlambda takes a exploited credentials file, as see in `./test/challenge_creds.txt`, from S3 and does the following:
 1. parses all valid entries
   a. if an entry is not valid, it is logged out and omitted from the database
 2. pushes all validly parsed entries to a dynamodb table called `exploitedCredentials`
   a. if this table does not exist, it is created
 3. deletes the processed S3 object that triggered the process

NOTE: I actively filter out duplicates by email:password, however, if you have email1:password1 and email1:password2 in the file, I will record both passwords under that single email (we won't lose any).

NOTE2: invalid emails (missing user or domain) are considered invalid and thrown out today (though logged); I toyed with the idea of creating a 'catch all' domain and user category for them, however, I'm not sure there is a lot of value in that...

Build the lambda:

```
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap ./cmd/lambda/main.go
```
```
zip credReader.zip bootstrap
```

You'll need to have a lambda execution role for this next step (below, I'm using the name credReaderLambda) with a reasonable policy for dynamodb, cloudwatch, and s3.  Here is mine:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "ReadWriteTable",
            "Effect": "Allow",
            "Action": [
                "dynamodb:BatchGetItem",
                "dynamodb:GetItem",
                "dynamodb:Query",
                "dynamodb:Scan",
                "dynamodb:BatchWriteItem",
                "dynamodb:PutItem",
                "dynamodb:UpdateItem",
                "dynamodb:DescribeTable"
            ],
            "Resource": "arn:aws:dynamodb:us-east-2:111122223333:table/*"
        },
        {
            "Sid": "WriteLogStreamsAndGroups",
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": "arn:aws:logs:us-east-2:111122223333:log-group:/aws/lambda/readerLambda:*"
        },
        {
            "Sid": "CreateLogGroup",
            "Effect": "Allow",
            "Action": "logs:CreateLogGroup",
            "Resource": "arn:aws:logs:us-east-2:111122223333:*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:DeleteObject"
            ],
            "Resource": "arn:aws:s3:::sctest-exploit-credlist/*"
        }
    ]
}
```

Deploy the lambda, either via the commandline (see below) OR interactivley in the console. For my project I actually used a S3 bucket that holds my zip-file and deploy from there.

```
aws lambda create-function --function-name credReader \
--runtime provided.al2023 --handler bootstrap \
--architectures arm64 \
--role arn:aws:iam::111122223333:role/credReaderLambda \
--zip-file fileb://credReader.zip
```

# To build for the local console:

I prefer to have a local test harness for doing quick-and-dirty testing on changes before moving to the cloud; it saves me time. For this, I use the recommended golang package structure to define a console app under the `cmd` sub directory, that when compiled gives me reasonable console-based test harness. For this project, the test harness also requires use of a local dynamodb instance (if you want to test storing the data); see the `dynamoddb-local` project in the provided package.

```
go build -o credreader ./cmd/console/main.go
```
Executing credreader with no arguments will look for the credential file in `./test/challenge_creds.txt` and only print to screen.

You may specify the file location using `-credfile <filename>`.

Additionally, if you have a local dynamodb instance setup and specify `-localdb`, it will attempt to write these entries into that dynamodb found at `localhost:8000`.