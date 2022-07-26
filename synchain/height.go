package main

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	db "github.com/guowenshuai/lotus_tool/db/mongo"
	"github.com/guowenshuai/lotus_tool/types"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

func getSyncConfig(ctx context.Context) (*types.SyncConfig, error) {
	confCol := globalDB.Collection(db.ConfigCollectionName)
	res := confCol.FindOne(ctx,  bson.D{{"height", bson.D{{"$gt", 0}}}})
	if res.Err() != nil {
		return nil, res.Err()
	}
	syncConfig := types.SyncConfig{}
	if err := res.Decode(&syncConfig); err != nil {
		return nil, err
	}
	return &syncConfig, nil
}

func getSyncHeight(ctx context.Context) (abi.ChainEpoch, error){
	syncConf, err := getSyncConfig(ctx)
	if err != nil {
		return 0, err
	}
	return syncConf.Height, nil
}

func setSyncHeight(ctx context.Context, next abi.ChainEpoch) {
	confCol := globalDB.Collection(db.ConfigCollectionName)
	if current, err := getSyncConfig(ctx); err != nil {
		// do insert
		logrus.Warnf("get sync conf err %s\n", err.Error())
		confCol.InsertOne(ctx, types.SyncConfig{
			Height: next,
		})
	} else {
		// do update
		_, err := confCol.UpdateOne(ctx, bson.D{{"height", current.Height}}, bson.D{{"$set", bson.M{"height": next}}})
		if err != nil {
			logrus.Errorf("findOneAndUpdate %s", err.Error())
			panic("")
		}
	}
}

func heightHasSynced(ctx context.Context, height abi.ChainEpoch) bool {
	blockCol := globalDB.Collection(db.BlockCollectionName)
	// matchPip := bson.D{{"$group", bson.D{{"_id", 0}, {"height", bson.D{{"$max", "$height"}}}}}}
	// cursor, err := blockCol.Aggregate(ctx, mongo.Pipeline{matchPip})
	// if err != nil {
	// 	return false
	// }
	//

	res := blockCol.FindOne(ctx, bson.D{{"height", height}})
	if res == nil {
		return false
	}
	if res.Err() != nil {
		return false
	}
	return true
}