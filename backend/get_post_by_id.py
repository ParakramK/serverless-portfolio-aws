import json
import os

import boto3

dynamodb = boto3.resource("dynamodb")

CORS = {
    "Access-Control-Allow-Origin": "*",
    "Access-Control-Allow-Headers": "Content-Type,Authorization",
    "Access-Control-Allow-Methods": "GET,OPTIONS",
}


def lambda_handler(event, context):
    table = dynamodb.Table(os.environ["TABLE_NAME"])
    post_id = (event.get("pathParameters") or {}).get("id")
    if not post_id:
        return {"statusCode": 400, "headers": CORS, "body": "Missing id path parameter"}
    try:
        resp = table.get_item(Key={"postId": post_id})
        item = resp.get("Item")
        if not item:
            return {"statusCode": 404, "headers": CORS, "body": "Post not found"}
        return {"statusCode": 200, "headers": CORS, "body": json.dumps(item)}
    except Exception as e:
        return {"statusCode": 500, "headers": CORS, "body": str(e)}
