package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Post struct {
	PostID    string `json:"postId" dynamodbav:"postId"`
	Title     string `json:"title" dynamodbav:"title"`
	Content   string `json:"content" dynamodbav:"content"`
	CreatedBy string `json:"createdBy" dynamodbav:"createdBy"`
	CreatedAt string `json:"createdAt" dynamodbav:"createdAt"`
}

var dbClient *dynamodb.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	dbClient = dynamodb.NewFromConfig(cfg)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	tableName := os.Getenv("TABLE_NAME")

	corsHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Headers": "Content-Type,Authorization",
		"Access-Control-Allow-Methods": "GET,OPTIONS",
	}

	id, ok := request.PathParameters["id"]
	if !ok || id == "" {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest, Headers: corsHeaders, Body: "Missing id path parameter"}, nil
	}

	result, err := dbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &tableName,
		Key: map[string]types.AttributeValue{
			"postId": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Headers: corsHeaders, Body: err.Error()}, nil
	}

	if result.Item == nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusNotFound, Headers: corsHeaders, Body: "Post not found"}, nil
	}

	var post Post
	err = attributevalue.UnmarshalMap(result.Item, &post)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Headers: corsHeaders, Body: err.Error()}, nil
	}

	body, _ := json.Marshal(post)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    corsHeaders,
		Body:       string(body),
	}, nil
}

func main() {
	lambda.Start(handler)
}
