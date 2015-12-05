// Copyright © 2015 Martin Bertschler <mbertschler@gmail.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package users provides user and session management for applications
where users are identified by a session ID or HTTP cookie.

For a basic usage example see the file example/main.go.

This is how you would use the Store methods in net/http HandlerFuncs. (Code
shortened for this example.)

	import "github.com/mbertschler/users"

	var userStore = users.NewMemoryStore()

	func handler(w http.ResponseWriter, r *http.Request) {
		user, err := userStore.Get(w, r)
		if err != nil {
			log.Println(err)
		}
		// use user object and handle errors ...
	}

	func loginHandler(w http.ResponseWriter, r *http.Request) {
		user, err := userStore.Login(w, r,
			r.PostFormValue("user"),
			r.PostFormValue("pass"),
		)
		// use user object and handle errors ...
	}
*/
package users
