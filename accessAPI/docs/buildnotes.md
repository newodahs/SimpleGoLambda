# To build the lambda (deploying to an arm64 lambda):

This is our API for retrieving exploited credentials from the dynamodb table. This only has two endponts:
 1. `/v1/ping` => returns 'pong'; just a sanity 'I'm working' type call
 2. `/v1/credentials?filter={someFilter}` where someFilter is a full email address or domain

I have code for scanning the table as well, however, it is not currently implemented as a route.

Build the lambda:

```
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap ./cmd/lambda/main.go
```
```
zip accessapi.zip bootstrap
```
You'll need to have a lambda execution role for this next step (below, I'm using the name accessapiLambda) with a reasonable policy for dynamodb and cloudwatch.  Here is mine:
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
                "dynamodb:Scan"
            ],
            "Resource": "arn:aws:dynamodb:us-east-2:111122223333:table/exploitedCredentials"
        },
        {
            "Sid": "WriteLogStreamsAndGroups",
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": "*"
        },
        {
            "Sid": "CreateLogGroup",
            "Effect": "Allow",
            "Action": "logs:CreateLogGroup",
            "Resource": "*"
        }
    ]
}
```

Deploy the lambda, either via the commandline (see below) OR interactivley in the console. For my project I actually used a S3 bucket that holds my zip-file and deploy from there.

```
aws lambda create-function --function-name accessapi \
--runtime provided.al2023 --handler bootstrap \
--architectures arm64 \
--role arn:aws:iam::111122223333:role/accessapiLambda \
--zip-file fileb://accessapi.zip
```

Next you'll need an API-Gateway route setup; easy way is to just standup a simple HTTP API in API-GW and with the following notes:
 * Add integration during creation for lambda
 * Route: `ANY => /{proxy+}`
 * Attach the integration to the route, selecting the above accesspi lambda that you've created
  * IMPORTANT: This version of the code is currently using the 1.0 payload format, be sure to set that explicitly during the integration setup

You can verify success by `curl`ing out to the deployed api using the `v1/ping` route for simplicity.

# To build for the local console:

I prefer to have a local test harness for doing quick-and-dirty testing on changes before moving to the cloud; it saves me time. For this, I use the recommended golang package structure to define a console app under the `cmd` sub directory, that when compiled gives me reasonable console-based test harness. For this project, the test harness also requires use of a local dynamodb instance to be able to retrieve the data; see the `dynamoddb-local` project in the provided package.

```
go build -o accessapi ./cmd/console/main.go
```

Assumes connecting to a local dynamodb instance at: `localhost:8000`.