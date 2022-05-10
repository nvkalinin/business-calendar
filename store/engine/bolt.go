package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/nvkalinin/business-calendar/log"
	"github.com/nvkalinin/business-calendar/store"
	"go.etcd.io/bbolt"
)

const calBucket = "cal"

// Bolt хранят все данные в одном бакете (const calBucket).
// По ключу /<y>/<m> хранится JSON, описывающий все дни месяца. Оба ключа - числовые.
//
// Есть два паттерна использования данного сервиса: клиенты могут обращаться через REST API всякий раз, когда
// нужна информация о дне/неделе, либо запросить, например, год и закешировать у себя. Первое подходит, например,
// для вывода какого-нибудь UI календаря, последнее — для обработки большого количества данных.
//
// Хранение каждого года в отдельном ключе неудобно с т. з. отладки и не имеет преимуществ по производительности для
// случая обработки большого кол-ва данных. Хранение каждого дня в отдельном ключе негативно скажется на длительности
// обработки запросов к месяцу. Хранение каждого месяца в отдельном ключе пока что выглядит самым удачным решением.
type Bolt struct {
	db *bbolt.DB
}

func NewBolt(file string) (*Bolt, error) {
	b, err := bbolt.Open(file, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot open bolt store: %w", err)
	}
	log.Printf("[DEBUG] store/bolt opened %s successfully", file)

	return &Bolt{
		db: b,
	}, nil
}

func (b *Bolt) Close() error {
	if err := b.db.Close(); err != nil {
		return fmt.Errorf("cannot close bolt store: %w", err)
	}
	log.Printf("[DEBUG] store/bolt closed successfully")
	return nil
}

func (b *Bolt) FindDay(y int, mon time.Month, d int) (*store.Day, bool) {
	days, ok := b.FindMonth(y, mon)
	if !ok {
		return nil, false
	}

	day, ok := days[d]
	if !ok {
		return nil, false
	}

	return &day, true
}

func (b *Bolt) FindMonth(y int, mon time.Month) (d store.Days, ok bool) {
	_ = b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(calBucket))
		if bucket == nil {
			ok = false
			return nil
		}

		key := fmt.Sprintf("/%d/%d", y, mon)
		daysJson := bucket.Get([]byte(key))
		log.Printf("[DEBUG] store/bolt get key=%s len=%d", key, len(daysJson))

		err := json.Unmarshal(daysJson, &d)
		if err != nil {
			d = nil
			ok = false
			log.Printf("[WARN] bolt: invalid month calendar at /%d/%d: %v", y, mon, err)
			return nil
		}

		ok = true
		return nil
	})
	return
}

func (b *Bolt) FindYear(y int) (m store.Months, ok bool) {
	_ = b.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(calBucket))
		if bucket == nil {
			ok = false
			return nil
		}

		m = make(store.Months, 12)

		prefix := []byte(fmt.Sprintf("/%d/", y))
		log.Printf("[DEBUG] store/bolt getting cursor at %s", prefix)
		c := bucket.Cursor()

		// Ключи в bolt отсортированы по возрастанию.
		// Поэтому можно перейти к первому ключу, который начинается с prefix, затем перебирать ключи,
		// пока не встретится другой префикс, либо не закончится бакет.
		k, v := c.Seek(prefix)
		for k != nil && bytes.HasPrefix(k, prefix) {
			log.Printf("[DEBUG] store/bolt cursor is at key=%s len=%d", k, len(v))

			strMon := string(bytes.TrimPrefix(k, prefix))
			monNum, err := strconv.Atoi(strMon)
			if err != nil {
				log.Printf("[WARN] bolt: invalid month key: %s", k)
				k, v = c.Next()
				continue
			}

			var d store.Days
			if err := json.Unmarshal(v, &d); err != nil {
				log.Printf("[WARN] bolt: invalid month calendar at %s: %v", k, err)
				k, v = c.Next()
				continue
			}

			mon := time.Month(monNum)
			m[mon] = d

			k, v = c.Next()
		}

		ok = len(m) > 0
		return nil
	})
	return
}

func (b *Bolt) PutYear(y int, data store.Months) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(calBucket))
		if err != nil {
			return fmt.Errorf("bolt cannot create bucket '%s': %v", calBucket, err)
		}

		for m, days := range data {
			key := []byte(fmt.Sprintf("/%d/%d", y, m))

			val, err := json.Marshal(days)
			if err != nil {
				return fmt.Errorf("bolt cannot marshal %s: %v", key, err)
			}

			log.Printf("[DEBUG] store/bolt put key=%s len=%d", key, len(val))
			if err := bucket.Put(key, val); err != nil {
				return fmt.Errorf("bolt cannot put %s: %v", key, err)
			}
		}
		return nil
	})
}

func (b *Bolt) Backup(w io.Writer) error {
	return b.db.View(func(tx *bbolt.Tx) error {
		log.Printf("[DEBUG] store/bolt writing backup len=%d", tx.Size())
		_, err := tx.WriteTo(w)
		return err
	})
}
