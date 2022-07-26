package db

import (
	"context"
	"fmt"
	"reflect"

	"github.com/guowenshuai/lotus_tool/types"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var localCli *mongo.Client

func Connect(ctx context.Context, config *types.Config) (*mongo.Database, error) {
	// 设置客户端连接配置
	clientOptions := options.Client().ApplyURI("mongodb://" + config.Mongodb.Server)
	if !config.Mongodb.NoAuth {
		clientOptions.SetAuth(options.Credential{
			Username:    config.Mongodb.Username,
			Password:    config.Mongodb.Password,
			PasswordSet: false,
		})
	}
	// 连接到MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return nil, err
	}
	localCli = client
	// 检查连接
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return nil, err
	}
	fmt.Println("Connected to MongoDB!")
	database := client.Database(config.Mongodb.Database)
	// createIndex(database)
	return database, nil
}

func Disconnect() error {
	if localCli == nil {
		return nil
	}
	return localCli.Disconnect(context.Background())
}

const (
	ConfigCollectionName   = "config"
	MessagesCollectionName = "messages"
	BlockCollectionName    = "blocks"
)

type idx struct {
	colName string
	mod     mongo.IndexModel
}

func createIndex(database *mongo.Database) error {
	todo := []idx{
		{
			ConfigCollectionName,
			mongo.IndexModel{
				Keys: bson.M{
					"height": 1,
				},
				Options: options.Index().SetName("config"),
			},
		}, {
			MessagesCollectionName,
			mongo.IndexModel{
				Keys: bson.M{
					"msgcid": 1,
				},
				Options: options.Index().SetUnique(true).SetName("message_cid"),
			},
		}, {
			MessagesCollectionName,
			mongo.IndexModel{
				Keys: bson.M{
					"height": 1,
				},
				Options: options.Index().SetName("message_height"),
			},
		}, {
			MessagesCollectionName,
			mongo.IndexModel{
				Keys: bson.M{
					"from": 1,
				},
				Options: options.Index().SetName("wallet_from"),
			},
		}, {
			BlockCollectionName,
			mongo.IndexModel{
				Keys: bson.M{
					"height": 1,
				},
				Options: options.Index().SetUnique(true).SetName("block_height"),
			},
		},
	}
	for _, i := range todo {
		initIndex(database, i)
	}
	return nil
}

func initIndex(database *mongo.Database, info idx) error {
	database.CreateCollection(context.Background(), info.colName)

	collection := database.Collection(info.colName)
	ctx := context.Background()
	// cur, err := collection.Indexes().List(ctx)
	// if cur.All()
	ind, err := collection.Indexes().CreateOne(ctx, info.mod)
	// Check if the CreateOne() method returned any errors
	if err != nil {
		logrus.Errorf("Indexes().CreateOne() ERROR: %s", err)
		return err
	} else {
		// API call returns string of the index name
		logrus.Println("CreateOne() index:", ind)
		logrus.Printf("CreateOne() type: %s", reflect.TypeOf(ind))
	}
	return nil
}
