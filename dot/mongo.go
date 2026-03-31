package dot

import (
	"context"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/reqwest/dnscache"
	"github.com/anxiwuyanzu/openscraper-framework/spider-common-go/v4/util"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

func NewMongoClient(addr string) *mongo.Client {
	if u, ok := Conf().Mongo.Others[addr]; ok {
		addr = u
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(addr).SetDialer(dnscache.NewDialer(nil)))
	if err != nil {
		Logger().WithError(err).Panic("failed to create mongo client")
	}

	err = client.Connect(context.Background())
	if err != nil {
		Logger().WithError(err).Panic("failed to connect mongo")
	}
	return client
}

type MongoBulkWriter struct {
	sync.Mutex
	logger     *logrus.Entry
	operations chan mongo.WriteModel
	options    *options.BulkWriteOptions
	collection *mongo.Collection
	stopCh     chan bool
	batchSize  int
	flushMin   int
	logVerbose bool
	wg         sync.WaitGroup
}

func NewMongoBulkWriter(client *mongo.Client, db, coll string) *MongoBulkWriter {
	return NewMongoBulkWriterWithConfig(client, db, coll, nil)
}

func NewMongoBulkWriterWithConfig(client *mongo.Client, db, coll string, cfg *MongoConfig) *MongoBulkWriter {
	if cfg == nil {
		cfg = &Conf().Mongo
	}
	collection := client.Database(db).Collection(coll)
	logger := Logger().WithFields(logrus.Fields{"coll": coll, "db": db})

	w := &MongoBulkWriter{
		logger:     logger,
		options:    options.BulkWrite().SetOrdered(false),
		collection: collection,
		operations: make(chan mongo.WriteModel, cfg.BatchWriteSize*cfg.FlushWorker),
		stopCh:     make(chan bool, 1),
		batchSize:  cfg.BatchWriteSize,
		flushMin:   cfg.FlushMin,
		logVerbose: cfg.LogVerbose,
		wg:         sync.WaitGroup{},
	}

	for i := 0; i < cfg.FlushWorker; i++ {
		w.wg.Add(1)
		go w.flushLoop(cfg)
	}
	EnsureClose(w.Close)

	return w
}

func (m *MongoBulkWriter) Add(w mongo.WriteModel) {
	m.operations <- w
}

// Close manually close and waiting flush finish
func (m *MongoBulkWriter) Close() {
	util.SafeClose(m.stopCh)
	m.wg.Wait()
}

func (m *MongoBulkWriter) flushLoop(cfg *MongoConfig) {
	defer m.wg.Done()

	interval := cfg.FlushIntervalSec
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	var ops []mongo.WriteModel
	for {
		select {
		case w := <-m.operations:
			ops = append(ops, w)
			if len(ops) >= m.batchSize {
				go m.flush(ops)
				ops = ops[:0]
			}
		case <-ticker.C:
			if len(ops) > m.flushMin {
				go m.flush(ops)
				ops = ops[:0]
			}
		case <-std.Context().Done():
			m.logger.Info("mongo stop")
			if len(ops) > 0 {
				m.flush(ops)
				ops = ops[:0]
			}
			return
		case <-m.stopCh:
			m.logger.Info("mongo stop")
			if len(ops) > 0 {
				m.flush(ops)
				ops = ops[:0]
			}
			return
		}
	}
}

func (m *MongoBulkWriter) flush(ops []mongo.WriteModel) {
	if len(ops) == 0 {
		return
	}

	start := time.Now()
	_, err := m.collection.BulkWrite(context.Background(), ops, m.options)
	took := time.Now().Sub(start).Milliseconds()
	fields := logrus.Fields{
		"took": took,
		"num":  len(ops),
	}

	if err != nil {
		m.logger.WithError(err).WithFields(fields).Error("error when flushing to mongo")
	} else {
		if len(m.operations) > 500 {
			fields["heap"] = len(m.operations)
		}
		if m.logVerbose {
			m.logger.WithFields(fields).Info("successfully flushed to mongo")
		}
	}
}
