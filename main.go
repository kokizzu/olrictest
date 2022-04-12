package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/config"
	"github.com/buraksezer/olric/query"
	"github.com/hashicorp/memberlist"
	"github.com/kokizzu/gotro/L"
	"github.com/kokizzu/gotro/S"
	"github.com/kokizzu/id64"
)

func main() {
	serviceId := id64.SID()

	c := config.New("local")

	ipSuffix := rand.Int()%250 + 1
	nodeIp := fmt.Sprintf("127.1.2.%d", ipSuffix)

	{ // comment when trying docker
		mc := memberlist.DefaultLocalConfig()
		mc.BindAddr = nodeIp
		mc.AdvertiseAddr = `127.255.255.255`
		c.MemberlistConfig = mc
		c.BindAddr = nodeIp
	}

	const fName = `/tmp/1.txt`
	raw := L.ReadFile(fName)
	ipList := S.Split(raw, ` `)
	raw += ` ` + nodeIp
	L.CreateFile(fName, raw)
	for _, ip := range ipList {
		if ip == `` {
			continue
		}
		c.Peers = append(c.Peers, ip)
	}
	defer func() {
		raw := L.ReadFile(fName)
		ipList := S.Split(raw, ` `)
		raw = ``
		for _, ip := range ipList {
			if ip == nodeIp {
				continue
			}
			raw += ` ` + ip
		}
		L.CreateFile(fName, raw)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	c.Started = func() {
		defer cancel()
		log.Println("Olric is up, ID: " + serviceId)
	}
	db, err := olric.New(c)
	L.PanicIf(err, `olric.New`)
	go func() {
		// Call Start at background. It's a blocker call.
		err = db.Start()
		L.PanicIf(err, `db.Start`)
	}()

	<-ctx.Done()

	defer func() {
		err := db.Shutdown(context.Background())
		L.PanicIf(err, `db.Shutdown`)
	}()

	// topic test
	const topic = `membersheep`
	dt, err := db.NewDTopic(topic, 0, olric.UnorderedDelivery)
	L.PanicIf(err, `db.NewDTopic`)
	listenerID, err := dt.AddListener(func(msg olric.DTopicMessage) {
		fmt.Println(topic, msg)
	})
	L.PanicIf(err, `dt.AddListener`)
	defer func() {
		err := dt.RemoveListener(listenerID)
		L.PanicIf(err, `dt.RemoveListener`)
	}()
	err = dt.Publish(serviceId)
	L.PanicIf(err, `dt.Publish`)

	// kv test
	const prefix = `member:`
	dm, err := db.NewDMap("membermap")
	L.PanicIf(err, `db.NewDMap`)

	counter, err := dm.Incr(`counter`, 1)
	L.PanicIf(err, `dm.Incr`)

	err = dm.Put(prefix+serviceId, counter)
	L.PanicIf(err, `dm.Put`)

	cursor, err := dm.Query(query.M{
		"$onKey": query.M{
			"$regexMatch": prefix,
		},
	})
	L.PanicIf(err, `dm.Query`)
	err = cursor.Range(func(k string, v interface{}) bool {
		fmt.Println(k, v)
		return true
	})
	L.PanicIf(err, `cursor.Range`)

	time.Sleep(10 * time.Second)
}
