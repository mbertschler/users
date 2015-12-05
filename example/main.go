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

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/mbertschler/users"
)

var (
	port      string
	path      string
	userStore stringStore
)

type stringStore struct {
	*users.Store
}

func (s stringStore) GetData(w http.ResponseWriter, r *http.Request) (*users.User, string, error) {
	u, err := s.Get(w, r)
	data, ok := u.Data.(string)
	if !ok {
		data = "&nbsp;"
	}
	return u, data, err
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.StringVar(&port, "port", ":8001", "Port for http server")
	flag.StringVar(&path, "path", "./users.db", "Path for db file")
	flag.Parse()

	userStore = stringStore{users.NewMemoryStore()}

	http.HandleFunc("/", index)
	http.HandleFunc("/login", login)
	http.HandleFunc("/register", register)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/save", save)

	log.Println("Testapp for \"github.com/mbertschler/users\"")
	log.Println("Serving HTTP at " + port)
	log.Println("Saving users DB at " + path)
	log.Println("------------------------------------------")
	log.Fatal(http.ListenAndServe(port, nil))
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.Login(w, r,
		r.PostFormValue("user"),
		r.PostFormValue("pass"),
	)
	if err != nil {
		log.Println("Login error:", err)
		w.Write(errorPage(fmt.Sprintln("Login error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func register(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.Register(w, r,
		r.PostFormValue("user"),
		r.PostFormValue("pass"))
	if err != nil {
		log.Println("Register error:", err)
		w.Write(errorPage(fmt.Sprintln("Register error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	_, err := userStore.Logout(w, r)
	if err != nil {
		log.Println("Logout error:", err)
		w.Write(errorPage(fmt.Sprintln("Logout error:", err)))
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
func save(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}
	user, err := userStore.Get(w, r)
	if err != nil {
		log.Println("Save error 1:", err)
		w.Write(errorPage(fmt.Sprintln("Save error 1:", err)))
		return
	}
	user.Data = r.PostFormValue("val")
	err = userStore.Save(user)
	if err != nil {
		log.Println("Save error 2:", err)
		w.Write(errorPage(fmt.Sprintln("Save error 2:", err)))
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func index(w http.ResponseWriter, r *http.Request) {
	user, data, err := userStore.GetData(w, r)
	if err != nil {
		log.Println("Index error:", err)
		w.Write(errorPage(fmt.Sprintln("Index error:", err)))
		return
	}

	w.Write([]byte(header + `
		<h1>Testapp for package "github.com/mbertschler/users"</h1>
		<table border="1">
			<thead>
					<th>Variable</th>
					<th>Value</th>
			</thead>
			<tbody>
				<tr>
					<td>Session ID</td>
					<td style="word-break: break-all;">` + fmt.Sprint(user.ID) + `</td>
				</tr>
				<tr>
					<td>Session Expires</td>
					<td>` + fmt.Sprint(user.Expires) + `</td>
				</tr>
				<tr>
					<td>Session LastCon</td>
					<td>` + fmt.Sprint(user.LastAccess) + `</td>
				</tr>
				<tr>
					<td>Session LoggedIn</td>
					<td>` + fmt.Sprint(user.LoggedIn) + `</td>
				</tr>
				<tr>
					<td>Session User</td>
					<td>` + fmt.Sprint(user.User) + `</td>
				</tr>
				<tr>
					<td>Username</td>
					<td>` + fmt.Sprint(user.Name) + `</td>
				</tr>
			</tbody>
		</table>
		<div>
			<h2>Register</h2>
			<form action="/register" method="POST">
				<input type="text" name="user" placeholder="Username"/> <br/>
				<input type="password" name="pass" placeholder="Password"/> <br/>
				<button type="submit">Register</button>
			</form>
		</div>
		<div>
			<h2>Login</h2>
			<form action="/login" method="POST">
				<input type="text" name="user" placeholder="Username"/> <br/>
				<input type="password" name="pass" placeholder="Password"/> <br/>
				<button type="submit">Login</button>
			</form>
		</div>
		<div>
			<h2>Logout</h2>
			<form action="/logout" method="POST">
				<button type="submit">Logout</button>
			</form>
		</div>
		<div>
			<h2>Set Value</h2>
			<p>` + data + `</p>
			<form action="/save" method="POST">
				<input type="text" name="val" placeholder="Value"/> <br/>
				<button type="submit">Save</button>
			</form>
		</div>` + footer))
}

func errorPage(in string) []byte {
	return []byte(header + `
		<h1>Testapp for package "github.com/mbertschler/users"</h1>
		<h2>Error</h2>
		<p>` + in + `</p>
		<a href="/"><button type="submit">Back</button></a>
	` + footer)
}

var header = `
<html>
	<head>
		<style>
			body{
				font-family: sans-serif;
				font-size:16px;
				color: #333;
			}
			td {
			    padding: 4px;
			}
			th {
			    padding: 6px;
			    font-size:19px;
			    background-color:#ddd;
			}
			input, button {
				font-size:16px;
				padding:3px;
				margin-top:6px;
			}
			div {
				display: inline-block;
				width: 200px;
				vertical-align:top;
			}
		</style>
	</head>
	<body>`

var footer = `
	</body>
</html>
`
