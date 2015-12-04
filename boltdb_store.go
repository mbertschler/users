package users

// type BoltDBStore struct {
// 	db *bolt.DB
// }

// func NewBoltDBStore(path string, mode os.FileMode, options *bolt.Options) *BoltDBStore {
// 	var s = BoltDBStore{
// 		users:    make(map[string]User),
// 		sessions: make(map[string]Session),
// 	}
// 	return &s
// }

// type OpenBoltDBStore struct {
// 	db            *bolt.DB
// 	sessionBucket string
// 	userBucket    string
// }

// func NewOpenBoltDBStore(db *bolt.DB, sessionBucket, userBucket string) *OpenBoltDBStore {
// 	var s = OpenBoltDBStore{
// 		users:    make(map[string]User),
// 		sessions: make(map[string]Session),
// 	}
// 	return &s
// }

// func loadDB() {
// 	var err error
// 	db, err := bolt.Open(config.DBPath, 0644, &bolt.Options{Timeout: 1 * time.Second})
// 	if err != nil {
// 		log.Fatal("Could not open DB at", config.DBPath, ": ", err)
// 	}
// 	db.Update(func(tx *bolt.Tx) error {
// 		_, err = tx.CreateBucketIfNotExists(BucketApp)
// 		if err != nil {
// 			log.Fatalln("Bucket app could not be created")
// 		}
// 		_, err = tx.CreateBucketIfNotExists(BucketEvents)
// 		if err != nil {
// 			log.Fatalln("Bucket events could not be created")
// 		}
// 		_, err = tx.CreateBucketIfNotExists(BucketProjects)
// 		if err != nil {
// 			log.Fatalln("Bucket projects could not be created")
// 		}
// 		_, err = tx.CreateBucketIfNotExists(BucketJury)
// 		if err != nil {
// 			log.Fatalln("Bucket jury could not be created")
// 		}
// 		_, err = tx.CreateBucketIfNotExists(BucketSMS)
// 		if err != nil {
// 			log.Fatalln("Bucket sms could not be created")
// 		}
// 		_, err = tx.CreateBucketIfNotExists(BucketSessions)
// 		if err != nil {
// 			log.Fatalln("Bucket sms could not be created")
// 		}
// 		return nil
// 	})
// }

// func B(in string) []byte {
// 	return []byte(in)
// }

// func IDtoBytes(id uint64) []byte {
// 	idbytes := make([]byte, 8)
// 	binary.BigEndian.PutUint64(idbytes, id)
// 	return idbytes
// }

// func BytesToID(b []byte) uint64 {
// 	return binary.BigEndian.Uint64(b)
// }

// func TimeToBytes(t time.Time) []byte {
// 	buf := make([]byte, 8)
// 	binary.BigEndian.PutUint64(buf, uint64(t.UnixNano()))
// 	return buf
// }
// func BytesToTime(b []byte) time.Time {
// 	unixnano := binary.BigEndian.Uint64(b)
// 	return time.Unix(0, int64(unixnano))
// }
