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
    try:
        resp = table.scan()
        return {
            "statusCode": 200,
            "headers": CORS,
            "body": json.dumps(resp.get("Items", [])),
        }
    except Exception as e:
        return {"statusCode": 500, "headers": CORS, "body": str(e)}
