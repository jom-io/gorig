package domainx

import (
	"context"
	"fmt"
	"github.com/jom-io/gorig/apix/load"
	"github.com/jom-io/gorig/global/errc"
	configure "github.com/jom-io/gorig/utils/cofigure"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/sys"
	"github.com/qiniu/qmgo"
	qoptions "github.com/qiniu/qmgo/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"strings"
	"time"
)

var MongoDBServ = &mongoDBService{}

type mongoDBService struct {
}

func init() {
	RegisterDBService(Mongo, MongoDBServ)
}

const configName = "mongo"

var qmMDBMap = make(map[string]*qmgo.Client)

func UseMongoDbConn(dbname string) *qmgo.Client {
	if dbname == "" {
		logger.Logger.Error(fmt.Sprintf(errc.ErrorsDBInitFail, dbname))
		return nil
	}
	dbname = strings.ToLower(dbname)
	if _, ok := qmMDBMap[dbname]; !ok {
		logger.Logger.Error(fmt.Sprintf(errc.ErrorsNotInitGlobalPointer, dbname))
		return nil
	}
	return qmMDBMap[dbname]
}

func (*mongoDBService) Start() error {
	sys.Info(" * DB Mongo service startup on: ", Mongo)
	sub := configure.GetSub(configName)
	if len(sub) > 0 {
		for k, _ := range sub {
			if err := initMgoDB(k); err != nil {
				return errors.Sys(err.Error())
			}
		}
	} else {
		sys.Info(" * No mongo configuration found")
		return nil
	}
	return nil
}

func getColl(c *Con) (*qmgo.Collection, *errors.Error) {
	mDb := c.MDB
	gDdb, e := ddbCon(mDb, c.DBName)
	if e != nil {
		return nil, e
	}
	gColl, e := collCon(gDdb, c.TableName())
	if e != nil {
		return nil, e
	}
	return gColl, nil
}

func (s *mongoDBService) Migrate(con *Con, tableName string, value ConTable, indexList []Index) error {
	if con.MDB == nil {
		return errors.Sys("Migrate: mdb is nil")
	}
	if coll, e := getColl(con); e != nil {
		return e
	} else {
		for _, index := range indexList {
			var unique *bool
			if index.IdxType == Unique {
				unique = new(bool)
				*unique = true
			}
			// 如果不是con或者options开头的字段，需要加上data.
			if index.Fields[0] != "con" && index.Fields[0] != "options" {
				for i, v := range index.Fields {
					index.Fields[i] = "data." + v
				}
			}
			if index.IdxType == Spatial2D {
				mColl, err := coll.CloneCollection()
				if err != nil {
					return err
				}
				if _, err = mColl.Indexes().CreateOne(context.Background(), mongo.IndexModel{
					Keys: bson.D{
						{Key: index.Fields[0], Value: "2dsphere"},
					},
				}); err != nil {
					return err
				}
				continue
			}
			if err := coll.CreateIndexes(context.Background(), []qoptions.IndexModel{{
				Key: index.Fields,
				IndexOptions: &options.IndexOptions{
					Unique: unique,
				},
			}}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (*mongoDBService) End() error {
	for k, client := range qmMDBMap {
		if err := client.Close(context.Background()); err != nil {
			logger.Logger.Error("close mongo.Client failed", zap.Error(err))
		}
		delete(qmMDBMap, k)
	}
	sys.Info(" * Mongo service shutdown on: ", Mongo)
	return nil
}

func initMgoDB(dbname ...string) error {
	for _, db := range dbname {
		sys.Info(" * Init mongo db: ", db)
		path := configName + "." + db
		// 读取配置
		uri, err := configure.MustGetString(path + ".uri")
		if err != nil {
			return err
		}

		qConfig := &qmgo.Config{Uri: uri}

		needAuth := configure.GetBool(path+".auth.need", false)
		if needAuth {
			source, err := configure.MustGetString(path + ".auth.source")
			if err != nil {
				return err
			}
			user, err := configure.MustGetString(path + ".auth.user")
			if err != nil {
				return err
			}
			pwd, err := configure.MustGetString(path + ".auth.password")
			if err != nil {
				return err
			}
			qConfig.Auth = &qmgo.Credential{
				AuthSource: source,
				Username:   user,
				Password:   pwd,
			}
		}

		clientOptions := options.Client().ApplyURI(uri)
		clientOptions.SetRetryWrites(configure.GetBool(path+".retry.writes", true)).
			SetRetryReads(configure.GetBool(path+".retry.reads", true)).
			SetMaxPoolSize(configure.GetUint64(path+".pool.max", 10)).
			SetMinPoolSize(configure.GetUint64(path+".pool.min", 1)).
			SetMaxConnIdleTime(configure.GetDuration(path+".conn.idle.time.max", 10) * time.Second)

		mgClient, e := qmgo.NewClient(context.Background(), qConfig, qoptions.ClientOptions{ClientOptions: clientOptions})
		if e != nil {
			return e
		}
		e = mgClient.Ping(30)
		if e != nil {
			return e
		}
		db = strings.ToLower(db)
		qmMDBMap[db] = mgClient
	}
	return nil
}

func ddbCon(mdb *qmgo.Client, db string) (*qmgo.Database, *errors.Error) {
	if mdb == nil {
		return nil, errors.Sys("DDB: mdb is nil")
	}
	database := mdb.Database(configure.GetString(configName + "." + db + ".db.name"))
	if database == nil {
		return nil, errors.Sys("DDB: database is nil")
	}
	return database, nil
}

func collCon(database *qmgo.Database, collection string) (*qmgo.Collection, *errors.Error) {
	if database == nil {
		return nil, errors.Sys(fmt.Sprintf("Coll: database is nil"))
	}
	if collection == "" {
		return nil, errors.Sys(fmt.Sprintf("Coll: collection is empty"))
	}
	c := database.Collection(collection)
	if c == nil {
		return nil, errors.Sys(fmt.Sprintf("Coll: %s collection is nil", collection))
	}
	return c, nil
}

func (s *mongoDBService) GetByID(c *Con, id int64, result interface{}) error {
	gColl, e := getColl(c)
	if e != nil {
		return e
	}
	if err := gColl.Find(context.Background(), map[string]interface{}{"con.id": id}).One(result); err != nil {
		return err
	}
	return nil
}

func (s *mongoDBService) Save(c *Con, data Identifiable, newID int64) (id int64, error error) {
	if coll, e := getColl(c); e != nil {
		return 0, e
	} else {
		if c.GetID().NotNil() {
			c.SaveUpdateTime()
			mErr := coll.UpdateOne(context.Background(), bson.M{"con.id": c.ID}, bson.M{"$set": data})
			if mErr != nil {
				return 0, mErr
			}
			return c.ID, nil
		} else {
			// 生成id
			if newID > 0 {
				c.ID = newID
			} else {
				c.ID = c.GetID().GenerateID()
			}
			if c.SaveCreateTime != nil {
				c.SaveCreateTime()
			}
			if c.SaveUpdateTime != nil {
				c.SaveUpdateTime()
			}
			one, mErr := coll.InsertOne(context.Background(), data)
			if mErr != nil {
				return 0, mErr
			}
			if one.InsertedID == nil {
				return 0, errors.Sys("插入失败")
			}
			return c.ID, nil
		}
	}
}

func (s *mongoDBService) UpdatePart(c *Con, id int64, data map[string]interface{}) error {
	if coll, e := getColl(c); e != nil {
		return e
	} else {
		m := mapToBsonM(data)
		m["options.updateTime"] = time.Now()
		mErr := coll.UpdateOne(context.Background(), bson.M{"con.id": id}, bson.M{"$set": mapToBsonM(data)})
		return mErr
	}
}

func (s *mongoDBService) Delete(c *Con, data Identifiable) error {
	if coll, e := getColl(c); e != nil {
		return e
	} else {
		mErr := coll.Remove(context.Background(), bson.M{"con.id": data.GetID()})
		return mErr
	}
}

func mapToBsonM(m map[string]interface{}, prefixes ...string) bson.M {
	prefix := ""
	if len(prefixes) == 0 {
		prefix = "data."
	} else {
		for _, v := range prefixes {
			prefix += v + "."
		}
	}
	bm := bson.M{}
	for k, v := range m {
		key := prefix + k
		// 最后最后一个字符是.的话，去掉
		if strings.HasSuffix(key, ".") {
			key = key[:len(key)-1]
		}
		bm[key] = v
	}
	return bm
}

func sortMongoFields(s []*Sort) []string {
	sortList := make([]string, 0)
	if s != nil {
		for _, v := range s {
			order := ""
			if !v.Asc {
				order = "-"
			}
			pre := "data."
			if v.Prefix != "" {
				pre = v.Prefix + "."
			}
			sortList = append(sortList, order+pre+v.Field)
		}
	}
	sortList = append(sortList, "-con.id") // 默认按id倒序
	return sortList
}

// matchMqCond Mongo根据条件列表获取条件
func matchMongoCond(matchList []Match) map[string]interface{} {
	condition := make(map[string]interface{}, len(matchList))
	for _, match := range matchList {
		switch match.Type {
		case MEq:
			condition[match.Field] = match.Value
		case MLt:
			condition[match.Field] = bson.M{"$lt": match.Value}
		case MLte:
			condition[match.Field] = bson.M{"$lte": match.Value}
		case MGt:
			condition[match.Field] = bson.M{"$gt": match.Value}
		case MGte:
			condition[match.Field] = bson.M{"$gte": match.Value}
		case MLIKE:
			condition[match.Field] = bson.M{"$regex": match.Value}
		case MNE:
			condition[match.Field] = bson.M{"$ne": match.Value}
		case MIN:
			condition[match.Field] = bson.M{"$in": match.Value}
		case MNOTIN:
			condition[match.Field] = bson.M{"$nin": match.Value}
		case MNEmpty:
			condition[match.Field] = bson.M{"$exists": true, "$not": bson.M{"$size": 0}}
		case NearLoc:
			near := match.ToNearMatch()
			if near.Distance == 0 {
				near.Distance = 5000 * 1000
			}
			condition[match.Field] = bson.M{
				"$near": bson.M{
					"$geometry": bson.M{
						"type":        "Point",
						"coordinates": []float64{near.Lng, near.Lat},
					},
					"$maxDistance": near.Distance,
				},
			}
		default:
			condition[match.Field] = match.Value
		}
	}
	return condition
}
func (s *mongoDBService) FindByMatch(c *Con, matchList []Match, result interface{}, prefixes ...string) error {
	condition := matchMongoCond(matchList)
	if coll, e := getColl(c); e != nil {
		return e
	} else {
		mErr := coll.Find(context.Background(), mapToBsonM(condition, prefixes...)).Sort(sortMongoFields(c.Sort)...).Limit(10000).All(result)
		return mErr
	}
}

func (s *mongoDBService) GetByMatch(c *Con, matchList []Match, result interface{}) error {
	condition := matchMongoCond(matchList)
	if coll, e := getColl(c); e != nil {
		return e
	} else {
		mErr := coll.Find(context.Background(), mapToBsonM(condition)).Sort(sortMongoFields(c.Sort)...).One(result)
		return mErr
	}
}

func (s *mongoDBService) CountByMatch(c *Con, matchList []Match) (int64, error) {
	condition := matchMongoCond(matchList)
	if coll, e := getColl(c); e != nil {
		return 0, e
	} else {
		count, mErr := coll.Find(context.Background(), mapToBsonM(condition)).Count()
		if mErr != nil {
			return 0, mErr
		}
		return count, nil
	}
}

func (s *mongoDBService) FindByPageMatch(c *Con, matchList []Match, page *load.Page, total *load.Total, result interface{}, prefixes ...string) error {
	if coll, e := getColl(c); e != nil {
		return e
	} else {
		condition := matchMongoCond(matchList)
		var mErr error
		var count int64
		if page.LastID > 0 {
			m := mapToBsonM(condition, prefixes...)
			m["con.id"] = bson.M{"$lt": page.LastID}
			count, _ = coll.Find(context.Background(), m).Count()
			mErr = coll.Find(context.Background(), m).Sort(sortMongoFields(c.Sort)...).Limit(page.Size).All(result)
		} else {
			bsonM := mapToBsonM(condition, prefixes...)
			count, _ = coll.Find(context.Background(), bsonM).Count()
			skip := (page.Page - 1) * page.Size
			mErr = coll.Find(context.Background(), bsonM).Sort(sortMongoFields(c.Sort)...).Skip(skip).Limit(page.Size).All(result)
		}
		total.Set(count)
		if mErr != nil {
			return mErr
		}
		return nil
	}
}
