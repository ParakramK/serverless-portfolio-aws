import datetime
import json
import os
import uuid

import boto3

dynamodb = boto3.resource("dynamodb")

CORS = {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Headers": "Content-Type,Authorization",
    "Access-Control-Allow-Methods": "POST,OPTIONS",
}


def lambda_handler(event, context):
    table = dynamodb.Table(os.environ["TABLE_NAME"])
    claims = ((event.get("requestContext") or {}).get("authorizer") or {}).get(
        "claims", {}
    )
    created_by = claims.get("email") or claims.get("cognito:username") or "Anonymous"
    try:
        body = json.loads(event.get("body") or "{}")
    except BaseException:
        return {"statusCode": 400, "headers": CORS, "body": "Invalid input"}
    item = {
        "postId": str(uuid.uuid4()),
        "title": body.get("title", ""),
        "content": body.get("content", ""),
        "createdBy": created_by,
        "createdAt": datetime.datetime.utcnow().isoformat() + "Z",
    }
    try:
        table.put_item(Item=item)
        return {"statusCode": 201, "headers": CORS, "body": json.dumps(item)}
    except BaseException as e:
        return {"statusCode": 500, "headers": CORS, "body": str(e)}
