package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
)

type Post struct {
	PostID    string `json:"postId" dynamodbav:"postId"`
	Title     string `json:"title" dynamodbav:"title"`
	Content   string `json:"content" dynamodbav:"content"`
	CreatedBy string `json:"createdBy" dynamodbav:"createdBy"` 
	CreatedAt string `json:"createdAt" dynamodbav:"createdAt"`
}

type CreatePostInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
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
		"Access-Control-Allow-Methods": "POST,OPTIONS",
	}

	createdBy := "Anonymous"
	if authorizer, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
		if email, exists := authorizer["email"].(string); exists && email != "" {
			createdBy = email
		} else if username, exists := authorizer["cognito:username"].(string); exists && username != "" {
			createdBy = username
		}
	}

	var input CreatePostInput
	err := json.Unmarshal([]byte(request.Body), &input)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest, Headers: corsHeaders, Body: "Invalid input"}, nil
	}

	newPost := Post{
		PostID:    uuid.New().String(),
		Title:     input.Title,
		Content:   input.Content,
		CreatedBy: createdBy, 
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	item, err := attributevalue.MarshalMap(newPost)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Headers: corsHeaders, Body: err.Error()}, nil
	}

	_, err = dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &tableName,
		Item:      item,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError, Headers: corsHeaders, Body: err.Error()}, nil
	}

	body, _ := json.Marshal(newPost)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Headers:    corsHeaders,
		Body:       string(body),
	}, nil
}
func main() {
	lambda.Start(handler)
}