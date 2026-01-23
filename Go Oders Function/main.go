package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const tableName = "Orders"

var dbClient *dynamodb.Client

// Order model
type Order struct {
	OrderID      string `json:"orderId"`
	CustomerName string `json:"customerName"`
	Product      string `json:"product"`
	Quantity     int    `json:"quantity"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
}

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}
	dbClient = dynamodb.NewFromConfig(cfg)
}

// Lambda handler
func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	switch req.HTTPMethod {

	case http.MethodPost:
		return createOrder(ctx, req)

	case http.MethodGet:
		// /orders/{orderId}
		if orderId, ok := req.PathParameters["orderId"]; ok && orderId != "" {
			return getOrderByID(ctx, orderId)
		}
		// /orders
		return getAllOrders(ctx)

	default:
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusMethodNotAllowed,
			Body:       "Method not allowed",
		}, nil
	}
}

// ---------- POST: Create Order ----------
func createOrder(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var order Order
	if err := json.Unmarshal([]byte(req.Body), &order); err != nil {
		return response(http.StatusBadRequest, "Invalid request body")
	}

	if order.CreatedAt == "" {
		order.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	_, err := dbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"orderId":      &types.AttributeValueMemberS{Value: order.OrderID},
			"customerName": &types.AttributeValueMemberS{Value: order.CustomerName},
			"product":      &types.AttributeValueMemberS{Value: order.Product},
			"quantity":     &types.AttributeValueMemberN{Value: intToString(order.Quantity)},
			"status":       &types.AttributeValueMemberS{Value: order.Status},
			"createdAt":    &types.AttributeValueMemberS{Value: order.CreatedAt},
		},
	})
	if err != nil {
		log.Println("PutItem error:", err)
		return response(http.StatusInternalServerError, "Failed to create order")
	}

	return response(http.StatusCreated, "Order created successfully")
}

// ---------- GET: All Orders ----------
func getAllOrders(ctx context.Context) (events.APIGatewayProxyResponse, error) {

	out, err := dbClient.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		log.Println("Scan error:", err)
		return response(http.StatusInternalServerError, "Failed to fetch orders")
	}

	orders := []Order{}
	for _, item := range out.Items {
		orders = append(orders, mapToOrder(item))
	}

	body, _ := json.Marshal(orders)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(body),
	}, nil
}

// ---------- GET: Order by ID ----------
func getOrderByID(ctx context.Context, orderId string) (events.APIGatewayProxyResponse, error) {

	out, err := dbClient.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("orderId = :oid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":oid": &types.AttributeValueMemberS{Value: orderId},
		},
	})
	if err != nil {
		log.Println("Query error:", err)
		return response(http.StatusInternalServerError, "Failed to fetch order")
	}

	if len(out.Items) == 0 {
		return response(http.StatusNotFound, "Order not found")
	}

	orders := []Order{}
	for _, item := range out.Items {
		orders = append(orders, mapToOrder(item))
	}

	body, _ := json.Marshal(orders)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(body),
	}, nil
}

// ---------- Helpers ----------
func response(status int, msg string) (events.APIGatewayProxyResponse, error) {
	body, _ := json.Marshal(map[string]string{"message": msg})
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       string(body),
	}, nil
}

func intToString(i int) string {
	return strconv.Itoa(i)
}

func mapToOrder(item map[string]types.AttributeValue) Order {
	return Order{
		OrderID:      item["orderId"].(*types.AttributeValueMemberS).Value,
		CustomerName: item["customerName"].(*types.AttributeValueMemberS).Value,
		Product:      item["product"].(*types.AttributeValueMemberS).Value,
		Quantity:     atoi(item["quantity"].(*types.AttributeValueMemberN).Value),
		Status:       item["status"].(*types.AttributeValueMemberS).Value,
		CreatedAt:    item["createdAt"].(*types.AttributeValueMemberS).Value,
	}
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func main() {
	lambda.Start(handler)

	// test for getting order by order_id
	// event := events.APIGatewayProxyRequest{
	// 	HTTPMethod: "GET",
	// 	PathParameters: map[string]string{
	// 		"orderId": "d9dcde4b-b163-4176-8d1a-f49d270a2f5e",
	// 	},
	// }

	// to get all orders
	// event := events.APIGatewayProxyRequest{
	// 	HTTPMethod: "GET",
	// }

	//to post an order
	// event := events.APIGatewayProxyRequest{
	// 	HTTPMethod: "POST",
	// 	Body: `{
	// 		"orderId": "20221",
	// 		"customerName": "Siri",
	// 		"product": "Lunch Box",
	// 		"quantity": 1,
	// 		"status": "CREATED"
	// 	}`,
	// }

	// resp, err := handler(context.Background(), event)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Println("STATUS:", resp.StatusCode)
	// log.Println("BODY:", resp.Body)
}
