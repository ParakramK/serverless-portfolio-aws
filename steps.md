# Serverless Portfolio - Replicate Steps (Floci-local first, Learner Lab AWS)

Stack: API Gateway + Lambda (Python, originally Go) + DynamoDB + Cognito + S3 website. Floci emulates AWS locally (`http://localhost.floci.io:4566`). Learner Lab account `771194017176`, region `us-east-1`, `StudentID=parakra` (truncated from `parakram` by lab/SAM).

## 0. Prereqs
- `aws` CLI, `sam` CLI, `floci`, Python 3.12, `zip`
- `floci --help`, `floci status`, `floci services` (need `s3,dynamodb,lambda,apigateway,cognito-idp,cloudfront`)

## 1. Floci-local base
```bash
floci status || floci start
floci env
eval $(floci env)
export TABLE_NAME=PostsTable
export SITE_BUCKET=portfolio-site-local
export API_ID=446399cadb  # after create-rest-api
```
`floci env` gives `AWS_ENDPOINT_URL=http://localhost.floci.io:4566`, `test/test`, `us-east-1`. Use `--endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION` for all local `aws` calls.

## 2. DynamoDB (floci)
```bash
aws dynamodb create-table --table-name $TABLE_NAME --attribute-definitions AttributeName=postId,AttributeType=S --key-schema AttributeName=postId,KeyType=HASH --billing-mode PAY_PER_REQUEST --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION
aws dynamodb put-item --table-name $TABLE_NAME --item '{"postId":{"S":"test-123"},"title":{"S":"Hello"},"content":{"S":"World"},"createdBy":{"S":"local"},"createdAt":{"S":"2026-09-21T00:00:00Z"}}' --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION
aws dynamodb get-item --table-name $TABLE_NAME --key '{"postId":{"S":"test-123"}}' --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION
```

## 3. Lambdas (floci, Go initially, then Python)
Go required zip root file named exactly `bootstrap` for `provided.al2023`:
```bash
GOOS=linux GOARCH=amd64 go build -o /tmp/getPosts-bootstrap ./src/getPosts
mkdir -p /tmp/pkg-getPosts && cp /tmp/getPosts-bootstrap /tmp/pkg-getPosts/bootstrap
zip -j /tmp/getPosts.zip /tmp/pkg-getPosts/bootstrap
aws lambda create-function --function-name getPosts --runtime provided.al2023 --handler bootstrap --role arn:aws:iam::000000000000:role/lambda-role --zip-file fileb:///tmp/getPosts.zip --environment Variables={TABLE_NAME=PostsTable} --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION
```
Repeat for `getPostById`, `createPosts`. Python final: `backend/get_posts.py`, `get_post_by_id.py`, `create_post.py` (boto3 Scan/GetItem/PutItem, CORS, `createdBy` from `authorizer.claims`).

## 4. API Gateway (floci manual, SAM transform Auth unsupported)
```bash
aws apigateway create-rest-api --name portfolio-api --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION # -> 446399cadb, root 29cfbdd5
aws apigateway create-resource --rest-api-id $API_ID --parent-id $ROOT_ID --path-part posts # -> 4127bf66
aws apigateway create-resource --rest-api-id $API_ID --parent-id 4127bf66 --path-part "{id}" # -> 502abead
aws cognito-idp create-user-pool --pool-name portfolio-pool --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION # -> us-east-1_acf475dd0
aws cognito-idp create-user-pool-client --user-pool-id <pool> --client-name web --no-generate-secret --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION # -> 1983e5fb...
aws apigateway create-authorizer --rest-api-id $API_ID --name cognito --type COGNITO_USER_POOLS --provider-arns arn:aws:cognito-idp:us-east-1:000000000000:userpool/<pool> --identity-source method.request.header.Authorization --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION # -> e71e5c
aws apigateway put-method --rest-api-id $API_ID --resource-id 4127bf66 --http-method GET --authorization-type NONE --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION
aws apigateway put-method --rest-api-id $API_ID --resource-id 502abead --http-method GET --authorization-type NONE --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION
aws apigateway put-method --rest-api-id $API_ID --resource-id 4127bf66 --http-method POST --authorization-type COGNITO_USER_POOLS --authorizer-id e71e5c --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION
# integrations AWS_PROXY to lambda URIs, add-permission with quoted 'arn:.../*/*' (zsh glob!), create-deployment prod
curl http://localhost.floci.io:4566/restapis/$API_ID/prod/_user_request_/posts
```

## 5. Frontend (S3 + CloudFront floci, runtime config)
`public/index.html` loads `main.js`. `public/main.js` moved from inline + added `fetchPostById(id)` -> `GET ${API_URL}/${id}`. No hardcode: `public/config.json` with `API_URL,USER_POOL_ID,CLIENT_ID,COGNITO_ENDPOINT`. `main.js:loadConfig()` fetches `config.json` first. `login()` passes `endpoint: COGNITO_ENDPOINT` to hit floci, not real AWS.
```bash
aws s3api create-bucket --bucket $SITE_BUCKET --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION
aws s3 sync public/ s3://$SITE_BUCKET --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION
aws cloudfront create-distribution --origin-domain-name $SITE_BUCKET.s3.localhost.floci.io --default-root-object index.html --endpoint-url $AWS_ENDPOINT_URL --region $AWS_DEFAULT_REGION
```

## 6. SAM template (Learner Lab AWS)
`template.yaml`: DynamoDB, Cognito (+`ALLOW_ADMIN_USER_PASSWORD_AUTH`), `BlogApi` CORS + `CognitoAuth` + `AddDefaultAuthorizerToCorsPreflight:false`, 3x Python `backend/` `python3.12`, `Role: arn:aws:iam::771194017176:role/LabRole`, S3 website + public policy, no CloudFront (lab denies `CreateDistribution`). Outputs `ApiUrl,UserPoolId,ClientId,WebsiteURL,TableName`.
```bash
unset AWS_ENDPOINT_URL
export AWS_PROFILE=lab
python3 configure_lab.py  # getpass local, sets --profile lab
aws sts get-caller-identity --region us-east-1
sam validate --template template.yaml --lint
sam build --template template.yaml
sam deploy --template template.yaml --stack-name serverless-portfolio --capabilities CAPABILITY_IAM --region us-east-1 --profile lab --resolve-s3 --parameter-overrides StudentID=parakra
sam list stack-outputs --stack-name serverless-portfolio --region us-east-1 --profile lab
# update public/config.json to ApiUrl/UserPoolId/ClientId + COGNITO_ENDPOINT=https://cognito-idp.us-east-1.amazonaws.com
aws s3 sync public/ s3://parakra-portfolio-bucket --region us-east-1 --profile lab
# create user, login via site
aws cognito-idp admin-create-user --user-pool-id <new> --username you@example.com --user-attributes Name=email,Value=you@example.com --message-action SUPPRESS --region us-east-1 --profile lab
aws cognito-idp admin-set-user-password --user-pool-id <new> --username you@example.com --password 'Temp123!' --permanent --region us-east-1 --profile lab
```

## Issues faced + fixes
- `aws: command not found` -> `brew install awscli`.

- `floci` has no `invoke` (emulator host) -> use `aws ... --endpoint-url` + `sam local` needs `template.yaml`.

- `SAM Auth not supported for explicit REST APIs (Floci)` -> manual CLI Gateway + authorizer for local; template Auth only for real AWS.

- Browser login `Network error` -> SDK defaults to AWS; added `COGNITO_ENDPOINT` to `config.json` + `endpoint` in `poolData`. Floci CORS still flaky -> CLI `admin-initiate-auth` token + curl POST bypass for local proof.
- `createdBy:Anonymous` locally -> floci authorizer `claims` shape differs from AWS; accepted locally, real AWS maps `email/cognito:username`.
- `InvalidClientTokenId` / `Unable to locate credentials` -> floci `test/test` vs real lab keys; `unset AWS_ENDPOINT_URL`, `export AWS_PROFILE=lab`, use 3 keys incl `SESSION_TOKEN` via `configure_lab.py` (getpass, never paste in chat).
- `CloudFront CreateDistribution AccessDenied` in lab -> removed distribution, S3 website only.
- `ROLLBACK_COMPLETE cannot be updated` -> `delete-stack` + `wait`.
- `DELETE_FAILED FrontendBucket not empty` -> `aws s3 rm --recursive` before delete.
- `StudentID parakram -> parakra` truncation in outputs/names -> use `parakra-*` fn/bucket/table names as deployed.
- `502 Internal server error` prod -> CloudWatch `no bootstrap`; fixed via Python switch + `sam build`.
- `404 NoSuchKey index.html`, `NS_UNKNOWN_HOST /posts` after delete -> bucket empty/stack gone; re-deploy then `s3 sync`.
