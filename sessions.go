package users

// func OpenSession(w http.ResponseWriter, r *http.Request) *Session {
// 	var sess *Session
// 	cookie, err := r.Cookie(SessionCookieName)
// 	if err != nil {
// 		if err == http.ErrNoCookie {
// 			if SessionDebug {
// 				log.Println("Creating new session")
// 			}
// 			sess = makeSession(w)
// 		} else {
// 			log.Fatal("OpenSession() error:", err)
// 			return nil
// 		}
// 	} else {
// 		if SessionDebug {
// 			log.Println("Loading session from database")
// 		}
// 		err = db.View(func(tx *bolt.Tx) error {
// 			val := tx.Bucket([]byte("sessions")).Get([]byte(cookie.Value))
// 			// TODO check if session is expired
// 			if val == nil {
// 				if SessionDebug {
// 					log.Println("Expired session - creating new session")
// 				}
// 				log.Println("Not found:    ", cookie.Value[:10])
// 				sess = makeSession(w)
// 			} else {
// 				sess = new(Session)
// 				err = gob.NewDecoder(bytes.NewBuffer(val)).Decode(sess)
// 				if err != nil {
// 					log.Fatal("Could not decode session: ", err)
// 				}
// 			}
// 			return nil
// 		})
// 		if err != nil {
// 			log.Fatal("Transaction error:", err)
// 		}
// 	}

// 	if SessionDebug {
// 		log.Printf("Session open: %#v\n", sess)
// 	}
// 	return sess
// }

// func (s *Session) LogIn(username, name string) {
// 	s.User = username
// 	s.Name = name
// 	s.Bound = true
// 	s.LoggedIn = true
// 	s.updated = true
// }

// func (s *Session) LogOut() {
// 	s.LoggedIn = false
// 	s.updated = true
// }

// func (s *Session) Close() error {
// 	if SessionDebug {
// 		log.Printf("Session close: %#v\n", s)
// 	}
// 	log.Printf("Session close: %s %#v %#v %#v\n", s.ID[:10], s.User, s.Bound, s.LoggedIn)
// 	if s.updated {
// 		if SessionDebug {
// 			log.Println("Saving Session", string(s.ID))
// 		}
// 		var buf bytes.Buffer
// 		err := gob.NewEncoder(&buf).Encode(&s)
// 		if err != nil {
// 			return err
// 		}
// 		err = db.Update(func(tx *bolt.Tx) error {
// 			err = tx.Bucket([]byte("sessions")).Put(s.ID, buf.Bytes())
// 			if err != nil {
// 				return err
// 			}
// 			return nil
// 		})
// 		if err != nil {
// 			log.Fatal("Session Close error:", err)
// 			return err
// 		}
// 	} else {
// 		if SessionDebug {
// 			log.Println("Session did not change", string(s.ID))
// 		}
// 	}

// 	return nil
// }

// func SessionManager() {
// 	for {
// 		count := 0
// 		removed := 0
// 		toRemove := make([][]byte, 0)
// 		db.View(func(tx *bolt.Tx) error {
// 			tx.Bucket([]byte("sessions")).
// 				ForEach(func(key, val []byte) error {
// 				count++
// 				sess := new(Session)
// 				err := gob.NewDecoder(bytes.NewBuffer(val)).Decode(sess)
// 				if err != nil {
// 					log.Fatal("!!! Could not decode session in manager: ", err)
// 				}
// 				if time.Now().Sub(sess.LastCon) > 3*time.Minute && !sess.Bound {
// 					toRemove = append(toRemove, key)
// 					removed++
// 				}
// 				return nil
// 			})
// 			return nil
// 		})
// 		if removed != 0 {
// 			db.Update(func(tx *bolt.Tx) error {
// 				b := tx.Bucket([]byte("sessions"))
// 				for _, k := range toRemove {
// 					err := b.Delete(k)
// 					if err != nil {
// 						log.Println("!!! Could not delete a session:", err)
// 					}
// 				}
// 				return nil
// 			})
// 			log.Println(removed, "unbound Sessions removed,", count-removed, "connected")
// 		}
// 		time.Sleep(3*time.Minute + 3*time.Second)
// 	}
// }

// func ListSessions() {
// 	count := 0
// 	db.View(func(tx *bolt.Tx) error {
// 		tx.Bucket([]byte("sessions")).
// 			ForEach(func(key, val []byte) error {
// 			count++
// 			sess := new(Session)
// 			err := gob.NewDecoder(bytes.NewBuffer(val)).Decode(sess)
// 			if err != nil {
// 				log.Fatal("!!! Could not decode session in list: ", err)
// 			}
// 			// TODO pretty print this
// 			log.Println(sess)
// 			return nil
// 		})
// 		return nil
// 	})
// 	log.Println("Listed", count, "sessions")
// }

// func DELETEallSessions() {
// 	db.Update(func(tx *bolt.Tx) error {
// 		err := tx.DeleteBucket(BucketSessions)
// 		if err != nil {
// 			log.Println("Delete Bucket error", err)
// 		}
// 		_, err = tx.CreateBucket(BucketSessions)
// 		if err != nil {
// 			log.Println("Create Bucket error", err)
// 		}
// 		return nil
// 	})
// }
